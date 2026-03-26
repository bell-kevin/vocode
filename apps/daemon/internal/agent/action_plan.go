package agent

import (
	"fmt"
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// StepKind discriminates one entry in an [ActionPlan].
type StepKind string

const (
	StepKindEdit       StepKind = "edit"
	StepKindRunCommand StepKind = "run_command"
)

// Step is one concrete action in the plan: either apply a structured edit, or
// run a shell command. It is the union of [EditIntent] and [RunCommandIntent]
// with an explicit kind.
type Step struct {
	Kind StepKind `json:"kind"`

	Edit       *EditIntent       `json:"edit,omitempty"`
	RunCommand *RunCommandIntent `json:"runCommand,omitempty"`
}

// ActionPlan is structured model output: an ordered list of edits and/or
// commands to run. The client supplies file snapshot context for edit steps
// via [protocol.EditApplyParams].
//
// JSON shape matches packages/protocol/schema/action-plan.schema.json.
type ActionPlan struct {
	Steps []Step `json:"steps"`
}

// RunCommandIntent maps to protocol.CommandRunParams.
type RunCommandIntent struct {
	Command   string   `json:"command"`
	Args      []string `json:"args,omitempty"`
	TimeoutMs *int64   `json:"timeoutMs,omitempty"`
}

// CommandParams returns protocol params for command execution.
func (i RunCommandIntent) CommandParams() protocol.CommandRunParams {
	return protocol.CommandRunParams{
		Command:   strings.TrimSpace(i.Command),
		Args:      i.Args,
		TimeoutMs: i.TimeoutMs,
	}
}

// ValidateStep checks a single step is self-consistent.
func ValidateStep(s Step) error {
	switch s.Kind {
	case StepKindEdit:
		if s.RunCommand != nil {
			return fmt.Errorf("step: kind %q must not include runCommand", s.Kind)
		}
		if s.Edit == nil {
			return fmt.Errorf("step: kind %q requires edit", s.Kind)
		}
		return ValidateEditIntent(*s.Edit)
	case StepKindRunCommand:
		if s.Edit != nil {
			return fmt.Errorf("step: kind %q must not include edit", s.Kind)
		}
		if s.RunCommand == nil {
			return fmt.Errorf("step: kind %q requires runCommand", s.Kind)
		}
		if strings.TrimSpace(s.RunCommand.Command) == "" {
			return fmt.Errorf("step: runCommand.command is empty")
		}
		return nil
	default:
		return fmt.Errorf("step: unknown kind %q", s.Kind)
	}
}

// ValidateActionPlan validates every step. Empty steps is allowed (no-op plan).
func ValidateActionPlan(p ActionPlan) error {
	for i := range p.Steps {
		if err := ValidateStep(p.Steps[i]); err != nil {
			return fmt.Errorf("action plan: step %d: %w", i, err)
		}
	}
	return nil
}
