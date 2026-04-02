package router

import "vocoding.net/vocode/v2/apps/core/internal/flows"

// Context is the minimal input for route classification: which flow we are in and what the user said.
// Per-route handlers build their own prompts and context when they call a model.
type Context struct {
	Flow        flows.ID
	Instruction string
}
