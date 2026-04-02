package run

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

const maxFileSelectionList = 1500

func jailUnderWorkspaceRoot(root, p string) (clean string, ok bool) {
	root = filepath.Clean(root)
	p = filepath.Clean(p)
	if root == "" || p == "" {
		return "", false
	}
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return "", false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return p, true
}

func listWorkspaceFiles(root string) ([]string, error) {
	skipName := map[string]bool{
		".git": true, "node_modules": true, "dist": true, "target": true,
		"__pycache__": true, ".svn": true, ".hg": true,
	}
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipName[filepath.Base(path)] {
				return fs.SkipDir
			}
			return nil
		}
		out = append(out, filepath.Clean(path))
		if len(out) >= maxFileSelectionList {
			return fs.SkipAll
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}

func ensureFileSelectionPaths(vs *agentcontext.VoiceSession, root string) error {
	if len(vs.FileSelectionPaths) > 0 {
		return nil
	}
	paths, err := listWorkspaceFiles(root)
	if err != nil {
		return err
	}
	vs.FileSelectionPaths = paths
	return nil
}

func resolveFileSelectionFocus(params protocol.VoiceTranscriptParams, vs *agentcontext.VoiceSession, root string) string {
	candidates := []string{
		strings.TrimSpace(params.FocusedWorkspacePath),
		strings.TrimSpace(vs.FileSelectionFocus),
		strings.TrimSpace(params.ActiveFile),
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if j, ok := jailUnderWorkspaceRoot(root, c); ok {
			return j
		}
	}
	return ""
}

func indexOfPath(paths []string, p string) int {
	for i, q := range paths {
		if strings.EqualFold(q, p) {
			return i
		}
	}
	return -1
}
