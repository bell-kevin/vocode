package symbols

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/symbols/tags"
)

// TreeSitterResolver resolves symbols via the `tree-sitter tags` CLI.
type TreeSitterResolver struct {
	binaryPath string
}

func NewTreeSitterResolver() *TreeSitterResolver {
	// Path is set by the VS Code extension from the provisioned tools/tree-sitter layout.
	if p := strings.TrimSpace(os.Getenv("VOCODE_TREE_SITTER_BIN")); p != "" {
		return &TreeSitterResolver{binaryPath: p}
	}
	return &TreeSitterResolver{}
}

func (r *TreeSitterResolver) ResolveSymbol(workspaceRoot, symbolName, symbolKind, hintPath string) ([]SymbolRef, error) {
	if strings.TrimSpace(r.binaryPath) == "" {
		return nil, errors.New("tree-sitter CLI not configured (run `pnpm provision:tree-sitter` from the repo; extension spawns daemon with the provisioned binary path)")
	}

	root := strings.TrimSpace(workspaceRoot)
	name := strings.TrimSpace(symbolName)
	if root == "" || name == "" {
		return nil, nil
	}

	kindFilter := tags.NormalizeKind(symbolKind)
	candidates, err := candidateFiles(root, name, hintPath)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	out := make([]SymbolRef, 0, 8)
	seen := map[string]bool{}
	for _, file := range candidates {
		tagList, err := tags.LoadTags(r.binaryPath, file)
		if err != nil {
			continue
		}
		for _, ref := range tagList {
			if !ref.IsDefinition {
				continue
			}
			if !strings.EqualFold(ref.Name, name) {
				continue
			}
			if kindFilter != "" && ref.Kind != kindFilter {
				continue
			}
			key := ref.Path + ":" + ref.Kind + ":" + ref.Name
			if seen[key] {
				continue
			}
			seen[key] = true
			match := tagToSymbolRef(ref)
			out = append(out, match)
		}
	}
	return out, nil
}

func candidateFiles(workspaceRoot, symbolName, hintPath string) ([]string, error) {
	files := make([]string, 0, 8)
	seen := map[string]bool{}

	if hint := strings.TrimSpace(hintPath); hint != "" {
		hint = filepath.Clean(hint)
		seen[hint] = true
		files = append(files, hint)
	}

	cmd := exec.Command(
		"rg",
		"--files-with-matches",
		"--glob",
		"*.{go,ts,tsx,js,jsx,py,java,rs,c,cc,cpp,cxx,h,hpp,cs,kt,kts,swift,rb,php,lua}",
		`\b`+regexp.QuoteMeta(symbolName)+`\b`,
		workspaceRoot,
	)
	var stdout strings.Builder
	var stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("candidate file search failed: %v (%s)", err, strings.TrimSpace(stderr.String()))
	}

	for _, line := range strings.Split(stdout.String(), "\n") {
		p := filepath.Clean(strings.TrimSpace(line))
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		files = append(files, p)
		if len(files) >= 50 {
			break
		}
	}
	return files, nil
}

func tagToSymbolRef(t tags.Tag) SymbolRef {
	line := t.Line
	if line < 1 && t.HasSpan() {
		line = t.StartLine + 1
	}
	ref := SymbolRef{
		Name: t.Name,
		Path: t.Path,
		Line: line,
		Kind: t.Kind,
	}
	ref.ID = BuildSymbolID(ref)
	return ref
}
