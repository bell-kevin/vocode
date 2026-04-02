package executor

import (
	"fmt"
	"path/filepath"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func resolveScopedTarget(params protocol.VoiceTranscriptParams, fileText string, scope agent.ScopeIntentResult) (agentcontext.ResolvedTarget, string, error) {
	active := filepath.Clean(strings.TrimSpace(params.ActiveFile))
	if active == "" {
		return agentcontext.ResolvedTarget{}, "", fmt.Errorf("activeFile is required")
	}
	lines := strings.Split(fileText, "\n")
	lastLine := len(lines) - 1
	startLine, endLine := 0, lastLine

	if scope.ScopeKind == agent.ScopeCurrentFile {
		startLine, endLine = 0, lastLine
	} else if scope.ScopeKind == agent.ScopeNamedSymbol && len(params.ActiveFileSymbols) > 0 {
		want := strings.ToLower(strings.TrimSpace(scope.SymbolName))
		bestIdx := -1
		bestSize := 0
		for i := range params.ActiveFileSymbols {
			s := params.ActiveFileSymbols[i]
			if strings.ToLower(strings.TrimSpace(s.Name)) != want {
				continue
			}
			r := s.Range
			sz := (int(r.EndLine)-int(r.StartLine))*100000 + (int(r.EndChar) - int(r.StartChar))
			if bestIdx == -1 || sz < bestSize {
				bestIdx = i
				bestSize = sz
			}
		}
		if bestIdx != -1 {
			r := params.ActiveFileSymbols[bestIdx].Range
			startLine = int(r.StartLine)
			endLine = int(r.EndLine)
		}
	} else if scope.ScopeKind == agent.ScopeCurrentFunction && len(params.ActiveFileSymbols) > 0 {
		if cp := params.CursorPosition; cp != nil {
			line := int(cp.Line)
			char := int(cp.Character)
			bestIdx := -1
			bestSize := 0
			for i := range params.ActiveFileSymbols {
				s := params.ActiveFileSymbols[i]
				r := s.Range
				if line < int(r.StartLine) || line > int(r.EndLine) {
					continue
				}
				if line == int(r.StartLine) && char < int(r.StartChar) {
					continue
				}
				if line == int(r.EndLine) && char > int(r.EndChar) {
					continue
				}
				sz := (int(r.EndLine)-int(r.StartLine))*100000 + (int(r.EndChar) - int(r.StartChar))
				if bestIdx == -1 || sz < bestSize {
					bestIdx = i
					bestSize = sz
				}
			}
			if bestIdx != -1 {
				r := params.ActiveFileSymbols[bestIdx].Range
				startLine = int(r.StartLine)
				endLine = int(r.EndLine)
			}
		}
	}
	if startLine < 0 {
		startLine = 0
	}
	if endLine < startLine {
		endLine = startLine
	}
	if endLine > lastLine {
		endLine = lastLine
	}

	text := ""
	if lastLine >= 0 && startLine <= endLine {
		text = strings.Join(lines[startLine:endLine+1], "\n")
	}
	endChar := 0
	if endLine >= 0 && endLine < len(lines) {
		endChar = len(lines[endLine])
	}
	t := agentcontext.ResolvedTarget{
		Path: filepath.Clean(active),
		Range: agentcontext.Range{
			StartLine: startLine,
			StartChar: 0,
			EndLine:   endLine,
			EndChar:   endChar,
		},
	}
	return t, text, nil
}
