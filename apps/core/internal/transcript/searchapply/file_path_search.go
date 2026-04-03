package searchapply

import (
	"fmt"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/search"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	"vocoding.net/vocode/v2/apps/core/internal/workspace"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// FileSearchFromQuery walks the workspace for paths whose relative path or basename contains the
// query fragment (case-insensitive), updates optional session file-list fields, and returns a
// completion with FileSelection plus host open of the first path when it is a file.
func (e *TranscriptSearch) FileSearchFromQuery(params protocol.VoiceTranscriptParams, q string, vs *session.VoiceSession) (protocol.VoiceTranscriptCompletion, bool, string) {
	q = strings.TrimSpace(q)
	if q == "" {
		return protocol.VoiceTranscriptCompletion{}, false, ""
	}
	root := strings.TrimSpace(workspace.EffectiveWorkspaceRoot(params.WorkspaceRoot, params.ActiveFile))
	if root == "" {
		return protocol.VoiceTranscriptCompletion{
			Success: false,
			Summary: "",
		}, true, "search requires workspaceRoot or activeFile"
	}

	matches, err := search.PathFragmentMatches(root, q, fileSearchMaxUniquePaths)
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, true, "file search failed: " + err.Error()
	}
	paths := make([]string, len(matches))
	isDir := make([]bool, len(matches))
	for i, m := range matches {
		paths[i] = m.Path
		isDir[i] = m.IsDir
	}
	mutateSessionFilePathSearchResults(vs, paths, isDir)

	if len(paths) == 0 {
		return completionFileSearchNoHits(q), true, ""
	}

	if e.HostApply == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, true, "host apply client not configured"
	}
	if e.NewBatchID == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, true, "search engine not fully configured"
	}
	first := matches[0]
	if !first.IsDir {
		dirs := openFirstFileDirectives(first.Path)
		batchID := e.NewBatchID()
		if vs != nil {
			vs.PendingDirectiveApply = &session.DirectiveApplyBatch{ID: batchID, NumDirectives: len(dirs)}
		}
		hostRes, err := e.HostApply.ApplyDirectives(protocol.HostApplyParams{
			ApplyBatchId: batchID,
			ActiveFile:   params.ActiveFile,
			Directives:   dirs,
		})
		if err != nil {
			if vs != nil {
				vs.PendingDirectiveApply = nil
			}
			return protocol.VoiceTranscriptCompletion{Success: false}, true, "host.applyDirectives failed: " + err.Error()
		}
		if vs != nil && vs.PendingDirectiveApply != nil {
			if err := vs.PendingDirectiveApply.ConsumeHostApplyReport(batchID, hostRes.Items); err != nil {
				vs.PendingDirectiveApply = nil
				return protocol.VoiceTranscriptCompletion{Success: false}, true, "host apply failed: " + err.Error()
			}
			vs.PendingDirectiveApply = nil
		}
	}

	return completionFileSearchWithPaths(paths, isDir, q), true, ""
}

func mutateSessionFilePathSearchResults(vs *session.VoiceSession, paths []string, isDir []bool) {
	if vs == nil {
		return
	}
	vs.SearchResults = nil
	vs.ActiveSearchIndex = 0
	vs.PendingDirectiveApply = nil
	vs.FileSelectionPaths = paths
	vs.FileSelectionIsDir = isDir
	if len(paths) > 0 {
		vs.FileSelectionIndex = 0
		vs.FileSelectionFocus = paths[0]
	} else {
		vs.FileSelectionIndex = 0
		vs.FileSelectionFocus = ""
		vs.FileSelectionIsDir = nil
	}
}

func completionFileSearchNoHits(q string) protocol.VoiceTranscriptCompletion {
	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       fmt.Sprintf("no file path matches for %q", q),
		UiDisposition: "hidden",
		FileSelection: &protocol.VoiceTranscriptFileSearchState{
			NoHits: true,
		},
	}
}

func completionFileSearchWithPaths(paths []string, isDir []bool, q string) protocol.VoiceTranscriptCompletion {
	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       fmt.Sprintf("found %d path(s) for %q", len(paths), q),
		UiDisposition: "hidden",
		FileSelection: FileSearchStateFromPathsWithDir(paths, isDir, 0),
	}
}

// OpenFirstFileDirectivesForPath returns a single open_file navigation directive (exported for file-select control).
func OpenFirstFileDirectivesForPath(path string) []protocol.VoiceTranscriptDirective {
	return openFirstFileDirectives(path)
}

func openFirstFileDirectives(path string) []protocol.VoiceTranscriptDirective {
	return []protocol.VoiceTranscriptDirective{
		{
			Kind: "navigate",
			NavigationDirective: &protocol.NavigationDirective{
				Kind: "success",
				Action: &protocol.NavigationAction{
					Kind: "open_file",
					OpenFile: &struct {
						Path string `json:"path"`
					}{Path: path},
				},
			},
		},
	}
}
