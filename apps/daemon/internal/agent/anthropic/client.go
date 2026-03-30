// Package anthropic will host an iterative [agent.ModelClient] backed by the
// Anthropic API (e.g. Claude Opus). The skeleton returns [ErrNotImplemented] until wired.
package anthropic

import (
	"context"
	"errors"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
)

// ErrNotImplemented is returned by [Client.NextTurn] until implemented.
var ErrNotImplemented = errors.New("anthropic model client: not implemented")

// Client holds Anthropic-specific configuration (API key, model id, etc.).
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
