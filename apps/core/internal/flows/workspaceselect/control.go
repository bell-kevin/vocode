package workspaceselectflow

import (
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/flows/helpers"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// HandleSelectControl handles the workspace-select flow "workspace_select_control" route only (hit-list navigation via [selection.ParseNav]).
func HandleSelectControl(deps *SelectionDeps, params protocol.VoiceTranscriptParams, vs *session.VoiceSession, text string) (protocol.VoiceTranscriptCompletion, string) {
	op, pick, ok := listNavOp(text)
	if !ok {
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			UiDisposition: "skipped",
		}, ""
	}
	applySelectControlOp(vs, op, pick)
	return selectApplyHostForActiveHit(deps, params, vs)
}

func applySelectControlOp(vs *session.VoiceSession, op string, pick1Based int) {
	op = strings.ToLower(strings.TrimSpace(op))
	n := len(vs.SearchResults)
	switch op {
	case "next":
		if n > 0 {
			if vs.ActiveSearchIndex < n-1 {
				vs.ActiveSearchIndex++
			} else {
				vs.ActiveSearchIndex = 0
			}
		}
	case "back":
		if n > 0 {
			if vs.ActiveSearchIndex > 0 {
				vs.ActiveSearchIndex--
			} else {
				vs.ActiveSearchIndex = n - 1
			}
		}
	case "pick":
		if pick1Based >= 1 && pick1Based <= n {
			vs.ActiveSearchIndex = pick1Based - 1
		}
	}
}

func selectApplyHostForActiveHit(deps *SelectionDeps, params protocol.VoiceTranscriptParams, vs *session.VoiceSession) (protocol.VoiceTranscriptCompletion, string) {
	if len(vs.SearchResults) == 0 {
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "search results",
			UiDisposition: "hidden",
			Search: &protocol.VoiceTranscriptWorkspaceSearchState{
				Closed: true,
			},
		}, ""
	}
	if deps.HostApply == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "host apply client not configured"
	}
	if deps.HitNavigateDirectives == nil || deps.NewBatchID == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "select flow not fully configured"
	}
	hit := vs.SearchResults[vs.ActiveSearchIndex]
	needleLen := hit.Len
	if needleLen <= 0 {
		needleLen = 1
	}
	dirs := deps.HitNavigateDirectives(params, hit.Path, hit.Line, hit.Character, needleLen)
	batchID := deps.NewBatchID()
	pending := &session.DirectiveApplyBatch{ID: batchID, NumDirectives: len(dirs)}
	vs.PendingDirectiveApply = pending
	hostRes, err := deps.HostApply.ApplyDirectives(protocol.HostApplyParams{
		ApplyBatchId: batchID,
		ActiveFile:   params.ActiveFile,
		Directives:   dirs,
	})
	if err != nil {
		vs.PendingDirectiveApply = nil
		return protocol.VoiceTranscriptCompletion{Success: false}, "host.applyDirectives failed: " + err.Error()
	}
	if err := pending.ConsumeHostApplyReport(batchID, hostRes.Items); err != nil {
		vs.PendingDirectiveApply = nil
		return protocol.VoiceTranscriptCompletion{Success: false}, "host apply failed: " + err.Error()
	}
	vs.PendingDirectiveApply = nil
	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       "search results",
		UiDisposition: "hidden",
		Search: &protocol.VoiceTranscriptWorkspaceSearchState{
			Results:     wireHitsToProtocol(vs.SearchResults),
			ActiveIndex: ptrInt64(int64(vs.ActiveSearchIndex)),
		},
	}, ""
}

func listNavOp(text string) (op string, pick1Based int, ok bool) {
	k, ord, ok := helpers.ParseNav(text)
	if !ok {
		return "", 0, false
	}
	switch k {
	case "next":
		return "next", 0, true
	case "back":
		return "back", 0, true
	case "pick":
		return "pick", ord, true
	default:
		return "", 0, false
	}
}

func ptrInt64(v int64) *int64 { return &v }
