// Package openai will host an iterative [agent.ModelClient] backed by the
// OpenAI API (e.g. GPT-4.1). The skeleton returns [ErrNotImplemented] until wired.
package openai

import (
	"context"
	"errors"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
)

// ErrNotImplemented is returned by [Client.NextTurn] until implemented.
var ErrNotImplemented = errors.New("openai model client: not implemented")

// Client holds OpenAI-specific configuration (API key, model name, base URL, etc.).
type Client struct{}

// New returns a client placeholder. Options will be added when the API is integrated.
func New() *Client {
	return &Client{}
}

// NextTurn implements [agent.ModelClient].
func (*Client) NextTurn(ctx context.Context, in agentcontext.TurnContext) (agent.TurnResult, error) {
	_ = ctx
	_ = in
	return agent.TurnResult{}, ErrNotImplemented
}
