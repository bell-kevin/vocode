package fileselectflow

import (
	"vocoding.net/vocode/v2/apps/core/internal/flows"
	global "vocoding.net/vocode/v2/apps/core/internal/flows/global"
	workspaceselectflow "vocoding.net/vocode/v2/apps/core/internal/flows/workspaceselect"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// SelectFileDeps are dependencies for select-file flow route handlers.
type SelectFileDeps struct {
	HostApply  protocolHostApply
	NewBatchID func() string
	Search     global.WorkspaceSearchApply
	// Editor supplies host/model deps for global route "create" (mutate active file).
	Editor *workspaceselectflow.SelectionDeps
}

type protocolHostApply interface {
	ApplyDirectives(protocol.HostApplyParams) (protocol.HostApplyResult, error)
}

func routeDeps(d *SelectFileDeps) *global.RouteDeps {
	if d == nil {
		return &global.RouteDeps{}
	}
	return &global.RouteDeps{Search: d.Search}
}

// DispatchRoute dispatches a classified select-file route (global routes → global/, file_select_control → control.go).
func DispatchRoute(
	deps *SelectFileDeps,
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
		return global.HandleControl(flows.SelectFile, params, vs, text)
	case "file_select_control":
		return HandleSelectFileControl(deps, params, vs, text)
	case "workspace_select":
		return global.HandleWorkspaceSelect(rd, params, vs, flows.SelectFile, searchQuery, searchSymbolKind)
	case "select_file":
		return global.HandleSelectFile(rd, params, vs, flows.SelectFile, searchQuery, searchSymbolKind)
	case "delete":
		return HandleDelete(deps, params, vs, text)
	case "move":
		return HandleMove(deps, params, vs, text)
	case "rename":
		return HandleRename(deps, params, vs, text)
	case "create":
		if deps == nil || deps.Editor == nil {
			return global.HandleIrrelevant(vs, flows.SelectFile)
		}
		return workspaceselectflow.HandleCreate(deps.Editor, params, vs, text)
	case "command":
		if deps == nil || deps.Editor == nil {
			return global.HandleIrrelevant(vs, flows.SelectFile)
		}
		e := deps.Editor
		cd := global.CommandDeps{
			HostApply:     e.HostApply,
			ExtensionHost: e.ExtensionHost,
			EditModel:     e.EditModel,
			NewBatchID:    e.NewBatchID,
		}
		return global.HandleCommand(&cd, params, vs, text)
	case "create_entry":
		return HandleCreateEntry(deps, params, vs, text)
	case "irrelevant":
		return global.HandleIrrelevant(vs, flows.SelectFile)
	default:
		return global.HandleIrrelevant(vs, flows.SelectFile)
	}
}
