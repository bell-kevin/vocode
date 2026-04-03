package searchapply

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

// HostApplyClient applies navigation/edit directives on the host (e.g. VS Code).
type HostApplyClient interface {
	ApplyDirectives(protocol.HostApplyParams) (protocol.HostApplyResult, error)
}

// TranscriptSearch runs workspace content search and file-path search for transcript handling.
type TranscriptSearch struct {
	HostApply             HostApplyClient
	NewBatchID            func() string
	NavigateHitDirectives func(path string, line0, char0, length int) []protocol.VoiceTranscriptDirective
}

// SearchFromQuery runs fixed-string workspace content search and returns a completion with Search hits.
func (e *TranscriptSearch) SearchFromQuery(params protocol.VoiceTranscriptParams, q string, vs *session.VoiceSession) (protocol.VoiceTranscriptCompletion, bool, string) {
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

	hits, err := search.FixedStringSearch(root, q, contentSearchMaxHits)
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, true, "search failed: " + err.Error()
	}

	if len(hits) == 0 {
		return completionWorkspaceSearchNoHits(q), true, ""
	}

	res := completionWorkspaceSearchWithHits(hits, q)

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

func hitsToProtocolSearchResults(hits []search.Hit) []protocol.VoiceTranscriptSearchHit {
	if len(hits) == 0 {
		return nil
	}
	out := make([]protocol.VoiceTranscriptSearchHit, 0, len(hits))
	for _, h := range hits {
		out = append(out, protocol.VoiceTranscriptSearchHit{
			Path:      h.Path,
			Line:      int64(h.Line0),
			Character: int64(h.Char0),
			Preview:   h.Preview,
		})
	}
	return out
}

func completionWorkspaceSearchNoHits(q string) protocol.VoiceTranscriptCompletion {
	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       fmt.Sprintf("no matches for %q", q),
		UiDisposition: "hidden",
		Search: &protocol.VoiceTranscriptWorkspaceSearchState{
			NoHits: true,
		},
	}
}

func completionWorkspaceSearchWithHits(hits []search.Hit, q string) protocol.VoiceTranscriptCompletion {
	wireHits := hitsToProtocolSearchResults(hits)
	var z int64
	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       fmt.Sprintf("found %d matches for %q", len(hits), q),
		UiDisposition: "hidden",
		Search: &protocol.VoiceTranscriptWorkspaceSearchState{
			Results:     wireHits,
			ActiveIndex: &z,
		},
	}
}
