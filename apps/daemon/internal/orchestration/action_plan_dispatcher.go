package orchestration

import (
	"fmt"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/commandexec"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// ActionPlanDispatcher runs a validated [agent.ActionPlan] step-by-step.
type ActionPlanDispatcher struct {
	edits   *EditApplyPipeline
	command *commandexec.Service
}

func NewActionPlanDispatcher(edits *EditApplyPipeline, command *commandexec.Service) *ActionPlanDispatcher {
	return &ActionPlanDispatcher{edits: edits, command: command}
}

// StepResult is the outcome of executing one [agent.Step] (exactly one pointer
// is non-nil).
type StepResult struct {
	EditResult    *protocol.EditApplyResult
	CommandResult *protocol.CommandRunResult
}

// ActionPlanResult lists execution outcomes in order. If Execute returns a
// non-nil error, Steps holds only completed steps before the failure.
type ActionPlanResult struct {
	Steps []StepResult `json:"steps"`
}

// Execute runs each plan step in order. editParams supplies the file snapshot
// for every edit step (callers should refresh fileText between edits if later
// steps depend on updated buffer content).
func (d *ActionPlanDispatcher) Execute(plan agent.ActionPlan, editParams protocol.EditApplyParams) (ActionPlanResult, error) {
	if err := agent.ValidateActionPlan(plan); err != nil {
		return ActionPlanResult{}, err
	}

	out := ActionPlanResult{Steps: make([]StepResult, 0, len(plan.Steps))}
	for i := range plan.Steps {
		step := plan.Steps[i]
		switch step.Kind {
		case agent.StepKindEdit:
			res, err := d.edits.ApplyWithIntent(editParams, *step.Edit)
			if err != nil {
				return out, fmt.Errorf("action plan: step %d: %w", i, err)
			}
			out.Steps = append(out.Steps, StepResult{EditResult: &res})
			if res.Kind == "failure" {
				return out, nil
			}
		case agent.StepKindRunCommand:
			cmd := d.command.Run(step.RunCommand.CommandParams())
			out.Steps = append(out.Steps, StepResult{CommandResult: &cmd})
			if cmd.Kind == "failure" {
				return out, nil
			}
		default:
			return out, fmt.Errorf("action plan: step %d: unreachable kind %q", i, step.Kind)
		}
	}
	return out, nil
}
