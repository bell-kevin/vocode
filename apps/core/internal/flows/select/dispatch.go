package selectflow

import (
	"vocoding.net/vocode/v2/apps/core/internal/flows"
	global "vocoding.net/vocode/v2/apps/core/internal/flows/global"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// SelectionDeps are dependencies for select-flow route handlers (post-classification).
type SelectionDeps struct {
	HostApply             protocolHostApply
	NewBatchID            func() string
	HitNavigateDirectives func(path string, line0, char0, length int) []protocol.VoiceTranscriptDirective
	Search                global.WorkspaceSearchApply
}

type protocolHostApply interface {
	ApplyDirectives(protocol.HostApplyParams) (protocol.HostApplyResult, error)
}

func routeDeps(d *SelectionDeps) *global.RouteDeps {
	if d == nil {
		return &global.RouteDeps{}
	}
	return &global.RouteDeps{Search: d.Search}
}

// DispatchRoute dispatches a classified select-flow route (global routes → global/, select_control → control.go).
func DispatchRoute(
	deps *SelectionDeps,
	params protocol.VoiceTranscriptParams,
	vs *session.VoiceSession,
	text string,
	route string,
	searchQuery string,
) (protocol.VoiceTranscriptCompletion, string) {
	rd := routeDeps(deps)
	switch route {
	case "control":
		return global.HandleControl(flows.Select, params, vs, text)
	case "select_control":
		return HandleSelectControl(deps, params, vs, text)
	case "select":
		return global.HandleSelect(rd, params, vs, flows.Select, searchQuery)
	case "select_file":
		return global.HandleSelectFile(rd, params, vs, flows.Select, searchQuery)
	case "edit":
		return HandleEdit(deps, params, vs, text)
	case "delete":
		return HandleDelete(deps, params, vs, text)
	case "irrelevant":
		return global.HandleIrrelevant(vs, flows.Select)
	default:
		return global.HandleIrrelevant(vs, flows.Select)
	}
}

func wireHitsToProtocol(in []session.SearchHit) []protocol.VoiceTranscriptSearchHit {
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
