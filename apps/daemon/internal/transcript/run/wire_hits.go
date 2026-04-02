package run

import (
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
)

func ptrInt64(v int64) *int64 { return &v }

func voiceSessionHitsToWire(in []agentcontext.SearchHit) []struct {
	Path      string `json:"path"`
	Line      int64  `json:"line"`
	Character int64  `json:"character"`
	Preview   string `json:"preview"`
} {
	out := make([]struct {
		Path      string `json:"path"`
		Line      int64  `json:"line"`
		Character int64  `json:"character"`
		Preview   string `json:"preview"`
	}, 0, len(in))
	for _, h := range in {
		out = append(out, struct {
			Path      string `json:"path"`
			Line      int64  `json:"line"`
			Character int64  `json:"character"`
			Preview   string `json:"preview"`
		}{Path: h.Path, Line: int64(h.Line), Character: int64(h.Character), Preview: h.Preview})
	}
	return out
}

func wireHitsToVoiceSession(in []struct {
	Path      string `json:"path"`
	Line      int64  `json:"line"`
	Character int64  `json:"character"`
	Preview   string `json:"preview"`
},
) []agentcontext.SearchHit {
	out := make([]agentcontext.SearchHit, 0, len(in))
	for _, h := range in {
		out = append(out, agentcontext.SearchHit{
			Path:      h.Path,
			Line:      int(h.Line),
			Character: int(h.Character),
			Preview:   h.Preview,
		})
	}
	return out
}

// syncSelectionStackForHits keeps the selection flow frame aligned with SearchResults.
// Clarify/file frames are not inferred from other session fields — only explicit FlowPush/Pop.
func syncSelectionStackForHits(vs *agentcontext.VoiceSession) {
	if vs == nil {
		return
	}
	if len(vs.SearchResults) == 0 {
		for agentcontext.FlowTopKind(vs.FlowStack) == agentcontext.FlowKindSelection {
			ns, _, ok := agentcontext.FlowPop(vs.FlowStack)
			if !ok {
				break
			}
			vs.FlowStack = ns
		}
		return
	}
	if agentcontext.FlowTopKind(vs.FlowStack) == agentcontext.FlowKindMain {
		if ns, ok := agentcontext.FlowPush(vs.FlowStack, agentcontext.FlowFrame{Kind: agentcontext.FlowKindSelection}); ok {
			vs.FlowStack = ns
		}
	}
}
