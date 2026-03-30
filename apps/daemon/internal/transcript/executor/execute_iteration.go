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
) (loopAdvance, protocol.VoiceTranscriptResult, bool) {
	succeededThisRPC := make([]intents.Intent, 0, len(extSucceeded)+len(st.completed))
	succeededThisRPC = append(succeededThisRPC, extSucceeded...)
	succeededThisRPC = append(succeededThisRPC, st.completed...)
	failedThisRPC := make([]agentcontext.FailedIntent, 0, len(extFailed)+len(st.failedIntents))
	failedThisRPC = append(failedThisRPC, extFailed...)
	failedThisRPC = append(failedThisRPC, st.failedIntents...)

	turnCtx := agentcontext.ComposeTurnContext(
		params, text, succeededThisRPC, failedThisRPC, extSkipped, intentApplyHistory, st.gathered, hostCursor)

	turn, err := e.agent.NextTurn(context.Background(), turnCtx)
	if err != nil {
		return advanceContinue, protocol.VoiceTranscriptResult{Success: false}, true
	}
	if err := turn.Validate(); err != nil {
		return advanceContinue, protocol.VoiceTranscriptResult{Success: false}, true
	}

	switch turn.Kind {
	case agent.TurnIrrelevant:
		st.transcriptSummary = strings.TrimSpace(turn.IrrelevantReason)
		return advanceBreakLoop, protocol.VoiceTranscriptResult{}, false

	case agent.TurnFinish:
		st.transcriptSummary = strings.TrimSpace(turn.FinishSummary)
		return advanceBreakLoop, protocol.VoiceTranscriptResult{}, false

	case agent.TurnGatherContext:
		st.contextRounds++
		st.consecutiveContextReq++
		if caps.MaxContextRounds > 0 && st.contextRounds > caps.MaxContextRounds {
			return advanceContinue, protocol.VoiceTranscriptResult{Success: false}, true
		}
		if caps.MaxConsecutiveContextReq > 0 && st.consecutiveContextReq > caps.MaxConsecutiveContextReq {
			return advanceContinue, protocol.VoiceTranscriptResult{Success: false}, true
		}
		updated, err := gather.FulfillSpec(e.gather, params, st.gathered, turn.GatherContext)
		if err != nil {
			return advanceContinue, protocol.VoiceTranscriptResult{Success: false}, true
		}
		if caps.MaxContextBytes > 0 && agentcontext.EstimatedGatheredBytes(updated) > caps.MaxContextBytes {
			return advanceContinue, protocol.VoiceTranscriptResult{Success: false}, true
		}
		st.gathered = updated
		return advanceContinue, protocol.VoiceTranscriptResult{}, false

	case agent.TurnIntents:
		if caps.MaxIntentsPerBatch > 0 && len(turn.Intents) > caps.MaxIntentsPerBatch {
			return advanceContinue, protocol.VoiceTranscriptResult{Success: false}, true
		}
		for i := range turn.Intents {
			adv, res, abort := e.dispatchOneIntent(params, hostCursor, st, turn.Intents[i], caps)
			if abort {
				return adv, res, true
			}
			switch adv {
			case advanceContinue:
				return advanceContinue, protocol.VoiceTranscriptResult{}, false
			case advanceBreakLoop:
				return advanceBreakLoop, protocol.VoiceTranscriptResult{}, false
			case advanceBatchIntentDone:
				continue
			default:
				return advanceContinue, protocol.VoiceTranscriptResult{Success: false}, true
			}
		}
		return advanceBreakLoop, protocol.VoiceTranscriptResult{}, false

	default:
		return advanceContinue, protocol.VoiceTranscriptResult{Success: false}, true
	}
}

func (e *Executor) dispatchOneIntent(
	params protocol.VoiceTranscriptParams,
	hostCursor *agentcontext.CursorSymbol,
	st *agentLoopState,
	next intents.Intent,
	caps ExecutionCaps,
) (loopAdvance, protocol.VoiceTranscriptResult, bool) {
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
			return advanceContinue, protocol.VoiceTranscriptResult{}, false
		}
		return advanceContinue, protocol.VoiceTranscriptResult{Success: false}, true
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
			return advanceContinue, protocol.VoiceTranscriptResult{}, false
		}
		return advanceContinue, protocol.VoiceTranscriptResult{Success: false}, true
	}

	return e.applyDirective(out, next, st, caps)
}
