package executor

import (
	"fmt"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func (e *Executor) applyHandleOutcome(out dispatch.HandleOutcome, next intents.Intent, st *agentLoopState) (loopAdvance, protocol.VoiceTranscriptResult, bool) {
	switch {
	case out.Control != nil:
		cr := out.Control
		if cr.Done != nil {
			st.transcriptSummary = cr.Done.Summary
			return advanceBreakLoop, protocol.VoiceTranscriptResult{}, false
		}
		if cr.Fulfilled != nil {
			updated := cr.Fulfilled.UpdatedGathered
			if e.maxContextBytes > 0 && agentcontext.EstimatedGatheredBytes(updated) > e.maxContextBytes {
				return advanceContinue, protocol.VoiceTranscriptResult{Accepted: false}, true
			}
			st.gathered = updated
			st.completed = append(st.completed, next)
			return advanceContinue, protocol.VoiceTranscriptResult{}, false
		}
		return advanceContinue, protocol.VoiceTranscriptResult{Accepted: false}, true

	case out.Executable != nil:
		st.maxRetries = e.maxIntentRetries
		if st.maxRetries < 0 {
			st.maxRetries = 0
		}
		st.failedIntents = nil
		st.consecutiveContextReq = 0
		ex := out.Executable
		switch {
		case ex.EditDirective != nil:
			ed := ex.EditDirective
			if ed.Kind == "success" {
				for j := range ed.Actions {
					if ed.Actions[j].EditId == "" {
						ed.Actions[j].EditId = fmt.Sprintf("edit-%d", st.editCounter)
						st.editCounter++
					}
				}
			}
			st.directives = append(st.directives, protocol.VoiceTranscriptDirective{Kind: "edit", EditDirective: ed})
			st.completed = append(st.completed, next)
			appendSourceIntentForDirective(&st.batchSourceIntents, next)
		case ex.CommandDirective != nil:
			st.directives = append(st.directives, protocol.VoiceTranscriptDirective{Kind: "command", CommandDirective: ex.CommandDirective})
			st.completed = append(st.completed, next)
			appendSourceIntentForDirective(&st.batchSourceIntents, next)
		case ex.NavigationDirective != nil:
			st.directives = append(st.directives, protocol.VoiceTranscriptDirective{Kind: "navigate", NavigationDirective: ex.NavigationDirective})
			st.completed = append(st.completed, next)
			appendSourceIntentForDirective(&st.batchSourceIntents, next)
		case ex.UndoDirective != nil:
			st.directives = append(st.directives, protocol.VoiceTranscriptDirective{Kind: "undo", UndoDirective: ex.UndoDirective})
			st.completed = append(st.completed, next)
			appendSourceIntentForDirective(&st.batchSourceIntents, next)
		default:
			return advanceContinue, protocol.VoiceTranscriptResult{Accepted: false}, true
		}
		return advanceContinue, protocol.VoiceTranscriptResult{}, false

	default:
		return advanceContinue, protocol.VoiceTranscriptResult{Accepted: false}, true
	}
}
