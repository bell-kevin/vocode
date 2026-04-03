package run

import (
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func ptrInt64(v int64) *int64 { return &v }

func voiceSessionHitsToWire(in []agentcontext.SearchHit) []protocol.VoiceTranscriptSearchHit {
	out := make([]protocol.VoiceTranscriptSearchHit, 0, len(in))
	for _, h := range in {
		out = append(out, protocol.VoiceTranscriptSearchHit{
			Path:      h.Path,
			Line:      int64(h.Line),
			Character: int64(h.Character),
			Preview:   h.Preview,
		})
	}
	return out
}

func wireHitsToVoiceSession(in []protocol.VoiceTranscriptSearchHit) []agentcontext.SearchHit {
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
	if len(vs.SearchResults) > 0 {
		if stackHasSelectionFrame(vs.FlowStack) {
			return
		}
		if agentcontext.FlowTopKind(vs.FlowStack) == agentcontext.FlowKindMain {
			if ns, ok := agentcontext.FlowPush(vs.FlowStack, agentcontext.FlowFrame{Kind: agentcontext.FlowKindSelection}); ok {
				vs.FlowStack = ns
			}
		}
		return
	}
	if len(vs.SearchResults) == 0 {
		for {
			top := agentcontext.FlowTopKind(vs.FlowStack)
			if top == agentcontext.FlowKindSelection {
				ns, _, ok := agentcontext.FlowPop(vs.FlowStack)
				if !ok {
					break
				}
				vs.FlowStack = ns
				continue
			}
			if top == agentcontext.FlowKindClarify && len(vs.FlowStack) >= 2 {
				below := vs.FlowStack[len(vs.FlowStack)-2].Kind
				if below == agentcontext.FlowKindSelection {
					ns, _, ok := agentcontext.FlowPop(vs.FlowStack)
					if !ok {
						break
					}
					vs.FlowStack = ns
					continue
				}
			}
			break
		}
		return
	}
}

func stackHasSelectionFrame(stack []agentcontext.FlowFrame) bool {
	for i := range stack {
		if stack[i].Kind == agentcontext.FlowKindSelection {
			return true
		}
	}
	return false
}
