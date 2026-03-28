package agent

import (
	"context"

	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
)

// Agent is the iterative planner facade.
type Agent struct {
	model ModelClient
}

func New(model ModelClient) *Agent {
	return &Agent{model: model}
}

// NextIntent proxies one iterative planner turn.
func (a *Agent) NextIntent(ctx context.Context, in ModelInput) (intents.Intent, error) {
	return a.model.NextIntent(ctx, in)
}
