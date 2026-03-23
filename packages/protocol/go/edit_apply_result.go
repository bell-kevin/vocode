package protocol

import "errors"

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

func (r EditApplyResult) Validate() error {
	switch r.Kind {
	case "success":
		if r.Actions == nil {
			return errors.New("success result must include actions")
		}
		if r.Failure != nil || r.Reason != "" {
			return errors.New("success result must not contain failure or reason")
		}
	case "failure":
		if r.Failure == nil {
			return errors.New("failure result must include failure")
		}
		if len(r.Actions) > 0 || r.Reason != "" {
			return errors.New("failure result must not contain actions or reason")
		}
	case "noop":
		if r.Reason == "" {
			return errors.New("noop result must include reason")
		}
		if len(r.Actions) > 0 || r.Failure != nil {
			return errors.New("noop result must not contain actions or failure")
		}
	default:
		return errors.New("unknown edit/apply result kind")
	}

	return nil
}
