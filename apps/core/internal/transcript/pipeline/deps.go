package pipeline

import (
	"context"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
	fileselectflow "vocoding.net/vocode/v2/apps/core/internal/flows/fileselect"
	rootflow "vocoding.net/vocode/v2/apps/core/internal/flows/root"
	"vocoding.net/vocode/v2/apps/core/internal/flows/router"
	workspaceselectflow "vocoding.net/vocode/v2/apps/core/internal/flows/workspaceselect"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/hostdirectives"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/run"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type preOpts struct {
	has              bool
	flow             flows.ID
	route            string
	searchQuery      string
	searchSymbolKind string
}

func preFromOpts(opts *Opts) preOpts {
	if opts == nil || !opts.HasPreclassified {
		return preOpts{}
	}
	return preOpts{
		has:              true,
		flow:             opts.PreclassifiedFlow,
		route:            opts.PreclassifiedRoute,
		searchQuery:      strings.TrimSpace(opts.PreclassifiedSearchQuery),
		searchSymbolKind: strings.TrimSpace(strings.ToLower(opts.PreclassifiedSearchSymbolKind)),
	}
}

func persist(e *run.Env, key string, vs session.VoiceSession) {
	if strings.TrimSpace(key) == "" {
		*e.Ephemeral = session.CloneVoiceSession(vs)
		return
	}
	e.Sessions.Put(key, vs)
}

func selectionDeps(e *run.Env) *workspaceselectflow.SelectionDeps {
	return &workspaceselectflow.SelectionDeps{
		HostApply:     e.ExtensionHost,
		ExtensionHost: e.ExtensionHost,
		EditModel:     e.EditModel,
		NewBatchID:    hostdirectives.NewApplyBatchID,
		HitNavigateDirectives: func(params protocol.VoiceTranscriptParams, path string, line0, char0, length int) []protocol.VoiceTranscriptDirective {
			syms := hostdirectives.DocumentSymbolsForPath(e.ExtensionHost, params, path)
			return hostdirectives.HitNavigateDirectivesExpandWithSymbols(path, line0, char0, length, syms)
		},
		Search: e.Search,
	}
}

func selectFileDeps(e *run.Env) *fileselectflow.SelectFileDeps {
	return &fileselectflow.SelectFileDeps{
		HostApply:  e.ExtensionHost,
		NewBatchID: hostdirectives.NewApplyBatchID,
		Search:     e.Search,
		Editor:     selectionDeps(e),
	}
}

func rootDeps(e *run.Env) *rootflow.RootDeps {
	return &rootflow.RootDeps{
		FlowRouter: e.FlowRouter,
		Search:     e.Search,
		Editor:     selectionDeps(e),
	}
}

func resolveWorkspaceSelectRoute(e *run.Env, params protocol.VoiceTranscriptParams, text string, pre preOpts) (route, searchQuery, searchSymbolKind string, ok bool) {
	if pre.has && pre.flow == flows.WorkspaceSelect {
		return pre.route, pre.searchQuery, pre.searchSymbolKind, true
	}
	if e.FlowRouter == nil {
		return "", "", "", false
	}
	fr, err := e.FlowRouter.ClassifyFlow(context.Background(), router.ContextForClassification(flows.WorkspaceSelect, text, params))
	if err != nil {
		return "", "", "", false
	}
	return fr.Route, fr.SearchQuery, fr.SearchSymbolKind, true
}

func resolveSelectFileRoute(e *run.Env, params protocol.VoiceTranscriptParams, text string, pre preOpts) (route, searchQuery, searchSymbolKind string, ok bool, clsErr string) {
	if pre.has && pre.flow == flows.SelectFile {
		return pre.route, pre.searchQuery, pre.searchSymbolKind, true, ""
	}
	if e.FlowRouter == nil {
		return "", "", "", false, ""
	}
	fr, err := e.FlowRouter.ClassifyFlow(context.Background(), router.ContextForClassification(flows.SelectFile, text, params))
	if err != nil {
		return "", "", "", false, err.Error()
	}
	return fr.Route, fr.SearchQuery, fr.SearchSymbolKind, true, ""
}
