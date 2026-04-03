package searchapply

import (
	"fmt"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/search"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	"vocoding.net/vocode/v2/apps/core/internal/workspace"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// FileSearchFromQuery runs fixed-string search, collapses to unique paths, updates optional session
// file-list fields, and returns a completion with FileSelection plus host open of the first path.
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

	hits, err := search.FixedStringSearch(root, q, fileSearchCollectMaxHits)
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, true, "file search failed: " + err.Error()
	}

	paths := search.UniqueSortedPaths(hits, fileSearchMaxUniquePaths)
	mutateSessionFilePathSearchResults(vs, paths)

	if len(paths) == 0 {
		return completionFileSearchNoHits(q), true, ""
	}

	if e.HostApply == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, true, "daemon has directives but no host apply client is configured"
	}
	if e.NewBatchID == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, true, "search engine not fully configured"
	}
	first := paths[0]
	dirs := openFirstFileDirectives(first)
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

	return completionFileSearchWithPaths(paths, q), true, ""
}

func mutateSessionFilePathSearchResults(vs *session.VoiceSession, paths []string) {
	if vs == nil {
		return
	}
	vs.SearchResults = nil
	vs.ActiveSearchIndex = 0
	vs.PendingDirectiveApply = nil
	vs.FileSelectionPaths = paths
	if len(paths) > 0 {
		vs.FileSelectionIndex = 0
		vs.FileSelectionFocus = paths[0]
	} else {
		vs.FileSelectionIndex = 0
		vs.FileSelectionFocus = ""
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

func completionFileSearchWithPaths(paths []string, q string) protocol.VoiceTranscriptCompletion {
	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       fmt.Sprintf("found %d path(s) for %q", len(paths), q),
		UiDisposition: "hidden",
		FileSelection: FileSearchStateFromPaths(paths, 0),
	}
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
