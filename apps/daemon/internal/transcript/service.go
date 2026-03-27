package transcript

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"vocoding.net/vocode/v2/apps/daemon/internal/actionplan"
	"vocoding.net/vocode/v2/apps/daemon/internal/actionplan/dispatch"
	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/edits"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// TranscriptService adapts the voice.transcript RPC to the agent runtime and
// action-plan execution (edits via structured results for the extension;
// commands on the daemon).
type TranscriptService struct {
	agent    *agent.Agent
	dispatch *dispatch.Dispatcher

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

	s := &TranscriptService{
		agent:          agentRuntime,
		dispatch:       dispatch,
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
		return s.acceptTranscriptDirect(params)
	}

	respCh := make(chan transcriptAcceptResp, 1)
	job := transcriptJob{
		params: params,
		resp:   respCh,
	}

	// Keep RPC calls responsive: if the queue is full, succeed with a planError
	// so the extension can ignore it without crashing the session.
	select {
	case s.queue <- job:
	default:
		job.resp <- transcriptAcceptResp{
			result: protocol.VoiceTranscriptResult{
				Accepted:  true,
				PlanError: "transcript queue full",
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

		mergedResult, ok := s.acceptTranscriptDirect(mergedParams)

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

func (s *TranscriptService) acceptTranscriptDirect(
	params protocol.VoiceTranscriptParams,
) (protocol.VoiceTranscriptResult, bool) {
	r := s.agent.HandleTranscript(context.Background(), params)
	if !r.Valid {
		return protocol.VoiceTranscriptResult{}, false
	}
	if r.Err != nil {
		return protocol.VoiceTranscriptResult{
			Accepted:  true,
			PlanError: r.Err.Error(),
		}, true
	}
	if r.Plan == nil {
		return protocol.VoiceTranscriptResult{
			Accepted:  true,
			PlanError: "no plan returned",
		}, true
	}

	editParams, planErr := buildEditApplyParams(params, r.Plan)
	if planErr != "" {
		return protocol.VoiceTranscriptResult{
			Accepted:  true,
			PlanError: planErr,
		}, true
	}

	execResult, err := s.dispatch.Execute(*r.Plan, editParams)
	if err != nil {
		return protocol.VoiceTranscriptResult{
			Accepted:  true,
			PlanError: err.Error(),
		}, true
	}

	steps := make([]protocol.VoiceTranscriptStepResult, 0, len(execResult.Steps))
	for _, st := range execResult.Steps {
		switch {
		case st.EditResult != nil:
			steps = append(steps, protocol.VoiceTranscriptStepResult{
				Kind:       "edit",
				EditResult: st.EditResult,
			})
		case st.CommandParams != nil:
			steps = append(steps, protocol.VoiceTranscriptStepResult{
				Kind:          "run_command",
				CommandParams: st.CommandParams,
			})
		case st.Navigation != nil:
			steps = append(steps, protocol.VoiceTranscriptStepResult{
				Kind:             "navigate",
				NavigationIntent: toProtocolNavigationIntent(*st.Navigation),
			})
		}
	}

	result := protocol.VoiceTranscriptResult{
		Accepted: true,
		Steps:    steps,
	}
	if err := result.Validate(); err != nil {
		return protocol.VoiceTranscriptResult{
			Accepted:  true,
			PlanError: err.Error(),
		}, true
	}
	return result, true
}

func planHasEditStep(p *actionplan.ActionPlan) bool {
	for _, s := range p.Steps {
		if s.Kind == actionplan.StepKindEdit {
			return true
		}
	}
	return false
}

// buildEditApplyParams loads file text on the daemon when activeFile is set.
// Unsaved editor buffers are not visible until workspace indexing supplies them.
func buildEditApplyParams(params protocol.VoiceTranscriptParams, plan *actionplan.ActionPlan) (edits.EditExecutionContext, string) {
	active := strings.TrimSpace(params.ActiveFile)
	if planHasEditStep(plan) && active == "" {
		return edits.EditExecutionContext{}, "activeFile is required when the plan includes edit steps"
	}
	fileText := ""
	if active != "" {
		b, err := os.ReadFile(active)
		if err != nil {
			return edits.EditExecutionContext{}, fmt.Sprintf("read active file: %v", err)
		}
		fileText = string(b)
	}
	return edits.EditExecutionContext{
		Instruction: params.Text,
		ActiveFile:  params.ActiveFile,
		FileText:    fileText,
	}, ""
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

func toProtocolNavigationIntent(n actionplan.NavigationIntent) *protocol.NavigationIntent {
	var out protocol.NavigationIntent
	b, err := json.Marshal(n)
	if err != nil {
		return &out
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return &out
	}
	return &out
}
