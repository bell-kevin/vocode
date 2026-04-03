package run

import (
	"context"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
	global "vocoding.net/vocode/v2/apps/core/internal/flows/global"
	rootflow "vocoding.net/vocode/v2/apps/core/internal/flows/root"
	"vocoding.net/vocode/v2/apps/core/internal/flows/router"
	selectflow "vocoding.net/vocode/v2/apps/core/internal/flows/select"
	selectfileflow "vocoding.net/vocode/v2/apps/core/internal/flows/selectfile"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/clarify"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Execute runs the full transcript accept pipeline (session load, clarify, flow dispatch, persist).
func Execute(e *Env, params protocol.VoiceTranscriptParams, opts *ExecuteOpts) (protocol.VoiceTranscriptCompletion, bool, string) {
	if e == nil || e.Ephemeral == nil || e.Sessions == nil {
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           "core transcript env not initialized",
			TranscriptOutcome: "completed",
			UiDisposition:     "hidden",
		}, true, ""
	}

	key := strings.TrimSpace(params.ContextSessionId)
	var vs session.VoiceSession
	if key == "" {
		vs = session.CloneVoiceSession(*e.Ephemeral)
	} else {
		vs = e.Sessions.Get(key, idleReset(params))
	}

	if cr := strings.TrimSpace(params.ControlRequest); cr != "" {
		out, ok := handleControlRequest(params, key, &vs, cr)
		e.persist(key, vs)
		return out, ok, ""
	}

	text := strings.TrimSpace(params.Text)
	if text == "" {
		return protocol.VoiceTranscriptCompletion{}, false, ""
	}

	if vs.Clarify != nil {
		if global.IsExitPhrase(text) {
			vs.Clarify = nil
			e.persist(key, vs)
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				Summary:           "Clarification cancelled",
				TranscriptOutcome: "clarify_control",
				UiDisposition:     "hidden",
			}, true, ""
		}

		ans := text
		ov := *vs.Clarify
		cc := clarify.BuildClarificationContext(ov, ans)
		_ = cc
		resumeFromClarification(&vs)
		vs.Clarify = nil
		e.persist(key, vs)
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           "Clarification resolved",
			TranscriptOutcome: "completed",
			UiDisposition:     "hidden",
		}, true, ""
	}

	if vs.BasePhase == "" {
		vs.BasePhase = session.BasePhaseMain
	}

	pre := executePreOpts(opts)

	switch vs.BasePhase {
	case session.BasePhaseSelection:
		route, searchQuery, ok := e.resolveSelectRoute(text, pre)
		if !ok {
			e.persist(key, vs)
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				Summary:           "core transcript (stub)",
				TranscriptOutcome: "completed",
				UiDisposition:     "hidden",
			}, true, ""
		}
		execRes, failure := selectflow.DispatchRoute(e.selectionDeps(), params, &vs, text, route, searchQuery)
		if strings.TrimSpace(failure) != "" {
			e.persist(key, vs)
			return protocol.VoiceTranscriptCompletion{Success: false}, true, failure
		}
		applyTranscriptOutcome(&vs, params, execRes)
		e.persist(key, vs)
		return execRes, true, ""

	case session.BasePhaseFileSelection:
		// Exit phrase closes file selection without requiring a hit list (matches legacy UX/tests).
		if global.IsExitPhrase(text) {
			vs.FileSelectionPaths = nil
			vs.FileSelectionIndex = 0
			vs.FileSelectionFocus = ""
			vs.BasePhase = session.BasePhaseMain
			e.persist(key, vs)
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				Summary:           "File selection closed",
				TranscriptOutcome: "completed",
				UiDisposition:     "hidden",
			}, true, ""
		}

		if len(vs.FileSelectionPaths) == 0 {
			vs.BasePhase = session.BasePhaseMain
			e.persist(key, vs)
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				Summary:           "No file hits in this session; use find file … (or your assistant’s file search) to get a path list first.",
				TranscriptOutcome: "completed",
				UiDisposition:     "shown",
			}, true, ""
		}

		if focused := strings.TrimSpace(params.FocusedWorkspacePath); focused != "" {
			for i, p := range vs.FileSelectionPaths {
				if p == focused {
					vs.FileSelectionIndex = i
					vs.FileSelectionFocus = p
					break
				}
			}
		}

		route, searchQuery, rOK, clsErrMsg := e.resolveSelectFileRoute(text, pre)
		if !rOK {
			e.persist(key, vs)
			if strings.TrimSpace(clsErrMsg) != "" {
				return protocol.VoiceTranscriptCompletion{}, true, clsErrMsg
			}
			return protocol.VoiceTranscriptCompletion{}, true, "flow classifier: empty route"
		}

		frRes, frFail := selectfileflow.DispatchRoute(e.selectFileDeps(), params, &vs, text, route, searchQuery)
		if strings.TrimSpace(frFail) != "" {
			e.persist(key, vs)
			if !frRes.Success {
				return frRes, true, frFail
			}
			return protocol.VoiceTranscriptCompletion{Success: false}, true, frFail
		}
		applyTranscriptOutcome(&vs, params, frRes)
		e.persist(key, vs)
		return frRes, true, ""
	}

	// Main / default
	if pre.has && pre.flow == flows.Root {
		rootD := e.rootDeps()
		fr := router.Result{Flow: flows.Root, Route: pre.route, SearchQuery: pre.searchQuery}
		res, fail := rootflow.DispatchRoute(rootD, params, &vs, text, fr)
		if strings.TrimSpace(fail) != "" {
			e.persist(key, vs)
			return protocol.VoiceTranscriptCompletion{Success: false}, true, fail
		}
		applyTranscriptOutcome(&vs, params, res)
		e.persist(key, vs)
		return res, true, ""
	}

	execRes, failure := rootflow.ExecuteMainPhase(e.rootDeps(), params, &vs, text)
	if strings.TrimSpace(failure) != "" {
		e.persist(key, vs)
		return protocol.VoiceTranscriptCompletion{Success: false}, true, failure
	}
	applyTranscriptOutcome(&vs, params, execRes)
	e.persist(key, vs)
	return execRes, true, ""
}

type preOpts struct {
	has         bool
	flow        flows.ID
	route       string
	searchQuery string
}

func executePreOpts(opts *ExecuteOpts) preOpts {
	if opts == nil || !opts.HasPreclassified {
		return preOpts{}
	}
	return preOpts{
		has:         true,
		flow:        opts.PreclassifiedFlow,
		route:       opts.PreclassifiedRoute,
		searchQuery: strings.TrimSpace(opts.PreclassifiedSearchQuery),
	}
}

func (e *Env) persist(key string, vs session.VoiceSession) {
	if strings.TrimSpace(key) == "" {
		*e.Ephemeral = session.CloneVoiceSession(vs)
		return
	}
	e.Sessions.Put(key, vs)
}

func (e *Env) selectionDeps() *selectflow.SelectionDeps {
	return &selectflow.SelectionDeps{
		HostApply:             e.HostApply,
		NewBatchID:            newApplyBatchID,
		HitNavigateDirectives: hitNavigateDirectives,
		Search:                e.Search,
	}
}

func (e *Env) selectFileDeps() *selectfileflow.SelectFileDeps {
	return &selectfileflow.SelectFileDeps{
		HostApply:  e.HostApply,
		NewBatchID: newApplyBatchID,
		Search:     e.Search,
	}
}

func (e *Env) rootDeps() *rootflow.RootDeps {
	return &rootflow.RootDeps{
		FlowRouter: e.FlowRouter,
		Search:     e.Search,
	}
}

func (e *Env) resolveSelectRoute(text string, pre preOpts) (route string, searchQuery string, ok bool) {
	if pre.has && pre.flow == flows.Select {
		return pre.route, pre.searchQuery, true
	}
	if e.FlowRouter == nil {
		return "", "", false
	}
	fr, err := e.FlowRouter.ClassifyFlow(context.Background(), router.Context{
		Flow:        flows.Select,
		Instruction: text,
	})
	if err != nil {
		return "", "", false
	}
	return fr.Route, fr.SearchQuery, true
}

func (e *Env) resolveSelectFileRoute(text string, pre preOpts) (route string, searchQuery string, ok bool, clsErr string) {
	if pre.has && pre.flow == flows.SelectFile {
		return pre.route, pre.searchQuery, true, ""
	}
	if e.FlowRouter == nil {
		return "", "", false, ""
	}
	fr, err := e.FlowRouter.ClassifyFlow(context.Background(), router.Context{
		Flow:        flows.SelectFile,
		Instruction: text,
	})
	if err != nil {
		return "", "", false, err.Error()
	}
	return fr.Route, fr.SearchQuery, true, ""
}
