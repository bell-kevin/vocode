package requestcontext

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	"vocoding.net/vocode/v2/apps/daemon/internal/symbols"
	"vocoding.net/vocode/v2/apps/daemon/internal/workspace"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Provider fulfills request_context intents by enriching [agentcontext.Gathered]
// (symbols, file excerpts, usage notes). Used by the transcript executor, not the
// directive pipeline (no protocol directive is emitted).
type Provider struct {
	symbols symbols.Resolver
}

func NewProvider(symbolResolver symbols.Resolver) *Provider {
	return &Provider{symbols: symbolResolver}
}

func Dispatch(
	p *Provider,
	params protocol.VoiceTranscriptParams,
	in agentcontext.Gathered,
	req *intents.RequestContextIntent,
) (agentcontext.Gathered, error) {
	if p == nil {
		return in, fmt.Errorf("request_context: provider not configured")
	}
	if req == nil {
		return in, fmt.Errorf("request_context missing payload")
	}
	out := in
	switch req.Kind {
	case intents.RequestContextKindSymbols:
		query := strings.TrimSpace(req.Query)
		if query == "" {
			return out, fmt.Errorf("request_symbols requires query")
		}
		if p.symbols == nil {
			return out, fmt.Errorf("symbol resolver unavailable")
		}
		root := workspace.EffectiveWorkspaceRoot(params.WorkspaceRoot, params.ActiveFile)
		matches, err := p.symbols.ResolveSymbol(root, query, "", strings.TrimSpace(params.ActiveFile))
		if err != nil {
			return out, err
		}
		limit := clampContextMax(req.MaxResult, 10)
		if out.Symbols == nil {
			out.Symbols = make([]symbols.SymbolRef, 0, limit)
		}
		seen := map[string]bool{}
		for _, sref := range out.Symbols {
			seen[sref.ID] = true
		}
		for _, m := range matches {
			if m.ID == "" || seen[m.ID] {
				continue
			}
			seen[m.ID] = true
			out.Symbols = append(out.Symbols, m)
			if len(out.Symbols) >= limit {
				break
			}
		}
		return out, nil
	case intents.RequestContextKindFileExcerpt:
		target := strings.TrimSpace(req.Path)
		path := workspace.ResolveTargetPath(params.WorkspaceRoot, params.ActiveFile, target)
		if path == "" {
			return out, fmt.Errorf("request_file_excerpt requires resolvable path")
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return out, err
		}
		content := string(b)
		const maxChars = 4000
		if len(content) > maxChars {
			content = content[:maxChars]
		}
		out = agentcontext.UpsertGatheredExcerpt(out, filepath.Clean(path), content)
		return out, nil
	case intents.RequestContextKindUsages:
		ref, err := symbols.ParseSymbolID(strings.TrimSpace(req.SymbolID))
		if err != nil {
			return out, fmt.Errorf("request_usages requires valid symbolId: %w", err)
		}
		limit := clampContextMax(req.MaxResult, 10)
		pattern := `\b` + regexp.QuoteMeta(strings.TrimSpace(ref.Name)) + `\b`
		searchRoot := workspace.EffectiveWorkspaceRoot(params.WorkspaceRoot, params.ActiveFile)
		if searchRoot == "" {
			return out, nil
		}
		cmd := exec.Command("rg", "-n", pattern, searchRoot)
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil && stdout.Len() == 0 {
			return out, nil
		}
		sc := bufio.NewScanner(bytes.NewReader(stdout.Bytes()))
		count := 0
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			out.Notes = append(out.Notes, "usage: "+line)
			count++
			if count >= limit {
				break
			}
		}
		return out, nil
	default:
		return out, fmt.Errorf("unsupported request_context kind %q", req.Kind)
	}
}

func clampContextMax(v int, def int) int {
	if v <= 0 {
		return def
	}
	if v > 50 {
		return 50
	}
	return v
}
