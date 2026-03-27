// Package stub provides a fixed-response iterative [agent.ModelClient] for tests
// and dev wiring.
package stub

import (
	"context"
	"runtime"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/intent"
	"vocoding.net/vocode/v2/apps/daemon/internal/symbols"
)

// Client ignores input and returns a fixed iterative action sequence.
type Client struct{}

// New returns a [Client] that satisfies [agent.ModelClient].
func New() *Client {
	return &Client{}
}

// NextIntent emits a deterministic 4-step sequence, then done.
func (*Client) NextIntent(ctx context.Context, in agent.ModelInput) (intent.NextIntent, error) {
	_ = ctx
	switch len(in.CompletedActions) {
	case 0:
		return intent.NextIntent{
			Kind: intent.NextIntentKindNavigate,
			Navigate: &intent.NavigationIntent{
				Kind: intent.NavigationIntentKindOpenFile,
				OpenFile: &intent.OpenFileNavigationIntent{
					Path: "test.js",
				},
			},
		}, nil
	case 1:
		return intent.NextIntent{
			Kind: intent.NextIntentKindNavigate,
			Navigate: &intent.NavigationIntent{
				Kind: intent.NavigationIntentKindRevealSymbol,
				RevealSymbol: &intent.RevealSymbolNavigationIntent{
					Path:       "test.js",
					SymbolName: "test",
					SymbolKind: "function",
				},
			},
		}, nil
	case 2:
		return intent.NextIntent{
			Kind: intent.NextIntentKindEdit,
			Edit: &intent.EditIntent{
				Kind: intent.EditIntentKindReplace,
				Replace: &intent.ReplaceEditIntent{
					Target: intent.EditTarget{
						Kind: intent.EditTargetKindSymbolID,
						SymbolID: &intent.SymbolIDTarget{
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
		return intent.NextIntent{
			Kind:       intent.NextIntentKindRunCommand,
			RunCommand: stubEchoRunCommand(),
		}, nil
	default:
		return intent.NextIntent{Kind: intent.NextIntentKindDone}, nil
	}
}

// stubEchoRunCommand: on Windows, `echo` is a cmd builtin (no echo.exe on PATH);
// Go's exec needs cmd.exe /c. On Unix, /bin/echo (or PATH) is a real binary.
func stubEchoRunCommand() *intent.CommandIntent {
	if runtime.GOOS == "windows" {
		return &intent.CommandIntent{
			Command: "cmd.exe",
			Args:    []string{"/c", "echo", "stub-model-client"},
		}
	}
	return &intent.CommandIntent{
		Command: "echo",
		Args:    []string{"stub-model-client"},
	}
}
