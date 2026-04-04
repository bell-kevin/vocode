package workspaceselectflow

import (
	"vocoding.net/vocode/v2/apps/core/internal/agent"
	"vocoding.net/vocode/v2/apps/core/internal/flows"
	global "vocoding.net/vocode/v2/apps/core/internal/flows/global"
	"vocoding.net/vocode/v2/apps/core/internal/flows/router"
	"vocoding.net/vocode/v2/apps/core/internal/rpc"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// SelectionDeps are dependencies for workspace-select flow route handlers (post-classification).
type SelectionDeps struct {
	HostApply             protocolHostApply
	ExtensionHost         rpc.ExtensionHost
	EditModel             agent.ModelClient
	NewBatchID            func() string
	HitNavigateDirectives func(params protocol.VoiceTranscriptParams, path string, line0, char0, length int) []protocol.VoiceTranscriptDirective
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

// DispatchRoute dispatches a classified workspace-select flow route (global routes → global/, workspace_select_control → control.go).
func DispatchRoute(
	deps *SelectionDeps,
	params protocol.VoiceTranscriptParams,
	vs *session.VoiceSession,
	text string,
	route string,
	searchQuery string,
	searchSymbolKind string,
) (protocol.VoiceTranscriptCompletion, string) {
	rd := routeDeps(deps)
	switch route {
	case "control":
		return global.HandleControl(flows.WorkspaceSelect, params, vs, text)
	case "workspace_select_control":
		return HandleSelectControl(deps, params, vs, text)
	case "workspace_select":
		return global.HandleWorkspaceSelect(rd, params, vs, flows.WorkspaceSelect, searchQuery, searchSymbolKind)
	case "file_select":
		return global.HandleFileSelect(rd, params, vs, flows.WorkspaceSelect, searchQuery, searchSymbolKind)
	case "edit":
		return HandleEdit(deps, params, vs, text)
	case "create":
		if c, msg, ok := router.RejectCreateWhenEditorSelection(params); ok {
			return c, msg
		}
		return HandleCreate(deps, params, vs, text)
	case "command":
		if deps == nil {
			return global.HandleIrrelevant(vs, flows.WorkspaceSelect)
		}
		cd := global.CommandDeps{
			HostApply:     deps.HostApply,
			ExtensionHost: deps.ExtensionHost,
			EditModel:     deps.EditModel,
			NewBatchID:    deps.NewBatchID,
		}
		return global.HandleCommand(&cd, params, vs, text)
	case "rename":
		return HandleRename(deps, params, vs, text)
	case "delete":
		return HandleDelete(deps, params, vs, text)
	case "irrelevant":
		return global.HandleIrrelevant(vs, flows.WorkspaceSelect)
	default:
		return global.HandleIrrelevant(vs, flows.WorkspaceSelect)
	}
}

func wireHitsToProtocol(in []session.SearchHit) []protocol.VoiceTranscriptSearchHit {
	out := make([]protocol.VoiceTranscriptSearchHit, 0, len(in))
	for _, h := range in {
		ml := int64(h.Len)
		if ml <= 0 {
			ml = 1
		}
		p := ml
		out = append(out, protocol.VoiceTranscriptSearchHit{
			Path:        h.Path,
			Line:        int64(h.Line),
			Character:   int64(h.Character),
			Preview:     h.Preview,
			MatchLength: &p,
		})
	}
	return out
}
