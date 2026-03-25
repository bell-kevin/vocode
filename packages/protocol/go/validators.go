package protocol

import "errors"

// EditApplyResult validation lives here alongside future protocol-level validators
// (mirrors typescript/validators.ts conceptually).

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

func (r VoiceTranscriptResult) Validate() error {
	if !r.Accepted {
		return errors.New("voice transcript result must have accepted=true")
	}

	return nil
}
