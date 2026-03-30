package executor

import (
	"fmt"

	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func (e *Executor) applyDirective(out dispatch.Directive, next intents.Intent, st *agentLoopState, caps ExecutionCaps) (loopAdvance, protocol.VoiceTranscriptResult, bool) {
	if out.IsEmpty() {
		return advanceContinue, protocol.VoiceTranscriptResult{Success: false}, true
	}
	if err := out.Validate(); err != nil {
		return advanceContinue, protocol.VoiceTranscriptResult{Success: false}, true
	}

	st.maxRetries = caps.MaxIntentRetries
	if st.maxRetries < 0 {
		st.maxRetries = 0
	}
	st.failedIntents = nil
	st.consecutiveContextReq = 0

	switch out.Kind {
	case dispatch.DirectiveKindEdit:
		ed := out.EditDirective
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
	case dispatch.DirectiveKindCommand:
		st.directives = append(st.directives, protocol.VoiceTranscriptDirective{Kind: "command", CommandDirective: out.CommandDirective})
		st.completed = append(st.completed, next)
		appendSourceIntentForDirective(&st.batchSourceIntents, next)
	case dispatch.DirectiveKindNavigate:
		st.directives = append(st.directives, protocol.VoiceTranscriptDirective{Kind: "navigate", NavigationDirective: out.NavigationDirective})
		st.completed = append(st.completed, next)
		appendSourceIntentForDirective(&st.batchSourceIntents, next)
	case dispatch.DirectiveKindUndo:
		st.directives = append(st.directives, protocol.VoiceTranscriptDirective{Kind: "undo", UndoDirective: out.UndoDirective})
		st.completed = append(st.completed, next)
		appendSourceIntentForDirective(&st.batchSourceIntents, next)
	default:
		return advanceContinue, protocol.VoiceTranscriptResult{Success: false}, true
	}
	return advanceBatchIntentDone, protocol.VoiceTranscriptResult{}, false
}
