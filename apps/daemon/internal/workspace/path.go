package workspace

import (
	"path/filepath"
	"strings"
)

// ResolveTargetPath interprets a path from the model or agent relative to workspace root.
// Empty target means the active file. Relative targets require a non-empty workspace root.
func ResolveTargetPath(workspaceRoot, activeFile, target string) string {
	t := strings.TrimSpace(target)
	if t == "" {
		return filepath.Clean(activeFile)
	}
	if filepath.IsAbs(t) || strings.HasPrefix(t, "/") || strings.HasPrefix(t, "\\") {
		return filepath.Clean(t)
	}
	root := strings.TrimSpace(workspaceRoot)
	if root == "" {
		return ""
	}
	return filepath.Clean(filepath.Join(root, t))
}
