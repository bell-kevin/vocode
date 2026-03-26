package protocol

import (
	"errors"
	"fmt"
)

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
		return errors.New("unknown edit.apply result kind")
	}

	return nil
}

func (s VoiceTranscriptStepResult) Validate() error {
	switch s.Kind {
	case "edit":
		if s.EditResult == nil || s.CommandResult != nil {
			return errors.New("voice transcript step: kind edit requires editResult and no commandResult")
		}
		return s.EditResult.Validate()
	case "run_command":
		if s.CommandResult == nil || s.EditResult != nil {
			return errors.New("voice transcript step: kind run_command requires commandResult and no editResult")
		}
		return s.CommandResult.Validate()
	default:
		return fmt.Errorf("voice transcript step: unknown kind %q", s.Kind)
	}
}

func (r VoiceTranscriptResult) Validate() error {
	if !r.Accepted {
		return errors.New("voice transcript result must have accepted=true")
	}
	if r.PlanError != "" && len(r.Steps) > 0 {
		return errors.New("voice transcript result must not include both planError and steps")
	}
	for i := range r.Steps {
		if err := r.Steps[i].Validate(); err != nil {
			return fmt.Errorf("voice transcript result steps[%d]: %w", i, err)
		}
	}
	return nil
}

func (r CommandRunResult) Validate() error {
	switch r.Kind {
	case "success":
		if r.Failure != nil {
			return errors.New("command.run success result must not include failure")
		}
		if r.ExitCode == nil {
			return errors.New("command.run success result must include exitCode")
		}
	case "failure":
		if r.Failure == nil {
			return errors.New("command.run failure result must include failure")
		}
		if r.ExitCode != nil {
			return errors.New("command.run failure result must not include exitCode")
		}
	default:
		return errors.New("unknown command.run result kind")
	}

	return nil
}
