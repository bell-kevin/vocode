package agent

import (
	"context"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
)

// Agent is the daemon-owned operation pipeline facade.
type Agent struct {
	model ModelClient
}

func New(model ModelClient) *Agent {
	return &Agent{model: model}
}

func (a *Agent) ClassifyTranscript(ctx context.Context, in agentcontext.TranscriptClassifierContext) (TranscriptClassifierResult, error) {
	return a.model.ClassifyTranscript(ctx, in)
}

func (a *Agent) ScopeIntent(ctx context.Context, in agentcontext.ScopeIntentContext) (ScopeIntentResult, error) {
	return a.model.ScopeIntent(ctx, in)
}

// ScopedEdit asks the model for replacement text for a daemon-resolved target.
func (a *Agent) ScopedEdit(ctx context.Context, in agentcontext.ScopedEditContext) (ScopedEditResult, error) {
	return a.model.ScopedEdit(ctx, in)
}
