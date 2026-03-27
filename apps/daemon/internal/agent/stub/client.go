// Package stub provides a fixed-response [agent.ModelClient] for tests and dev wiring.
package stub

import (
	"context"
	"runtime"

	"vocoding.net/vocode/v2/apps/daemon/internal/actionplan"
	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
)

// Client ignores input and always returns the same hardcoded [actionplan.ActionPlan].
type Client struct{}

// New returns a [Client] that satisfies [agent.ModelClient].
func New() *Client {
	return &Client{}
}

// Plan implements [agent.ModelClient].
func (*Client) Plan(ctx context.Context, in agent.ModelInput) (actionplan.ActionPlan, error) {
	_ = ctx
	_ = in
	return actionplan.ActionPlan{
		Steps: []actionplan.Step{
			{
				Kind: actionplan.StepKindNavigate,
				Navigate: &actionplan.NavigationIntent{
					Kind: actionplan.NavigationIntentKindOpenFile,
					OpenFile: &actionplan.OpenFileNavigationIntent{
						Path: "test.js",
					},
				},
			},
			{
				Kind: actionplan.StepKindNavigate,
				Navigate: &actionplan.NavigationIntent{
					Kind: actionplan.NavigationIntentKindRevealSymbol,
					RevealSymbol: &actionplan.RevealSymbolNavigationIntent{
						Path:       "test.js",
						SymbolName: "test",
						SymbolKind: "function",
					},
				},
			},
			{
				Kind: actionplan.StepKindEdit,
				Edit: &actionplan.EditIntent{
					Kind: actionplan.EditIntentKindReplace,
					Replace: &actionplan.ReplaceEditIntent{
						Target: actionplan.EditTarget{
							Kind: actionplan.EditTargetKindAnchor,
							Anchor: &actionplan.AnchorTarget{
								Path:   "test.js",
								Before: "function test() {",
								After:  "}",
							},
						},
						NewText: "\n  console.log(\"updated from stub\");\n",
					},
				},
			},
			{
				Kind:       actionplan.StepKindRunCommand,
				RunCommand: stubEchoRunCommand(),
			},
		},
	}, nil
}

// stubEchoRunCommand: on Windows, `echo` is a cmd builtin (no echo.exe on PATH);
// Go's exec needs cmd.exe /c. On Unix, /bin/echo (or PATH) is a real binary.
func stubEchoRunCommand() *actionplan.CommandIntent {
	if runtime.GOOS == "windows" {
		return &actionplan.CommandIntent{
			Command: "cmd.exe",
			Args:    []string{"/c", "echo", "stub-model-client"},
		}
	}
	return &actionplan.CommandIntent{
		Command: "echo",
		Args:    []string{"stub-model-client"},
	}
}
