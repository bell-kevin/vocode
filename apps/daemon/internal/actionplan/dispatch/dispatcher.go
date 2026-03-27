package dispatch

import (
	"fmt"

	"vocoding.net/vocode/v2/apps/daemon/internal/actionplan"
	"vocoding.net/vocode/v2/apps/daemon/internal/commandexec"
	"vocoding.net/vocode/v2/apps/daemon/internal/edits"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Dispatcher runs a validated [actionplan.ActionPlan] step-by-step.
type Dispatcher struct {
	edits    *edits.Service
	commands *commandexec.Service
}

func NewDispatcher(editsService *edits.Service, commandService *commandexec.Service) *Dispatcher {
	return &Dispatcher{edits: editsService, commands: commandService}
}

// StepResult is the outcome of executing one plan step (exactly one pointer is non-nil).
type StepResult struct {
	EditResult    *protocol.EditApplyResult
	CommandParams *protocol.CommandRunParams
	Navigation    *actionplan.NavigationIntent
}

// ExecuteResult lists execution outcomes in order. If Execute returns a non-nil
// error, Steps holds only completed steps before the failure.
type ExecuteResult struct {
	Steps []StepResult `json:"steps"`
}

// Execute runs each plan step in order. editParams supplies the file snapshot
// for every edit step (callers should refresh fileText between edits if later
// steps depend on updated buffer content).
func (d *Dispatcher) Execute(plan actionplan.ActionPlan, editCtx edits.EditExecutionContext) (ExecuteResult, error) {
	if err := actionplan.ValidateActionPlan(plan); err != nil {
		return ExecuteResult{}, err
	}

	out := ExecuteResult{Steps: make([]StepResult, 0, len(plan.Steps))}
	editCounter := 0
	for i := range plan.Steps {
		step := plan.Steps[i]
		switch step.Kind {
		case actionplan.StepKindEdit:
			res, err := d.edits.ApplyIntent(editCtx, *step.Edit)
			if err != nil {
				return out, fmt.Errorf("action plan: step %d: %w", i, err)
			}
			if res.Kind == "success" {
				for j := range res.Actions {
					if res.Actions[j].EditId == "" {
						res.Actions[j].EditId = fmt.Sprintf("edit-%d", editCounter)
						editCounter++
					}
				}
			}
			out.Steps = append(out.Steps, StepResult{EditResult: &res})
			if res.Kind == "failure" {
				return out, nil
			}
		case actionplan.StepKindRunCommand:
			params := step.RunCommand.CommandParams()
			if d.commands != nil {
				if err := d.commands.Validate(params); err != nil {
					return out, fmt.Errorf("action plan: step %d: %w", i, err)
				}
			}
			out.Steps = append(out.Steps, StepResult{CommandParams: &params})
		case actionplan.StepKindNavigate:
			out.Steps = append(out.Steps, StepResult{Navigation: step.Navigate})
		default:
			return out, fmt.Errorf("action plan: step %d: unreachable kind %q", i, step.Kind)
		}
	}
	return out, nil
}
