package agent

import (
	"context"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
)

// Agent is the iterative agent-loop facade.
type Agent struct {
	model ModelClient
}

func New(model ModelClient) *Agent {
	return &Agent{model: model}
}

// NextTurn proxies one model completion for the transcript executor.
func (a *Agent) NextTurn(ctx context.Context, in agentcontext.TurnContext) (TurnResult, error) {
	return a.model.NextTurn(ctx, in)
}
