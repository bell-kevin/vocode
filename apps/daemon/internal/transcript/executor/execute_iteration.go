package executor

import (
	"context"
	"fmt"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/gather"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func (e *Executor) runOneAgentLoopIteration(
	params protocol.VoiceTranscriptParams,
	text string,
	hostCursor *agentcontext.CursorSymbol,
	intentApplyHistory []agentcontext.IntentApplyRecord,
	extSucceeded []intents.Intent,
	extFailed []agentcontext.FailedIntent,
	extSkipped []intents.Intent,
	st *agentLoopState,
	caps ExecutionCaps,
) (loopAdvance, protocol.VoiceTranscriptCompletion, bool, string) {
	succeededThisRPC := make([]intents.Intent, 0, len(extSucceeded)+len(st.completed))
	succeededThisRPC = append(succeededThisRPC, extSucceeded...)
	succeededThisRPC = append(succeededThisRPC, st.completed...)
	failedThisRPC := make([]agentcontext.FailedIntent, 0, len(extFailed)+len(st.failedIntents))
	failedThisRPC = append(failedThisRPC, extFailed...)
	failedThisRPC = append(failedThisRPC, st.failedIntents...)

	turnCtx := agentcontext.ComposeTurnContext(
		params,
		text,
		succeededThisRPC,
		failedThisRPC,
		extSkipped,
		intentApplyHistory,
		st.gathered,
		hostCursor,
		agentcontext.TurnLimits{MaxContextRounds: caps.MaxContextRounds},
	)

	turn, err := e.agent.NextTurn(context.Background(), turnCtx)
	if err != nil {
		return advanceContinue, protocol.VoiceTranscriptCompletion{Success: false}, true, fmt.Sprintf("model.NextTurn failed: %v", err)
	}
	if err := turn.Validate(); err != nil {
		return advanceContinue, protocol.VoiceTranscriptCompletion{Success: false}, true, fmt.Sprintf("model returned invalid turn JSON: %v", err)
	}

	switch turn.Kind {
	case agent.TurnIrrelevant:
		st.transcriptSummary = strings.TrimSpace(turn.IrrelevantReason)
		st.transcriptOutcome = "irrelevant"
		return advanceBreakLoop, protocol.VoiceTranscriptCompletion{}, false, ""

	case agent.TurnFinish:
		st.transcriptSummary = strings.TrimSpace(turn.FinishSummary)
		return advanceBreakLoop, protocol.VoiceTranscriptCompletion{}, false, ""

	case agent.TurnGatherContext:
		st.contextRounds++
		st.consecutiveContextReq++
		if caps.MaxContextRounds > 0 && st.contextRounds > caps.MaxContextRounds {
			return advanceContinue, protocol.VoiceTranscriptCompletion{Success: false}, true, fmt.Sprintf("maxContextRounds exceeded: %d > %d", st.contextRounds, caps.MaxContextRounds)
		}
		if caps.MaxConsecutiveContextReq > 0 && st.consecutiveContextReq > caps.MaxConsecutiveContextReq {
			return advanceContinue, protocol.VoiceTranscriptCompletion{Success: false}, true, fmt.Sprintf("maxConsecutiveContextRequests exceeded: %d > %d", st.consecutiveContextReq, caps.MaxConsecutiveContextReq)
		}
		updated, err := gather.FulfillSpec(e.gather, params, st.gathered, turn.GatherContext)
		if err != nil {
			return advanceContinue, protocol.VoiceTranscriptCompletion{Success: false}, true, fmt.Sprintf("gather context failed: %v", err)
		}
		if caps.MaxContextBytes > 0 && agentcontext.EstimatedGatheredBytes(updated) > caps.MaxContextBytes {
			return advanceContinue, protocol.VoiceTranscriptCompletion{Success: false}, true, fmt.Sprintf("maxContextBytes exceeded: gatheredBytes=%d > %d", agentcontext.EstimatedGatheredBytes(updated), caps.MaxContextBytes)
		}
		st.gathered = updated
		return advanceContinue, protocol.VoiceTranscriptCompletion{}, false, ""

	case agent.TurnIntents:
		if caps.MaxIntentsPerBatch > 0 && len(turn.Intents) > caps.MaxIntentsPerBatch {
			return advanceContinue, protocol.VoiceTranscriptCompletion{Success: false}, true, fmt.Sprintf("maxIntentsPerBatch exceeded: %d > %d", len(turn.Intents), caps.MaxIntentsPerBatch)
		}
		for i := range turn.Intents {
			adv, res, abort, reason := e.dispatchOneIntent(params, hostCursor, st, turn.Intents[i], caps)
			if abort {
				return adv, res, true, reason
			}
			switch adv {
			case advanceContinue:
				return advanceContinue, protocol.VoiceTranscriptCompletion{}, false, ""
			case advanceBreakLoop:
				return advanceBreakLoop, protocol.VoiceTranscriptCompletion{}, false, ""
			case advanceBatchIntentDone:
				continue
			default:
				return advanceContinue, protocol.VoiceTranscriptCompletion{Success: false}, true, fmt.Sprintf("unknown loop advance state: %d", adv)
			}
		}
		return advanceBreakLoop, protocol.VoiceTranscriptCompletion{}, false, ""

	default:
		return advanceContinue, protocol.VoiceTranscriptCompletion{Success: false}, true, fmt.Sprintf("unknown turn kind %q", turn.Kind)
	}
}

func (e *Executor) dispatchOneIntent(
	params protocol.VoiceTranscriptParams,
	hostCursor *agentcontext.CursorSymbol,
	st *agentLoopState,
	next intents.Intent,
	caps ExecutionCaps,
) (loopAdvance, protocol.VoiceTranscriptCompletion, bool, string) {
	editCtx, preExecErr := buildEditExecutionContext(params, &next)
	if preExecErr != "" {
		if st.maxRetries > 0 {
			st.failedIntents = append(st.failedIntents, agentcontext.FailedIntent{
				Intent: next,
				Phase:  agentcontext.PhasePreExecute,
				Reason: preExecErr,
			})
			st.gathered = appendGatheredNote(st.gathered, fmt.Sprintf("daemon rejected %q intent before execution: %s; retry with corrected intent", next.Kind, preExecErr))
			st.maxRetries--
			return advanceContinue, protocol.VoiceTranscriptCompletion{}, false, ""
		}
		return advanceContinue, protocol.VoiceTranscriptCompletion{Success: false}, true, fmt.Sprintf("intent rejected before execution (%s): %s", next.Kind, preExecErr)
	}

	out, err := e.intentHandler.Handle(dispatch.HandleInput{
		Params:  params,
		Intent:  next,
		EditCtx: editCtx,
	})
	if err != nil {
		if st.maxRetries > 0 {
			st.failedIntents = append(st.failedIntents, agentcontext.FailedIntent{
				Intent: next,
				Phase:  agentcontext.PhaseDispatch,
				Reason: err.Error(),
			})
			st.gathered = appendGatheredNote(st.gathered, fmt.Sprintf("daemon execution failed for %q intent: %v; retry with corrected intent", next.Kind, err))
			st.maxRetries--
			return advanceContinue, protocol.VoiceTranscriptCompletion{}, false, ""
		}
		return advanceContinue, protocol.VoiceTranscriptCompletion{Success: false}, true, fmt.Sprintf("intent dispatch failed (%s): %v", next.Kind, err)
	}

	adv, res, abort, reason := e.applyDirective(out, next, st, caps)
	return adv, res, abort, reason
}
