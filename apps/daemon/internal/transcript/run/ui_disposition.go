package run

import (
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// applyTranscriptUIDisposition sets completion.UiDisposition when the producer omitted it, and
// applies session-aware rules (e.g. irrelevant utterances during search/file-selection should not
// surface as "skipped" in the sidebar).
func applyTranscriptUIDisposition(
	res *protocol.VoiceTranscriptCompletion,
	topFlow string,
	hasSearchHits bool,
) {
	if res == nil {
		return
	}
	if !res.Success {
		if strings.TrimSpace(res.UiDisposition) == "" {
			res.UiDisposition = "hidden"
		}
		return
	}

	if strings.TrimSpace(res.UiDisposition) == "skipped" &&
		(topFlow == agentcontext.FlowKindFileSelection ||
			(topFlow == agentcontext.FlowKindSelection && hasSearchHits)) {
		res.UiDisposition = "hidden"
		return
	}

	if strings.TrimSpace(res.UiDisposition) == "" {
		res.UiDisposition = inferTranscriptUIDisposition(res)
	}
}

// inferTranscriptUIDisposition is the default mapping when uiDisposition is empty. Keep in sync
// with the extension panel.
func inferTranscriptUIDisposition(res *protocol.VoiceTranscriptCompletion) string {
	if res.Question != nil && strings.TrimSpace(res.Question.AnswerText) != "" {
		return "hidden"
	}
	if res.Search != nil {
		s := res.Search
		if s.Closed || s.NoHits || len(s.Results) > 0 {
			return "hidden"
		}
	}
	if res.Clarify != nil && strings.TrimSpace(res.Clarify.TargetResolution) != "" {
		return "hidden"
	}
	if res.FileSelection != nil {
		return "hidden"
	}
	if res.Workspace != nil && res.Workspace.NeedsFolder {
		return "shown"
	}
	return "shown"
}
