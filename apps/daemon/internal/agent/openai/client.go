// Package openai will host an iterative [agent.ModelClient] backed by the
// OpenAI API (e.g. GPT-4.1). The skeleton returns [ErrNotImplemented] until wired.
package openai

import (
	"context"
	"errors"

	"vocoding.net/vocode/v2/apps/daemon/internal/actionplan"
	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
)

// ErrNotImplemented is returned by [Client.NextAction] until implemented.
var ErrNotImplemented = errors.New("openai model client: not implemented")

// Client holds OpenAI-specific configuration (API key, model name, base URL, etc.).
type Client struct{}

// New returns a client placeholder. Options will be added when the API is integrated.
func New() *Client {
	return &Client{}
}

// NextAction implements [agent.ModelClient].
func (*Client) NextAction(ctx context.Context, in agent.ModelInput) (actionplan.NextAction, error) {
	_ = ctx
	_ = in
	return actionplan.NextAction{}, ErrNotImplemented
}
