package edits

import protocol "vocoding.net/vocode/v2/packages/protocol/go"

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) ValidateAction(
	fileText string,
	action protocol.ReplaceBetweenAnchorsAction,
) *EditBuildFailure {
	start, end, failure := findUniqueAnchoredRange(fileText, action.Anchor.Before, action.Anchor.After)
	if failure != nil {
		return failure
	}

	if end < start {
		return &EditBuildFailure{Code: "validation_failed", Message: "Resolved anchor range was invalid."}
	}

	return nil
}
