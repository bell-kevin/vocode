package agent

import "context"

// ModelInput is everything the model needs to propose an [ActionPlan] for one
// user turn. Fields may grow (active file, selection, workspace roots, etc.).
type ModelInput struct {
	Transcript string
}

// ModelClient calls an LLM or other planner to turn [ModelInput] into a
// validated-shaped [ActionPlan]. Vendor implementations live in subpackages
// (e.g. agent/stub, agent/openai, agent/anthropic); compose with [New] at the app boundary.
type ModelClient interface {
	Plan(ctx context.Context, in ModelInput) (ActionPlan, error)
}
