package run

import (
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/transcript/clarify"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func sessionHitsFromProtocol(in []struct {
	Path      string `json:"path"`
	Line      int64  `json:"line"`
	Character int64  `json:"character"`
	Preview   string `json:"preview"`
},
) []session.SearchHit {
	out := make([]session.SearchHit, 0, len(in))
	for _, h := range in {
		out = append(out, session.SearchHit{
			Path:      h.Path,
			Line:      int(h.Line),
			Character: int(h.Character),
			Preview:   h.Preview,
		})
	}
	return out
}

// applyTranscriptOutcome mutates session state based on transcript outcome.
func applyTranscriptOutcome(
	vs *session.VoiceSession,
	params protocol.VoiceTranscriptParams,
	res protocol.VoiceTranscriptCompletion,
) {
	if vs == nil {
		return
	}

	switch res.TranscriptOutcome {
	case "search", "selection", "selection_control":
		if len(res.SearchResults) > 0 {
			vs.SearchResults = sessionHitsFromProtocol(res.SearchResults)
			if res.ActiveSearchIndex != nil {
				i := int(*res.ActiveSearchIndex)
				if i < 0 {
					i = 0
				}
				if i >= len(vs.SearchResults) {
					i = 0
				}
				vs.ActiveSearchIndex = i
			} else {
				vs.ActiveSearchIndex = 0
			}
			vs.BasePhase = session.BasePhaseSelection
		} else if res.SearchResults != nil {
			vs.SearchResults = nil
			vs.ActiveSearchIndex = 0
			vs.BasePhase = session.BasePhaseMain
		}

	case "file_selection", "file_selection_control":
		vs.BasePhase = session.BasePhaseFileSelection
		if res.FileSelectionFocusPath != "" {
			vs.FileSelectionFocus = res.FileSelectionFocusPath
		}
		vs.SearchResults = nil
		vs.ActiveSearchIndex = 0

	case "clarify":
		if strings.TrimSpace(res.Summary) == "" {
			return
		}
		if err := clarify.ValidateForBasePhase(vs.BasePhase, res.ClarifyTargetResolution); err != nil {
			return
		}
		vs.Clarify = &session.ClarifyOverlay{
			TargetResolution:   res.ClarifyTargetResolution,
			Question:           strings.TrimSpace(res.Summary),
			OriginalTranscript: strings.TrimSpace(params.Text),
		}
	}
}
