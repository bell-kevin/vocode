package edits

import protocol "vocoding.net/vocode/v2/packages/protocol/go"

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) ValidateAction(fileText string, action protocol.ReplaceBetweenAnchorsAction) *protocol.EditFailure {
	start, end, failure := findUniqueAnchoredRange(fileText, action.Anchor.Before, action.Anchor.After)
	if failure != nil {
		return failure
	}

	if end < start {
		return editFailure("validation_failed", "Resolved anchor range was invalid.")
	}

	return nil
}
