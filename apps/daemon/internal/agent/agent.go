package agent

import (
	"context"

	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
)

// Agent is the iterative agent-loop facade.
type Agent struct {
	model ModelClient
}

func New(model ModelClient) *Agent {
	return &Agent{model: model}
}

// NextIntent proxies one iterative agent-loop turn.
func (a *Agent) NextIntent(ctx context.Context, in agentcontext.TurnContext) (intents.Intent, error) {
	return a.model.NextIntent(ctx, in)
}
