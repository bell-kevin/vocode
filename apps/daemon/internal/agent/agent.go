package agent

import (
	"context"

	"vocoding.net/vocode/v2/apps/daemon/internal/intent"
)

// Agent is the daemon-side runtime facade around the planner [ModelClient].
type Agent struct {
	model ModelClient
}

// New builds an agent with the given [ModelClient] (stub, OpenAI, Anthropic, tests, …).
func New(model ModelClient) *Agent {
	return &Agent{model: model}
}

// NextIntent proxies one iterative planner turn.
func (a *Agent) NextIntent(ctx context.Context, in ModelInput) (intent.NextIntent, error) {
	return a.model.NextIntent(ctx, in)
}
