package protocol

func NewEditApplySuccess(actions []EditAction) EditApplyResult {
	return EditApplyResult{
		Kind:    "success",
		Actions: actions,
	}
}

func NewEditApplyFailure(failure EditFailure) EditApplyResult {
	return EditApplyResult{
		Kind:    "failure",
		Failure: &failure,
	}
}

func NewEditApplyNoop(reason string) EditApplyResult {
	return EditApplyResult{
		Kind:   "noop",
		Reason: reason,
	}
}
