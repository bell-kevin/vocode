package transcript

import (
	"log"
	"strings"
	"sync"
	"time"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/config"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/executor"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/run"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// TranscriptService adapts the voice.transcript RPC to the agent runtime and
// transcript executor (structured directives for the extension to apply).
type TranscriptService struct {
	executor *executor.Executor
	logger   *log.Logger

	sessions *agentcontext.VoiceSessionStore
	// When contextSessionId is empty, full session (gathered, apply history, pending batch) is kept here.
	ephemeralVoiceSession agentcontext.VoiceSession

	executeMu sync.Mutex

	queue chan transcriptJob

	coalesceWindow time.Duration
	maxMergeJobs   int
	maxMergeChars  int
	startOnce      sync.Once

	hostApplyClient hostApplyClient
}

type transcriptJob struct {
	params protocol.VoiceTranscriptParams
	resp   chan transcriptAcceptResp
}

type hostApplyClient interface {
	ApplyDirectives(protocol.HostApplyParams) (protocol.HostApplyResult, error)
}

func (s *TranscriptService) SetHostApplyClient(client hostApplyClient) {
	s.hostApplyClient = client
}

type transcriptAcceptResp struct {
	result protocol.VoiceTranscriptCompletion
	ok     bool
	reason string
}

func NewService(
	agentRuntime *agent.Agent,
	logger *log.Logger,
) *TranscriptService {
	queueSize := config.DefaultTranscriptQueueSize
	coalesceMs := config.DefaultTranscriptCoalesceMs
	maxMergeJobs := config.DefaultTranscriptMaxMergeJobs
	maxMergeChars := config.DefaultTranscriptMaxMergeChars
	exec := executor.New(agentRuntime, executor.Options{})

	s := &TranscriptService{
		executor:       exec,
		logger:         logger,
		sessions:       agentcontext.NewVoiceSessionStore(),
		coalesceWindow: time.Duration(coalesceMs) * time.Millisecond,
		maxMergeJobs:   maxMergeJobs,
		maxMergeChars:  maxMergeChars,
	}

	if queueSize > 0 {
		s.queue = make(chan transcriptJob, queueSize)
		s.startOnce.Do(func() {
			go s.runWorker()
		})
	}

	return s
}

func (s *TranscriptService) AcceptTranscript(
	params protocol.VoiceTranscriptParams,
) (protocol.VoiceTranscriptCompletion, bool, string) {
	cr := strings.TrimSpace(params.ControlRequest)
	if cr != "" {
		// UI cancel/abort: must not enter the coalescing queue (ordering vs later speech).
		params.Text = strings.TrimSpace(params.Text)
		params.ControlRequest = cr
		return s.runExecute(params)
	}

	params.Text = strings.TrimSpace(params.Text)
	if params.Text == "" {
		return protocol.VoiceTranscriptCompletion{}, false, ""
	}

	if s.queue == nil {
		return s.runExecute(params)
	}

	respCh := make(chan transcriptAcceptResp, 1)
	job := transcriptJob{
		params: params,
		resp:   respCh,
	}

	select {
	case s.queue <- job:
	default:
		job.resp <- transcriptAcceptResp{
			result: protocol.VoiceTranscriptCompletion{Success: false, UiDisposition: "hidden"},
			ok:     true,
			reason: "voice.transcript queue is full",
		}
	}

	resp := <-respCh
	return resp.result, resp.ok, resp.reason
}

func (s *TranscriptService) runExecute(params protocol.VoiceTranscriptParams) (protocol.VoiceTranscriptCompletion, bool, string) {
	s.executeMu.Lock()
	defer s.executeMu.Unlock()
	return run.Execute(&run.Env{
		Sessions:  s.sessions,
		Ephemeral: &s.ephemeralVoiceSession,
		Executor:  s.executor,
		HostApply: s.hostApplyClient,
	}, params)
}
