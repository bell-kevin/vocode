package agent

import "context"

// CompletionRequest is a single model turn: system + user text, plus an optional JSON Schema
// for structured output. Providers interpret the schema as they support (e.g. OpenAI
// response_format json_schema; Anthropic may fold it into the system prompt).
type CompletionRequest struct {
	System string
	User   string
	// JSONSchema, when non-empty, describes the desired assistant message shape (draft-07 style).
	JSONSchema map[string]any
}

// ModelClient is a thin provider facade: one completion call per request, no domain logic.
type ModelClient interface {
	Call(ctx context.Context, req CompletionRequest) (assistantText string, err error)
}
