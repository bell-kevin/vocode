package transcript

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/dispatch"
	"vocoding.net/vocode/v2/apps/daemon/internal/indexing"
	"vocoding.net/vocode/v2/apps/daemon/internal/intentloop"
	"vocoding.net/vocode/v2/apps/daemon/internal/symbols"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// TranscriptService adapts the voice.transcript RPC to the agent runtime and
// action-plan execution (edits via structured results for the extension;
// commands on the daemon).
type TranscriptService struct {
	loop *intentloop.Runner

	queue chan transcriptJob

	coalesceWindow time.Duration
	maxMergeJobs   int
	maxMergeChars  int
	startOnce sync.Once
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
	dispatch *dispatch.Dispatcher,
) *TranscriptService {
	queueSize := envInt("VOCODE_DAEMON_VOICE_TRANSCRIPT_QUEUE_SIZE", 10)
	coalesceMs := envInt("VOCODE_DAEMON_VOICE_TRANSCRIPT_COALESCE_MS", 750)
	maxMergeJobs := envInt("VOCODE_DAEMON_VOICE_TRANSCRIPT_MAX_MERGE_JOBS", 5)
	maxMergeChars := envInt("VOCODE_DAEMON_VOICE_TRANSCRIPT_MAX_MERGE_CHARS", 6000)
	maxPlannerTurns := envInt("VOCODE_DAEMON_VOICE_MAX_PLANNER_TURNS", 8)
	maxIntentRetries := envInt("VOCODE_DAEMON_VOICE_MAX_INTENT_RETRIES", 2)
	maxContextRounds := envInt("VOCODE_DAEMON_VOICE_MAX_CONTEXT_ROUNDS", 2)
	maxContextBytes := envInt("VOCODE_DAEMON_VOICE_MAX_CONTEXT_BYTES", 12000)
	maxConsecutiveContextReq := envInt("VOCODE_DAEMON_VOICE_MAX_CONSECUTIVE_CONTEXT_REQUESTS", 3)

	symbolResolver := symbols.NewTreeSitterResolver()
	cp := indexing.NewContextProvider(symbolResolver)
	loop := intentloop.NewRunner(agentRuntime, dispatch, cp, intentloop.Options{
		MaxPlannerTurns:          maxPlannerTurns,
		MaxIntentRetries:         maxIntentRetries,
		MaxContextRounds:         maxContextRounds,
		MaxContextBytes:          maxContextBytes,
		MaxConsecutiveContextReq: maxConsecutiveContextReq,
	})

	s := &TranscriptService{
		loop:          loop,
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

	// If the queue is disabled, execute immediately.
	if s.queue == nil {
		return s.loop.Execute(params)
	}

	respCh := make(chan transcriptAcceptResp, 1)
	job := transcriptJob{
		params: params,
		resp:   respCh,
	}

	// Keep RPC calls responsive: if the queue is full, reject quickly.
	select {
	case s.queue <- job:
	default:
		job.resp <- transcriptAcceptResp{
			result: protocol.VoiceTranscriptResult{
				Accepted: false,
			},
			ok: true,
		}
	}

	resp := <-respCh
	return resp.result, resp.ok
}

func (s *TranscriptService) runWorker() {
	// buffered contains jobs that couldn't be merged into the current coalescing batch,
	// but must preserve FIFO order for later processing.
	buffered := make([]transcriptJob, 0, cap(s.queue))

	for {
		primary := func() transcriptJob {
			if len(buffered) > 0 {
				j := buffered[0]
				buffered = buffered[1:]
				return j
			}
			return <-s.queue
		}()

		baseActiveFile := strings.TrimSpace(primary.params.ActiveFile)

		batch := []transcriptJob{primary}
		mergedTextParts := []string{primary.params.Text}
		mergedChars := len(primary.params.Text)

		timer := time.NewTimer(s.coalesceWindow)

		for collecting := true; collecting; {
			select {
			case j := <-s.queue:
				activeFile := strings.TrimSpace(j.params.ActiveFile)
				text := strings.TrimSpace(j.params.Text)
				if text == "" {
					// Invalid/empty transcripts should just no-op.
					j.resp <- transcriptAcceptResp{
						result: protocol.VoiceTranscriptResult{Accepted: true},
						ok:     true,
					}
					continue
				}

				eligible := activeFile == baseActiveFile &&
					len(batch) < s.maxMergeJobs &&
					mergedChars+1+len(text) <= s.maxMergeChars

				if eligible {
					j.params.Text = text
					batch = append(batch, j)
					mergedTextParts = append(mergedTextParts, text)
					mergedChars += 1 + len(text)
				} else {
					buffered = append(buffered, j)
				}
			case <-timer.C:
				collecting = false
			}
		}

		timer.Stop()

		mergedParams := primary.params
		mergedParams.Text = strings.Join(mergedTextParts, " ")

		mergedResult, ok := s.loop.Execute(mergedParams)

		// Respond: only the primary job returns the actual execution.
		// Coalesced jobs succeed with an empty result to avoid duplicate UI edits.
		for i, j := range batch {
			if i == 0 {
				j.resp <- transcriptAcceptResp{
					result: mergedResult,
					ok:     ok,
				}
			} else {
				j.resp <- transcriptAcceptResp{
					result: protocol.VoiceTranscriptResult{Accepted: true},
					ok:     true,
				}
			}
		}
	}
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

