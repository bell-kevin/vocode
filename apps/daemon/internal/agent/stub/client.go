// Package stub provides a fixed-response [agent.ModelClient] for tests and dev wiring.
package stub

import (
	"context"
	"runtime"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
)

// Client ignores input and always returns the same hardcoded [agent.ActionPlan].
type Client struct{}

// New returns a [Client] that satisfies [agent.ModelClient].
func New() *Client {
	return &Client{}
}

// Plan implements [agent.ModelClient].
func (*Client) Plan(ctx context.Context, in agent.ModelInput) (agent.ActionPlan, error) {
	_ = ctx
	_ = in
	return agent.ActionPlan{
		Steps: []agent.Step{
			{
				Kind: agent.StepKindEdit,
				Edit: &agent.EditIntent{
					Kind:    agent.EditIntentReplaceCurrentFunctionBody,
					NewText: `console.log("hello from vocode");`,
				},
			},
			{
				Kind:       agent.StepKindRunCommand,
				RunCommand: stubEchoRunCommand(),
			},
		},
	}, nil
}

// stubEchoRunCommand: on Windows, `echo` is a cmd builtin (no echo.exe on PATH);
// Go's exec needs cmd.exe /c. On Unix, /bin/echo (or PATH) is a real binary.
func stubEchoRunCommand() *agent.RunCommandIntent {
	if runtime.GOOS == "windows" {
		return &agent.RunCommandIntent{
			Command: "cmd.exe",
			Args:    []string{"/c", "echo", "stub-model-client"},
		}
	}
	return &agent.RunCommandIntent{
		Command: "echo",
		Args:    []string{"stub-model-client"},
	}
}
