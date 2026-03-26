package agent

import (
	"context"
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Agent is the daemon-side voice/agent runtime: transcript handling and
// coordination with a [ModelClient] to produce [ActionPlan] values.
type Agent struct {
	model ModelClient
}

// New builds an agent with the given [ModelClient] (stub, OpenAI, Anthropic, tests, …).
func New(model ModelClient) *Agent {
	return &Agent{model: model}
}

// HandleTranscriptResult is the outcome of [Agent.HandleTranscript].
type HandleTranscriptResult struct {
	// Valid is false when the transcript is empty (rejected before the model).
	Valid bool
	// Plan is set when the model returned a plan and validation succeeded.
	Plan *ActionPlan
	// Err is set when the model fails or the plan fails [ValidateActionPlan].
	Err error
}

// HandleTranscript runs the model on non-empty transcript text.
func (a *Agent) HandleTranscript(ctx context.Context, params protocol.VoiceTranscriptParams) HandleTranscriptResult {
	text := strings.TrimSpace(params.Text)
	if text == "" {
		return HandleTranscriptResult{Valid: false}
	}

	plan, err := a.model.Plan(ctx, ModelInput{Transcript: text})
	if err != nil {
		return HandleTranscriptResult{Valid: true, Err: err}
	}
	if err := ValidateActionPlan(plan); err != nil {
		return HandleTranscriptResult{Valid: true, Plan: nil, Err: err}
	}
	return HandleTranscriptResult{Valid: true, Plan: &plan}
}
