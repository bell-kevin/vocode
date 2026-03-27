package symbols

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type SymbolRef struct {
	Path string
	Line int
	Kind string
}

type Resolver interface {
	ResolveSymbol(workspaceRoot, symbolName, symbolKind, hintPath string) ([]SymbolRef, error)
}

type RipgrepResolver struct{}

func NewRipgrepResolver() *RipgrepResolver {
	return &RipgrepResolver{}
}

func (r *RipgrepResolver) ResolveSymbol(workspaceRoot, symbolName, symbolKind, hintPath string) ([]SymbolRef, error) {
	root := strings.TrimSpace(workspaceRoot)
	name := strings.TrimSpace(symbolName)
	if root == "" || name == "" {
		return nil, nil
	}
	kind := strings.ToLower(strings.TrimSpace(symbolKind))

	// Language-neutral lexical baseline: exact symbol token match.
	// This intentionally avoids language-specific syntax heuristics.
	pattern := `\b` + regexp.QuoteMeta(name) + `\b`

	cmd := exec.Command("rg", "--json", "-n", "--glob", "*.{go,ts,tsx,js,jsx}", pattern, root)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil && stdout.Len() == 0 {
		// No matches is not fatal; rg exits 1 in that case.
		return nil, nil
	}

	out := make([]SymbolRef, 0, 8)
	seen := map[string]bool{}
	scanner := bufio.NewScanner(bytes.NewReader(stdout.Bytes()))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		match, ok := parseRGJSONMatchLine(line)
		if !ok {
			continue
		}
		p := filepath.Clean(match.Path)
		key := fmt.Sprintf("%s:%d", p, match.Line)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, SymbolRef{Path: p, Line: match.Line, Kind: kind})
	}

	// If a hint path exists, bias to that file first.
	if hint := strings.TrimSpace(hintPath); hint != "" {
		hint = filepath.Clean(hint)
		biased := make([]SymbolRef, 0, len(out))
		for _, m := range out {
			if samePath(m.Path, hint) {
				biased = append(biased, m)
			}
		}
		if len(biased) > 0 {
			return biased, nil
		}
	}
	return out, nil
}

type rgJSONLine struct {
	Type string `json:"type"`
	Data struct {
		Path struct {
			Text string `json:"text"`
		} `json:"path"`
		LineNumber int `json:"line_number"`
	} `json:"data"`
}

func parseRGJSONMatchLine(line string) (SymbolRef, bool) {
	var msg rgJSONLine
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return SymbolRef{}, false
	}
	if msg.Type != "match" {
		return SymbolRef{}, false
	}
	if strings.TrimSpace(msg.Data.Path.Text) == "" || msg.Data.LineNumber <= 0 {
		return SymbolRef{}, false
	}
	return SymbolRef{
		Path: msg.Data.Path.Text,
		Line: msg.Data.LineNumber,
	}, true
}

func samePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
