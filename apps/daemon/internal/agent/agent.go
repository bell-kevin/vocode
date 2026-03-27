package agent

import (
	"context"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/actionplan"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Agent is the daemon-side voice/agent runtime: transcript handling and
// coordination with a [ModelClient] to produce iterative action plans.
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
	Plan *actionplan.ActionPlan
	// Err is set when the model fails or the plan fails [ValidateActionPlan].
	Err error
}

// HandleTranscript runs the model on non-empty transcript text.
func (a *Agent) HandleTranscript(ctx context.Context, params protocol.VoiceTranscriptParams) HandleTranscriptResult {
	text := strings.TrimSpace(params.Text)
	if text == "" {
		return HandleTranscriptResult{Valid: false}
	}

	plan, err := a.planForTranscript(ctx, text)
	if err != nil {
		return HandleTranscriptResult{Valid: true, Plan: nil, Err: err}
	}
	return HandleTranscriptResult{Valid: true, Plan: &plan}
}

func (a *Agent) planForTranscript(ctx context.Context, text string) (actionplan.ActionPlan, error) {
	const maxTurns = 8
	out := actionplan.ActionPlan{Steps: make([]actionplan.Step, 0, maxTurns)}
	for i := 0; i < maxTurns; i++ {
		next, err := a.model.NextAction(ctx, ModelInput{
			Transcript:     text,
			CompletedSteps: append([]actionplan.Step(nil), out.Steps...),
		})
		if err != nil {
			return actionplan.ActionPlan{}, err
		}
		step, done, err := actionplan.NextActionToStep(next)
		if err != nil {
			return actionplan.ActionPlan{}, err
		}
		if done {
			return out, actionplan.ValidateActionPlan(out)
		}
		out.Steps = append(out.Steps, step)
	}
	return actionplan.ActionPlan{}, actionplan.ValidateActionPlan(out)
}
