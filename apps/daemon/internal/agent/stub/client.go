// Package stub provides a fixed-response iterative [agent.ModelClient] for tests
// and dev wiring.
package stub

import (
	"context"
	"runtime"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	"vocoding.net/vocode/v2/apps/daemon/internal/symbols"
)

// Client ignores input and returns a fixed iterative action sequence.
type Client struct{}

// New returns a [Client] that satisfies [agent.ModelClient].
func New() *Client {
	return &Client{}
}

// NextIntent emits a deterministic 4-step sequence, then done.
func (*Client) NextIntent(ctx context.Context, in agent.ModelInput) (intents.Intent, error) {
	_ = ctx
	switch len(in.CompletedActions) {
	case 0:
		return intents.FromExecutable(intents.ExecutableIntent{
			Kind: intents.ExecutableIntentKindNavigate,
			Navigate: &intents.NavigationIntent{
				Kind: intents.NavigationIntentKindOpenFile,
				OpenFile: &intents.OpenFileNavigationIntent{
					Path: "test.js",
				},
			},
		}), nil
	case 1:
		return intents.FromExecutable(intents.ExecutableIntent{
			Kind: intents.ExecutableIntentKindNavigate,
			Navigate: &intents.NavigationIntent{
				Kind: intents.NavigationIntentKindRevealSymbol,
				RevealSymbol: &intents.RevealSymbolNavigationIntent{
					Path:       "test.js",
					SymbolName: "test",
					SymbolKind: "function",
				},
			},
		}), nil
	case 2:
		return intents.FromExecutable(intents.ExecutableIntent{
			Kind: intents.ExecutableIntentKindEdit,
			Edit: &intents.EditIntent{
				Kind: intents.EditIntentKindReplace,
				Replace: &intents.ReplaceEditIntent{
					Target: intents.EditTarget{
						Kind: intents.EditTargetKindSymbolID,
						SymbolID: &intents.SymbolIDTarget{
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
		}), nil
	case 3:
		return intents.FromExecutable(intents.ExecutableIntent{
			Kind:    intents.ExecutableIntentKindCommand,
			Command: stubEchoRunCommand(),
		}), nil
	default:
		return intents.ControlDone(), nil
	}
}

// stubEchoRunCommand: on Windows, `echo` is a cmd builtin (no echo.exe on PATH);
// Go's exec needs cmd.exe /c. On Unix, /bin/echo (or PATH) is a real binary.
func stubEchoRunCommand() *intents.CommandIntent {
	if runtime.GOOS == "windows" {
		return &intents.CommandIntent{
			Command: "cmd.exe",
			Args:    []string{"/c", "echo", "stub-model-client"},
		}
	}
	return &intents.CommandIntent{
		Command: "echo",
		Args:    []string{"stub-model-client"},
	}
}
