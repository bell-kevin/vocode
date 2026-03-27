// Package stub provides a fixed-response iterative [agent.ModelClient] for tests
// and dev wiring.
package stub

import (
	"context"
	"runtime"

	"vocoding.net/vocode/v2/apps/daemon/internal/actionplan"
	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/symbols"
)

// Client ignores input and always returns the same hardcoded [actionplan.ActionPlan].
type Client struct{}

// New returns a [Client] that satisfies [agent.ModelClient].
func New() *Client {
	return &Client{}
}

// NextAction emits a deterministic 4-step sequence, then done.
func (*Client) NextAction(ctx context.Context, in agent.ModelInput) (actionplan.NextAction, error) {
	_ = ctx
	switch len(in.CompletedSteps) {
	case 0:
		return actionplan.NextAction{
			Kind: actionplan.NextActionKindNavigate,
			Navigate: &actionplan.NavigationIntent{
				Kind: actionplan.NavigationIntentKindOpenFile,
				OpenFile: &actionplan.OpenFileNavigationIntent{
					Path: "test.js",
				},
			},
		}, nil
	case 1:
		return actionplan.NextAction{
			Kind: actionplan.NextActionKindNavigate,
			Navigate: &actionplan.NavigationIntent{
				Kind: actionplan.NavigationIntentKindRevealSymbol,
				RevealSymbol: &actionplan.RevealSymbolNavigationIntent{
					Path:       "test.js",
					SymbolName: "test",
					SymbolKind: "function",
				},
			},
		}, nil
	case 2:
		return actionplan.NextAction{
			Kind: actionplan.NextActionKindEdit,
			Edit: &actionplan.EditIntent{
				Kind: actionplan.EditIntentKindReplace,
				Replace: &actionplan.ReplaceEditIntent{
					Target: actionplan.EditTarget{
						Kind: actionplan.EditTargetKindSymbolID,
						SymbolID: &actionplan.SymbolIDTarget{
							ID: symbols.BuildSymbolID(symbols.SymbolRef{
								Name: "test",
								Path: "test.js",
								Line: 1,
								Kind: "function",
							}),
						},
					},
					NewText: "\n  console.log(\"updated from stub\");\n",
				},
			},
		}, nil
	case 3:
		return actionplan.NextAction{
			Kind:       actionplan.NextActionKindRunCommand,
			RunCommand: stubEchoRunCommand(),
		}, nil
	default:
		return actionplan.NextAction{Kind: actionplan.NextActionKindDone}, nil
	}
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
