package transcript

import (
	"log"
	"strings"
	"sync"
	"time"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/gather"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch"
	"vocoding.net/vocode/v2/apps/daemon/internal/symbols"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/config"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/executor"
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
	result protocol.VoiceTranscriptResult
	ok     bool
}

func NewService(
	agentRuntime *agent.Agent,
	intentHandler *dispatch.Handler,
	gatherProvider *gather.Provider,
	symbolResolver symbols.Resolver,
	logger *log.Logger,
) *TranscriptService {
	queueSize := config.DefaultTranscriptQueueSize
	coalesceMs := config.DefaultTranscriptCoalesceMs
	maxMergeJobs := config.DefaultTranscriptMaxMergeJobs
	maxMergeChars := config.DefaultTranscriptMaxMergeChars
	maxAgentTurns := config.DefaultMaxAgentTurns
	maxIntentRetries := config.DefaultMaxIntentRetries
	maxContextRounds := config.DefaultMaxContextRounds
	maxContextBytes := config.DefaultMaxContextBytes
	maxConsecutiveContextReq := config.DefaultMaxConsecutiveContextReq
	maxIntentsPerBatch := config.DefaultMaxIntentsPerBatch

	exec := executor.New(agentRuntime, intentHandler, gatherProvider, executor.Options{
		MaxAgentTurns:            maxAgentTurns,
		MaxIntentRetries:         maxIntentRetries,
		MaxContextRounds:         maxContextRounds,
		MaxContextBytes:          maxContextBytes,
		MaxConsecutiveContextReq: maxConsecutiveContextReq,
		MaxIntentsPerBatch:       maxIntentsPerBatch,
		Symbols:                  symbolResolver,
	})

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
) (protocol.VoiceTranscriptResult, bool) {
	params.Text = strings.TrimSpace(params.Text)
	if params.Text == "" {
		return protocol.VoiceTranscriptResult{}, false
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
			result: protocol.VoiceTranscriptResult{
				Success: false,
			},
			ok: true,
		}
	}

	resp := <-respCh
	return resp.result, resp.ok
}
