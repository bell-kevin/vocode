package executor

import (
	"context"
	"fmt"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch/edit"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func (e *Executor) runOneAgentLoopIteration(
	params protocol.VoiceTranscriptParams,
	text string,
	hostCursor *agentcontext.CursorSymbol,
	extSucceeded []intents.Intent,
	extFailed []agentcontext.FailedIntent,
	st *agentLoopState,
) (loopAdvance, protocol.VoiceTranscriptResult, bool) {
	succeededThisRPC := make([]intents.Intent, 0, len(extSucceeded)+len(st.completed))
	succeededThisRPC = append(succeededThisRPC, extSucceeded...)
	succeededThisRPC = append(succeededThisRPC, st.completed...)
	failedThisRPC := make([]agentcontext.FailedIntent, 0, len(extFailed)+len(st.failedIntents))
	failedThisRPC = append(failedThisRPC, extFailed...)
	failedThisRPC = append(failedThisRPC, st.failedIntents...)

	turnCtx := agentcontext.ComposeTurnContext(params, text, succeededThisRPC, failedThisRPC, st.gathered, hostCursor)
	next, err := e.agent.NextIntent(context.Background(), turnCtx)
	if err != nil {
		return advanceContinue, protocol.VoiceTranscriptResult{Accepted: false}, true
	}
	if err := next.Validate(); err != nil {
		return advanceContinue, protocol.VoiceTranscriptResult{Accepted: false}, true
	}

	var editCtx edit.EditExecutionContext
	if next.Executable != nil {
		var preExecErr string
		editCtx, preExecErr = buildEditExecutionContext(params, next.Executable)
		if preExecErr != "" {
			if st.maxRetries > 0 {
				st.failedIntents = append(st.failedIntents, agentcontext.FailedIntent{
					Intent: next,
					Phase:  agentcontext.PhasePreExecute,
					Reason: preExecErr,
				})
				st.gathered = appendGatheredNote(st.gathered, fmt.Sprintf("daemon rejected %q intent before execution: %s; retry with corrected intent", next.Executable.Kind, preExecErr))
				st.maxRetries--
				return advanceContinue, protocol.VoiceTranscriptResult{}, false
			}
			return advanceContinue, protocol.VoiceTranscriptResult{Accepted: false}, true
		}
	}

	if c := next.Control; c != nil && c.Kind == intents.ControlIntentKindRequestContext {
		st.contextRounds++
		st.consecutiveContextReq++
		if e.maxContextRounds > 0 && st.contextRounds > e.maxContextRounds {
			return advanceContinue, protocol.VoiceTranscriptResult{Accepted: false}, true
		}
		if e.maxConsecutiveContextReq > 0 && st.consecutiveContextReq > e.maxConsecutiveContextReq {
			return advanceContinue, protocol.VoiceTranscriptResult{Accepted: false}, true
		}
	}

	out, err := e.intentHandler.Handle(dispatch.HandleInput{
		Params:   params,
		Gathered: st.gathered,
		Intent:   next,
		EditCtx:  editCtx,
	})
	if err != nil {
		if next.Executable != nil {
			if st.maxRetries > 0 {
				st.failedIntents = append(st.failedIntents, agentcontext.FailedIntent{
					Intent: next,
					Phase:  agentcontext.PhaseDispatch,
					Reason: err.Error(),
				})
				st.gathered = appendGatheredNote(st.gathered, fmt.Sprintf("daemon execution failed for %q intent: %v; retry with corrected intent", next.Executable.Kind, err))
				st.maxRetries--
				return advanceContinue, protocol.VoiceTranscriptResult{}, false
			}
		}
		return advanceContinue, protocol.VoiceTranscriptResult{Accepted: false}, true
	}

	return e.applyHandleOutcome(out, next, st)
}
