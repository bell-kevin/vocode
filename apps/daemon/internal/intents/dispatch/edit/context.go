package edit

import (
	"fmt"
	"os"

	"vocoding.net/vocode/v2/apps/daemon/internal/workspace"
)

// EditExecutionContext is the daemon-side context used to compile edit intents
// into concrete edit actions.
type EditExecutionContext struct {
	Instruction   string
	ActiveFile    string
	FileText      string
	WorkspaceRoot string

	// Files is an optional prefetched file-text map keyed by path.
	// It can be populated by a context gatherer (ripgrep/tree-sitter pipeline)
	// to avoid repeated disk reads and keep intent handling deterministic.
	Files map[string]string
}

func (c EditExecutionContext) ResolvePath(targetPath string) string {
	return workspace.ResolveTargetPath(c.WorkspaceRoot, c.ActiveFile, targetPath)
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
