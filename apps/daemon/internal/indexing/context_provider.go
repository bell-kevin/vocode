package indexing

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/edits"
	"vocoding.net/vocode/v2/apps/daemon/internal/intent"
	"vocoding.net/vocode/v2/apps/daemon/internal/symbols"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type ContextProvider struct {
	symbols symbols.Resolver
}

func NewContextProvider(symbolResolver symbols.Resolver) *ContextProvider {
	return &ContextProvider{symbols: symbolResolver}
}

func (p *ContextProvider) Fulfill(
	params protocol.VoiceTranscriptParams,
	in agent.PlanningContext,
	req *intent.RequestContextIntent,
) (agent.PlanningContext, error) {
	if req == nil {
		return in, fmt.Errorf("request_context missing payload")
	}
	out := in
	switch req.Kind {
	case intent.RequestContextKindSymbols:
		query := strings.TrimSpace(req.Query)
		if query == "" {
			return out, fmt.Errorf("request_symbols requires query")
		}
		if p.symbols == nil {
			return out, fmt.Errorf("symbol resolver unavailable")
		}
		matches, err := p.symbols.ResolveSymbol(strings.TrimSpace(params.WorkspaceRoot), query, "", strings.TrimSpace(params.ActiveFile))
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
	case intent.RequestContextKindFileExcerpt:
		target := strings.TrimSpace(req.Path)
		ec := edits.EditExecutionContext{ActiveFile: params.ActiveFile, WorkspaceRoot: params.WorkspaceRoot}
		path := ec.ResolvePath(target)
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
		out.Excerpts = append(out.Excerpts, agent.FileExcerpt{Path: filepath.Clean(path), Content: content})
		return out, nil
	case intent.RequestContextKindUsages:
		ref, err := symbols.ParseSymbolID(strings.TrimSpace(req.SymbolID))
		if err != nil {
			return out, fmt.Errorf("request_usages requires valid symbolId: %w", err)
		}
		limit := clampContextMax(req.MaxResult, 10)
		pattern := `\b` + regexp.QuoteMeta(strings.TrimSpace(ref.Name)) + `\b`
		cmd := exec.Command("rg", "-n", pattern, strings.TrimSpace(params.WorkspaceRoot))
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
