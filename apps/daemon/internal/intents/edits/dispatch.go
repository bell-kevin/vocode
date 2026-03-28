package edits

import (
	"fmt"

	"vocoding.net/vocode/v2/apps/daemon/internal/intent"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// DispatchEdit validates [intent.EditIntent], runs action building, and returns a protocol
// [protocol.EditDirective] (success or noop).
func (e *Engine) DispatchEdit(ctx EditExecutionContext, editIntent intent.EditIntent) (protocol.EditDirective, error) {
	if err := intent.ValidateEditIntent(editIntent); err != nil {
		return protocol.EditDirective{}, err
	}
	actions, failure := e.BuildActions(ctx, editIntent)
	if failure != nil {
		if failure.Code == "no_change_needed" {
			result := protocol.NewEditDirectiveNoop(failure.Message)
			return result, result.Validate()
		}
		return protocol.EditDirective{}, fmt.Errorf("edit dispatch failed: %s", failure.Message)
	}
	result := protocol.NewEditDirectiveSuccess(actions)
	return result, result.Validate()
}
