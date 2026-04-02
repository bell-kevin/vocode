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

	if res.TranscriptOutcome == "irrelevant" &&
		(topFlow == agentcontext.FlowKindFileSelection ||
			(topFlow == agentcontext.FlowKindSelection && hasSearchHits)) {
		res.UiDisposition = "hidden"
		return
	}

	if strings.TrimSpace(res.UiDisposition) == "" {
		res.UiDisposition = inferTranscriptUIDisposition(res.TranscriptOutcome, hasSearchHits)
	}
}

// inferTranscriptUIDisposition is the default mapping from transcriptOutcome → uiDisposition when
// the executor or a flow handler leaves uiDisposition empty. Keep in sync with protocol docs and
// the extension panel (main-panel-store fallback switch).
func inferTranscriptUIDisposition(outcome string, hasActiveSearchHits bool) string {
	switch strings.TrimSpace(outcome) {
	case "", "completed":
		return "shown"
	case "answer":
		return "hidden"
	case "search", "selection", "selection_control",
		"clarify", "clarify_control",
		"file_selection", "file_selection_control":
		return "hidden"
	case "needs_workspace_folder":
		return "shown"
	case "irrelevant":
		if hasActiveSearchHits {
			return "hidden"
		}
		return "skipped"
	default:
		return "hidden"
	}
}
