package transcript

import (
	"strings"
	"sync"
	"time"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
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

	sessions *agentcontext.VoiceSessionStore
	// When contextSessionId is empty, full session (gathered, apply history, pending batch) is kept here.
	ephemeralVoiceSession agentcontext.VoiceSession

	executeMu sync.Mutex

	queue chan transcriptJob

	coalesceWindow time.Duration
	maxMergeJobs   int
	maxMergeChars  int
	startOnce      sync.Once
}

type transcriptJob struct {
	params protocol.VoiceTranscriptParams
	resp   chan transcriptAcceptResp
}

type transcriptAcceptResp struct {
	result protocol.VoiceTranscriptResult
	ok     bool
}

func NewService(
	agentRuntime *agent.Agent,
	intentHandler *dispatch.Handler,
	symbolResolver symbols.Resolver,
) *TranscriptService {
	queueSize := config.Int("VOCODE_DAEMON_VOICE_TRANSCRIPT_QUEUE_SIZE", 10)
	coalesceMs := config.Int("VOCODE_DAEMON_VOICE_TRANSCRIPT_COALESCE_MS", 750)
	maxMergeJobs := config.Int("VOCODE_DAEMON_VOICE_TRANSCRIPT_MAX_MERGE_JOBS", 5)
	maxMergeChars := config.Int("VOCODE_DAEMON_VOICE_TRANSCRIPT_MAX_MERGE_CHARS", 6000)
	maxAgentTurns := config.Int("VOCODE_DAEMON_VOICE_MAX_AGENT_TURNS", 8)
	maxIntentRetries := config.Int("VOCODE_DAEMON_VOICE_MAX_INTENT_RETRIES", 2)
	maxContextRounds := config.Int("VOCODE_DAEMON_VOICE_MAX_CONTEXT_ROUNDS", 2)
	maxContextBytes := config.Int("VOCODE_DAEMON_VOICE_MAX_CONTEXT_BYTES", 12000)
	maxConsecutiveContextReq := config.Int("VOCODE_DAEMON_VOICE_MAX_CONSECUTIVE_CONTEXT_REQUESTS", 3)
	maxIntentsPerBatch := config.Int("VOCODE_DAEMON_VOICE_MAX_INTENTS_PER_BATCH", 16)

	exec := executor.New(agentRuntime, intentHandler, executor.Options{
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
