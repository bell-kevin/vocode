package run

import (
	"fmt"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/search"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	"vocoding.net/vocode/v2/apps/core/internal/workspace"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

const (
	contentSearchMaxHits     = 20
	fileSearchCollectMaxHits = 120
	fileSearchMaxUniquePaths = 20
)

// TranscriptSearch runs workspace rg queries and wires results into voice session + host navigation.
type TranscriptSearch struct {
	HostApply             hostApplyClient
	NewBatchID            func() string
	NavigateHitDirectives func(path string, line0, char0, length int) []protocol.VoiceTranscriptDirective
}

// SearchFromQuery runs content search and navigates to the first hit when configured.
func (e *TranscriptSearch) SearchFromQuery(params protocol.VoiceTranscriptParams, q string, vs *session.VoiceSession) (protocol.VoiceTranscriptCompletion, bool, string) {
	q = strings.TrimSpace(q)
	if q == "" {
		return protocol.VoiceTranscriptCompletion{}, false, ""
	}
	root := strings.TrimSpace(workspace.EffectiveWorkspaceRoot(params.WorkspaceRoot, params.ActiveFile))
	if root == "" {
		return protocol.VoiceTranscriptCompletion{
			Success:           false,
			Summary:           "",
			TranscriptOutcome: "",
		}, true, "search requires workspaceRoot or activeFile"
	}

	hits, err := search.FixedStringSearch(root, q, contentSearchMaxHits)
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, true, "search failed: " + err.Error()
	}

	wireHits := hitsToProtocolSearchResults(hits)
	if len(hits) == 0 {
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           fmt.Sprintf("no matches for %q", q),
			TranscriptOutcome: "selection",
			UiDisposition:     "hidden",
			SearchResults:     wireHits,
			ActiveSearchIndex: nil,
		}, true, ""
	}

	var z int64 = 0
	res := protocol.VoiceTranscriptCompletion{
		Success:           true,
		Summary:           fmt.Sprintf("found %d matches for %q", len(hits), q),
		TranscriptOutcome: "selection",
		UiDisposition:     "hidden",
		SearchResults:     wireHits,
		ActiveSearchIndex: &z,
	}

	if e.HostApply == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, true, "daemon has directives but no host apply client is configured"
	}
	if e.NavigateHitDirectives == nil || e.NewBatchID == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, true, "search engine not fully configured"
	}
	first := hits[0]
	dirs := e.NavigateHitDirectives(first.Path, first.Line0, first.Char0, first.Len)
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

	return res, true, ""
}

// FileSearchFromQuery collects unique matching paths and opens the first.
func (e *TranscriptSearch) FileSearchFromQuery(params protocol.VoiceTranscriptParams, q string, vs *session.VoiceSession) (protocol.VoiceTranscriptCompletion, bool, string) {
	q = strings.TrimSpace(q)
	if q == "" {
		return protocol.VoiceTranscriptCompletion{}, false, ""
	}
	root := strings.TrimSpace(workspace.EffectiveWorkspaceRoot(params.WorkspaceRoot, params.ActiveFile))
	if root == "" {
		return protocol.VoiceTranscriptCompletion{
			Success:           false,
			Summary:           "",
			TranscriptOutcome: "",
		}, true, "search requires workspaceRoot or activeFile"
	}

	hits, err := search.FixedStringSearch(root, q, fileSearchCollectMaxHits)
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, true, "file search failed: " + err.Error()
	}

	paths := search.UniqueSortedPaths(hits, fileSearchMaxUniquePaths)

	if vs != nil {
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

	if len(paths) == 0 {
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           fmt.Sprintf("no file path matches for %q", q),
			TranscriptOutcome: "file_selection",
			UiDisposition:     "hidden",
		}, true, ""
	}

	if e.HostApply == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, true, "daemon has directives but no host apply client is configured"
	}
	if e.NewBatchID == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, true, "search engine not fully configured"
	}
	first := paths[0]
	dirs := []protocol.VoiceTranscriptDirective{
		{
			Kind: "navigate",
			NavigationDirective: &protocol.NavigationDirective{
				Kind: "success",
				Action: &protocol.NavigationAction{
					Kind: "open_file",
					OpenFile: &struct {
						Path string `json:"path"`
					}{Path: first},
				},
			},
		},
	}
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

	return protocol.VoiceTranscriptCompletion{
		Success:                true,
		Summary:                fmt.Sprintf("found %d path(s) for %q", len(paths), q),
		TranscriptOutcome:      "file_selection",
		UiDisposition:          "hidden",
		FileSelectionFocusPath: first,
	}, true, ""
}

func hitsToProtocolSearchResults(hits []search.Hit) []struct {
	Path      string `json:"path"`
	Line      int64  `json:"line"`
	Character int64  `json:"character"`
	Preview   string `json:"preview"`
} {
	if len(hits) == 0 {
		return []struct {
			Path      string `json:"path"`
			Line      int64  `json:"line"`
			Character int64  `json:"character"`
			Preview   string `json:"preview"`
		}{}
	}
	out := make([]struct {
		Path      string `json:"path"`
		Line      int64  `json:"line"`
		Character int64  `json:"character"`
		Preview   string `json:"preview"`
	}, 0, len(hits))
	for _, h := range hits {
		out = append(out, struct {
			Path      string `json:"path"`
			Line      int64  `json:"line"`
			Character int64  `json:"character"`
			Preview   string `json:"preview"`
		}{
			Path:      h.Path,
			Line:      int64(h.Line0),
			Character: int64(h.Char0),
			Preview:   h.Preview,
		})
	}
	return out
}
