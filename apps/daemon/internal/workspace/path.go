package workspace

import (
	"path/filepath"
	"strings"
)

// EffectiveWorkspaceRoot chooses a directory for resolving relative paths and workspace-scoped tools.
// When the editor has no workspace folder open (single loose file), host params may omit workspaceRoot;
// in that case the active file's directory is used so relative paths still resolve.
func EffectiveWorkspaceRoot(workspaceRoot, activeFile string) string {
	r := strings.TrimSpace(workspaceRoot)
	if r != "" {
		return r
	}
	a := strings.TrimSpace(activeFile)
	if a == "" {
		return ""
	}
	return filepath.Dir(filepath.Clean(a))
}

// ResolveTargetPath interprets a path from the model or agent relative to workspace root.
// Empty target means the active file. Relative targets join [EffectiveWorkspaceRoot] with activeFile.
func ResolveTargetPath(workspaceRoot, activeFile, target string) string {
	t := strings.TrimSpace(target)
	if t == "" {
		a := strings.TrimSpace(activeFile)
		if a == "" {
			return ""
		}
		return filepath.Clean(a)
	}
	if filepath.IsAbs(t) || strings.HasPrefix(t, "/") || strings.HasPrefix(t, "\\") {
		return filepath.Clean(t)
	}
	root := EffectiveWorkspaceRoot(workspaceRoot, activeFile)
	if root == "" {
		return ""
	}
	return filepath.Clean(filepath.Join(root, t))
}
