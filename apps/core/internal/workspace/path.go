package workspace

import (
	"path/filepath"
	"strings"
)

// EffectiveWorkspaceRoot mirrors the daemon’s behavior at a basic level:
// - if a workspaceRoot is provided, use it
// - otherwise fall back to the parent directory of the active file
func EffectiveWorkspaceRoot(workspaceRoot, activeFile string) string {
	root := strings.TrimSpace(workspaceRoot)
	if root != "" {
		return filepath.Clean(root)
	}
	active := strings.TrimSpace(activeFile)
	if active == "" {
		return ""
	}
	return filepath.Dir(filepath.Clean(active))
}

// PathSearchWorkspaceRoot is the folder to walk for path-basename file search. The host sends
// pathSearchWorkspaceRoot as the outermost workspace folder that contains the active file when
// VS Code multi-root lists nested folders; otherwise [EffectiveWorkspaceRoot] is used.
func PathSearchWorkspaceRoot(workspaceRoot, pathSearchRoot, activeFile string) string {
	if r := strings.TrimSpace(pathSearchRoot); r != "" {
		return filepath.Clean(r)
	}
	return EffectiveWorkspaceRoot(workspaceRoot, activeFile)
}

