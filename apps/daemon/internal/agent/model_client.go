package agent

import (
	"context"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
)

// ModelClient is the transcript agent contract: each call receives [agentcontext.TurnContext] and returns
// one [TurnResult] (irrelevant, done, request_context, or a batch of intents).
type ModelClient interface {
	NextTurn(ctx context.Context, in agentcontext.TurnContext) (TurnResult, error)
}
