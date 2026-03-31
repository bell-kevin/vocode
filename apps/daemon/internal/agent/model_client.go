package agent

import (
	"context"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
)

// ModelClient is the operation pipeline contract.
type ModelClient interface {
	ClassifyTranscript(ctx context.Context, in agentcontext.TranscriptClassifierContext) (TranscriptClassifierResult, error)
	ScopeIntent(ctx context.Context, in agentcontext.ScopeIntentContext) (ScopeIntentResult, error)
	ScopedEdit(ctx context.Context, in agentcontext.ScopedEditContext) (ScopedEditResult, error)
}
