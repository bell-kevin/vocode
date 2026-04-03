package rootflow

import (
	"context"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
	global "vocoding.net/vocode/v2/apps/core/internal/flows/global"
	"vocoding.net/vocode/v2/apps/core/internal/flows/router"
	workspaceselectflow "vocoding.net/vocode/v2/apps/core/internal/flows/workspaceselect"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// RootDeps are dependencies for root-flow handling (main transcript phase).
type RootDeps struct {
	FlowRouter *router.FlowRouter
	Search     global.WorkspaceSearchApply
	// Editor supplies host/model deps for routes that mutate the active file (e.g. create).
	Editor *workspaceselectflow.SelectionDeps
}

// IrrelevantSkipped is the completion when root has nothing actionable (no heuristic search).
func IrrelevantSkipped() (protocol.VoiceTranscriptCompletion, string) {
	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		UiDisposition: "skipped",
	}, ""
}

// DispatchRoute handles a root-flow route after classification (same naming pattern as select/selectfile dispatch).
func DispatchRoute(
	deps *RootDeps,
	params protocol.VoiceTranscriptParams,
	vs *session.VoiceSession,
	text string,
	res router.Result,
) (protocol.VoiceTranscriptCompletion, string) {
	text = strings.TrimSpace(text)
	rd := &global.RouteDeps{Search: nil}
	if deps != nil {
		rd.Search = deps.Search
	}

	switch res.Route {
	case "workspace_select":
		if r, fail, ok := global.TryHandleWorkspaceSelectSearch(rd, params, vs, res.SearchQuery, res.SearchSymbolKind); ok {
			return r, fail
		}
		return IrrelevantSkipped()

	case "select_file":
		if r, fail, ok := global.TryHandleSelectFileSearch(rd, params, vs, res.SearchQuery, res.SearchSymbolKind, flows.Root); ok {
			return r, fail
		}
		return IrrelevantSkipped()

	case "create":
		if deps == nil || deps.Editor == nil {
			return IrrelevantSkipped()
		}
		return workspaceselectflow.HandleCreate(deps.Editor, params, vs, text)

	case "command":
		if deps == nil || deps.Editor == nil {
			return IrrelevantSkipped()
		}
		e := deps.Editor
		cd := global.CommandDeps{
			HostApply:     e.HostApply,
			ExtensionHost: e.ExtensionHost,
			EditModel:     e.EditModel,
			NewBatchID:    e.NewBatchID,
		}
		return global.HandleCommand(&cd, params, vs, text)

	case "question":
		return HandleQuestion(deps, params, text)

	case "irrelevant":
		return global.HandleIrrelevant(vs, flows.Root)

	case "control":
		return global.HandleControl(flows.Root, params, vs, text)

	default:
		return IrrelevantSkipped()
	}
}

// ExecuteMainPhase runs root-flow classification and route handling. Workspace search runs only
// when the classifier returns workspace_select or select_file with a non-empty search_query (see global.TryHandle*).
func ExecuteMainPhase(
	deps *RootDeps,
	params protocol.VoiceTranscriptParams,
	vs *session.VoiceSession,
	text string,
) (protocol.VoiceTranscriptCompletion, string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "Nothing transcribed",
			UiDisposition: "hidden",
		}, ""
	}

	if deps == nil || deps.FlowRouter == nil {
		return IrrelevantSkipped()
	}

	fr, err := deps.FlowRouter.ClassifyFlow(context.Background(), router.ContextForClassification(flows.Root, text, params))
	if err != nil {
		return IrrelevantSkipped()
	}
	return DispatchRoute(deps, params, vs, text, fr)
}
