package edits

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EditExecutionContext is the daemon-side context used to compile edit intents
// into concrete edit actions.
type EditExecutionContext struct {
	Instruction string
	ActiveFile  string
	FileText    string

	// Files is an optional prefetched file-text map keyed by path.
	// It can be populated by a context gatherer (ripgrep/tree-sitter pipeline)
	// to avoid repeated disk reads and keep planning/execution deterministic.
	Files map[string]string
}

func (c EditExecutionContext) ResolvePath(targetPath string) string {
	target := strings.TrimSpace(targetPath)
	if target == "" {
		return filepath.Clean(c.ActiveFile)
	}
	if filepath.IsAbs(target) || strings.HasPrefix(target, "/") || strings.HasPrefix(target, "\\") {
		return filepath.Clean(target)
	}
	activeRelative := filepath.Clean(filepath.Join(filepath.Dir(c.ActiveFile), target))
	if pathExists(activeRelative) {
		return activeRelative
	}
	if wd, err := os.Getwd(); err == nil && wd != "" {
		workspaceRelative := filepath.Clean(filepath.Join(wd, target))
		if pathExists(workspaceRelative) {
			return workspaceRelative
		}
	}
	return activeRelative
}

func (c EditExecutionContext) GetFileText(path string) (string, error) {
	resolved := c.ResolvePath(path)
	if resolved == "" {
		return "", fmt.Errorf("no file path available")
	}

	if samePath(resolved, c.ActiveFile) {
		return c.FileText, nil
	}
	if c.Files != nil {
		if text, ok := c.Files[resolved]; ok {
			return text, nil
		}
	}
	b, err := os.ReadFile(resolved)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func pathExists(p string) bool {
	if strings.TrimSpace(p) == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}
