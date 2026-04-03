package outcome

import (
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/transcript/clarify"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func sessionHitsFromProtocol(in []protocol.VoiceTranscriptSearchHit) []session.SearchHit {
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

// Apply mutates session state from grouped completion fields (and in-session file path lists).
func Apply(
	vs *session.VoiceSession,
	params protocol.VoiceTranscriptParams,
	res protocol.VoiceTranscriptCompletion,
) {
	if vs == nil {
		return
	}

	if res.Success && res.Clarify != nil && strings.TrimSpace(res.Clarify.TargetResolution) != "" && strings.TrimSpace(res.Summary) != "" {
		if err := clarify.ValidateForBasePhase(vs.BasePhase, res.Clarify.TargetResolution); err != nil {
			return
		}
		vs.Clarify = &session.ClarifyOverlay{
			TargetResolution:   res.Clarify.TargetResolution,
			Question:           strings.TrimSpace(res.Summary),
			OriginalTranscript: strings.TrimSpace(params.Text),
		}
		return
	}

	if res.Search != nil {
		s := res.Search
		if len(s.Results) > 0 {
			vs.SearchResults = sessionHitsFromProtocol(s.Results)
			if s.ActiveIndex != nil {
				i := int(*s.ActiveIndex)
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
			return
		}
		if s.Closed || s.NoHits {
			vs.SearchResults = nil
			vs.ActiveSearchIndex = 0
			vs.BasePhase = session.BasePhaseMain
			return
		}
	}

	fs := res.FileSelection
	if fs == nil {
		return
	}

	if len(fs.Results) > 0 {
		paths := make([]string, 0, len(fs.Results))
		for _, h := range fs.Results {
			paths = append(paths, strings.TrimSpace(h.Path))
		}
		vs.FileSelectionPaths = paths
		if fs.ActiveIndex != nil {
			i := int(*fs.ActiveIndex)
			if i < 0 || i >= len(paths) {
				i = 0
			}
			vs.FileSelectionIndex = i
			vs.FileSelectionFocus = paths[i]
		} else {
			vs.FileSelectionIndex = 0
			vs.FileSelectionFocus = paths[0]
		}
		vs.BasePhase = session.BasePhaseFileSelection
		vs.SearchResults = nil
		vs.ActiveSearchIndex = 0
		return
	}

	if fs.Closed || fs.NoHits {
		vs.FileSelectionPaths = nil
		vs.FileSelectionIndex = 0
		vs.FileSelectionFocus = ""
		vs.BasePhase = session.BasePhaseMain
		return
	}

	// Non-nil fileSelection with no results and not terminal: enter file-selection mode (wire {}).
	vs.BasePhase = session.BasePhaseFileSelection
	vs.FileSelectionPaths = nil
	vs.FileSelectionIndex = 0
	vs.FileSelectionFocus = ""
	vs.SearchResults = nil
	vs.ActiveSearchIndex = 0
}
