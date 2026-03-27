package agent

import (
	"context"

	"vocoding.net/vocode/v2/apps/daemon/internal/actionplan"
)

// ModelInput is everything the model needs to propose the next action.
// Fields may grow (active file, selection, workspace roots, etc.).
type ModelInput struct {
	Transcript     string
	CompletedSteps []actionplan.Step
}

// ModelClient is the iterative planning contract.
type ModelClient interface {
	NextAction(ctx context.Context, in ModelInput) (actionplan.NextAction, error)
}
