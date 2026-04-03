package transcript

import (
	"context"
	"strings"
	"sync"

	"vocoding.net/vocode/v2/apps/core/internal/agent"
	"vocoding.net/vocode/v2/apps/core/internal/flows"
	"vocoding.net/vocode/v2/apps/core/internal/flows/router"
	"vocoding.net/vocode/v2/apps/core/internal/rpc"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/idle"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/pipeline"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/run"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

const defaultTranscriptQueueSize = 64

// Service is the RPC-facing transcript facade: nil-safe, bounded queue for spoken transcripts, and
// a fast path for protocol ControlRequest (cancel_clarify, cancel_selection, …) that bypasses the queue.
//
// ControlRequest is handled inside [pipeline.Execute] before FlowRouter classification and skips the job queue.
type Service struct {
	env *run.Env

	executeMu sync.Mutex
	queue     chan transcriptJob
	startOnce sync.Once
}

type transcriptJob struct {
	params protocol.VoiceTranscriptParams
	resp   chan transcriptAcceptResp
}

type transcriptAcceptResp struct {
	result protocol.VoiceTranscriptCompletion
	ok     bool
	reason string
}

func NewService(flowRouter *router.FlowRouter, editModel agent.ModelClient) *Service {
	if flowRouter == nil {
		flowRouter = router.NewFlowRouter(nil)
	}
	ephemeral := session.VoiceSession{}
	env := &run.Env{
		Sessions:   session.NewVoiceSessionStore(),
		Ephemeral:  &ephemeral,
		EditModel:  editModel,
		FlowRouter: flowRouter,
	}
	run.WireSearchEngine(env)
	return &Service{env: env}
}

func (s *Service) SetHostApplyClient(
	client interface {
		ApplyDirectives(protocol.HostApplyParams) (protocol.HostApplyResult, error)
	},
) {
	if s == nil || s.env == nil {
		return
	}
	if eh, ok := client.(rpc.ExtensionHost); ok {
		s.env.ExtensionHost = eh
	} else {
		s.env.ExtensionHost = nil
	}
	if s.env.Search != nil {
		s.env.Search.HostApply = client
		if eh, ok := client.(rpc.ExtensionHost); ok {
			s.env.Search.ExtensionHost = eh
		} else {
			s.env.Search.ExtensionHost = nil
		}
		run.WireSearchEngine(s.env)
	}
}

func (s *Service) AcceptTranscript(params protocol.VoiceTranscriptParams) (protocol.VoiceTranscriptCompletion, bool, string) {
	if s == nil || s.env == nil {
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "core daemon not initialized yet",
			UiDisposition: "hidden",
		}, true, ""
	}

	s.startOnce.Do(func() {
		s.queue = make(chan transcriptJob, defaultTranscriptQueueSize)
		go s.runWorker()
	})

	cr := strings.TrimSpace(params.ControlRequest)
	if cr != "" {
		params.Text = strings.TrimSpace(params.Text)
		params.ControlRequest = cr
		s.executeMu.Lock()
		defer s.executeMu.Unlock()
		return pipeline.Execute(s.env, params, nil)
	}

	params.Text = strings.TrimSpace(params.Text)
	if params.Text == "" {
		return protocol.VoiceTranscriptCompletion{}, false, ""
	}

	if handled, res, ok, reason := s.tryImmediateAfterClassify(params); handled {
		return res, ok, reason
	}

	respCh := make(chan transcriptAcceptResp, 1)
	job := transcriptJob{params: params, resp: respCh}
	select {
	case s.queue <- job:
	default:
		return protocol.VoiceTranscriptCompletion{
			Success:       false,
			UiDisposition: "hidden",
		}, true, "voice.transcript queue is full"
	}
	resp := <-respCh
	return resp.result, resp.ok, resp.reason
}

func (s *Service) tryImmediateAfterClassify(params protocol.VoiceTranscriptParams) (handled bool, res protocol.VoiceTranscriptCompletion, ok bool, reason string) {
	s.executeMu.Lock()
	defer s.executeMu.Unlock()

	key := strings.TrimSpace(params.ContextSessionId)
	var vs session.VoiceSession
	if key == "" {
		vs = session.CloneVoiceSession(*s.env.Ephemeral)
	} else {
		vs = s.env.Sessions.Get(key, idle.SessionResetDuration(params))
	}

	if vs.Clarify != nil {
		return false, protocol.VoiceTranscriptCompletion{}, true, ""
	}

	if vs.BasePhase == "" {
		vs.BasePhase = session.BasePhaseMain
	}

	flowID := basePhaseToFlow(vs.BasePhase)
	fr, clsErr := classifySpoken(s.env.FlowRouter, flowID, params.Text)
	if clsErr != nil || strings.TrimSpace(fr.Route) == "" {
		return false, protocol.VoiceTranscriptCompletion{}, true, ""
	}

	if flows.RouteExecution(flowID, fr.Route) != flows.ExecutionImmediate {
		return false, protocol.VoiceTranscriptCompletion{}, true, ""
	}

	opts := &pipeline.Opts{
		HasPreclassified:         true,
		PreclassifiedFlow:        flowID,
		PreclassifiedRoute:       fr.Route,
		PreclassifiedSearchQuery: fr.SearchQuery,
	}
	r, execOK, rea := pipeline.Execute(s.env, params, opts)
	return true, r, execOK, rea
}

func basePhaseToFlow(phase session.BasePhase) flows.ID {
	switch phase {
	case session.BasePhaseSelection:
		return flows.WorkspaceSelect
	case session.BasePhaseFileSelection:
		return flows.SelectFile
	default:
		return flows.Root
	}
}

func classifySpoken(r *router.FlowRouter, flow flows.ID, text string) (router.Result, error) {
	if r == nil {
		return router.Result{}, nil
	}
	return r.ClassifyFlow(context.Background(), router.Context{Flow: flow, Instruction: text})
}

func (s *Service) runWorker() {
	for job := range s.queue {
		s.executeMu.Lock()
		res, ok, reason := pipeline.Execute(s.env, job.params, nil)
		s.executeMu.Unlock()
		job.resp <- transcriptAcceptResp{result: res, ok: ok, reason: reason}
	}
}
