package agent

import (
	"context"

	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
)

// ModelClient is the iterative agent-loop contract. Each turn receives an [agentcontext.TurnContext].
type ModelClient interface {
	NextIntent(ctx context.Context, in agentcontext.TurnContext) (intents.Intent, error)
}
