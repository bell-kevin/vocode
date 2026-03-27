package symbols

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// TreeSitterResolver resolves symbols via the `tree-sitter tags` CLI.
type TreeSitterResolver struct {
	hasCLI bool
}

func NewTreeSitterResolver() *TreeSitterResolver {
	_, err := exec.LookPath("tree-sitter")
	return &TreeSitterResolver{hasCLI: err == nil}
}

func (r *TreeSitterResolver) ResolveSymbol(workspaceRoot, symbolName, symbolKind, hintPath string) ([]SymbolRef, error) {
	if !r.hasCLI {
		return nil, errors.New("tree-sitter CLI not found in PATH")
	}

	root := strings.TrimSpace(workspaceRoot)
	name := strings.TrimSpace(symbolName)
	if root == "" || name == "" {
		return nil, nil
	}

	kindFilter := normalizeKind(symbolKind)
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
		refs, err := tagsForFile(file)
		if err != nil {
			continue
		}
		for _, ref := range refs {
			if !strings.EqualFold(ref.Name, name) {
				continue
			}
			if kindFilter != "" && normalizeKind(ref.Kind) != kindFilter {
				continue
			}
			key := ref.Path + ":" + ref.Kind + ":" + ref.Name
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, SymbolRef{
				Path: ref.Path,
				Line: ref.Line,
				Kind: normalizeKind(ref.Kind),
			})
		}
	}
	return out, nil
}

type tagRef struct {
	Name string
	Path string
	Kind string
	Line int
}

func candidateFiles(workspaceRoot, symbolName, hintPath string) ([]string, error) {
	files := make([]string, 0, 8)
	seen := map[string]bool{}

	if hint := strings.TrimSpace(hintPath); hint != "" {
		hint = filepath.Clean(hint)
		seen[hint] = true
		files = append(files, hint)
	}

	// Gather candidate files across common source-language extensions.
	cmd := exec.Command(
		"rg",
		"--files-with-matches",
		"--glob",
		"*.{go,ts,tsx,js,jsx,py,java,rs,c,cc,cpp,cxx,h,hpp,cs,kt,kts,swift,rb,php,lua}",
		`\b`+regexp.QuoteMeta(symbolName)+`\b`,
		workspaceRoot,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("candidate file search failed: %v (%s)", err, strings.TrimSpace(stderr.String()))
	}

	sc := bufio.NewScanner(bytes.NewReader(stdout.Bytes()))
	for sc.Scan() {
		p := filepath.Clean(strings.TrimSpace(sc.Text()))
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		files = append(files, p)
		// Keep runtime predictable for now.
		if len(files) >= 50 {
			break
		}
	}
	return files, nil
}

func tagsForFile(path string) ([]tagRef, error) {
	cmd := exec.Command("tree-sitter", "tags", path)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("tree-sitter tags failed for %q: %v (%s)", path, err, strings.TrimSpace(stderr.String()))
	}

	out := make([]tagRef, 0, 16)
	sc := bufio.NewScanner(bytes.NewReader(stdout.Bytes()))
	for sc.Scan() {
		ref, ok := parseTagLine(sc.Text())
		if !ok {
			continue
		}
		if ref.Path == "" {
			ref.Path = path
		}
		ref.Path = filepath.Clean(ref.Path)
		out = append(out, ref)
	}
	return out, nil
}

// parseTagLine parses ctags-like lines:
// name \t path \t /^...$/;" \t kind [ \t line:123 ] [ \t kind:function ]
func parseTagLine(line string) (tagRef, bool) {
	parts := strings.Split(line, "\t")
	if len(parts) < 4 {
		return tagRef{}, false
	}
	ref := tagRef{
		Name: strings.TrimSpace(parts[0]),
		Path: strings.TrimSpace(parts[1]),
		Kind: normalizeKind(strings.TrimSpace(parts[3])),
	}
	if ref.Name == "" {
		return tagRef{}, false
	}
	for _, p := range parts[4:] {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "line:") {
			_, _ = fmt.Sscanf(strings.TrimPrefix(p, "line:"), "%d", &ref.Line)
			continue
		}
		if strings.HasPrefix(p, "kind:") {
			ref.Kind = normalizeKind(strings.TrimPrefix(p, "kind:"))
		}
	}
	return ref, true
}

func normalizeKind(kind string) string {
	k := strings.ToLower(strings.TrimSpace(kind))
	k = strings.TrimPrefix(k, "kind:")
	switch k {
	case "f", "func", "function", "function_definition", "function_declaration", "constructor":
		return "function"
	case "m", "method", "member_function", "member":
		return "method"
	case "c", "class", "class_declaration", "struct":
		return "class"
	case "i", "interface", "trait", "protocol":
		return "interface"
	case "e", "enum":
		return "enum"
	case "t", "type", "type_alias", "typedef":
		return "type"
	case "v", "var", "variable", "field", "property", "member_variable":
		return "variable"
	default:
		return k
	}
}
