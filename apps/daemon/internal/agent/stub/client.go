// Package stub provides a fixed-response [agent.ModelClient] for tests and dev wiring.
package stub

import (
	"context"
	"runtime"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	"vocoding.net/vocode/v2/apps/daemon/internal/symbols"
)

// Client ignores most input and returns a deterministic turn sequence.
type Client struct{}

// New returns a [Client] that satisfies [agent.ModelClient].
func New() *Client {
	return &Client{}
}

// NextTurn implements [agent.ModelClient].
func (*Client) NextTurn(ctx context.Context, in agentcontext.TurnContext) (agent.TurnResult, error) {
	_ = ctx
	if len(in.IntentApplyHistory) > 0 {
		return agent.TurnResult{Kind: agent.TurnDone, DoneSummary: "stub model acknowledged prior host apply"}, nil
	}
	return agent.TurnResult{
		Kind: agent.TurnIntents,
		Intents: []intents.Intent{
			intents.FromExecutable(intents.ExecutableIntent{
				Kind: intents.ExecutableIntentKindNavigate,
				Navigate: &intents.NavigationIntent{
					Kind: intents.NavigationIntentKindOpenFile,
					OpenFile: &intents.OpenFileNavigationIntent{
						Path: "test.js",
					},
				},
			}),
			intents.FromExecutable(intents.ExecutableIntent{
				Kind: intents.ExecutableIntentKindNavigate,
				Navigate: &intents.NavigationIntent{
					Kind: intents.NavigationIntentKindRevealSymbol,
					RevealSymbol: &intents.RevealSymbolNavigationIntent{
						Path:       "test.js",
						SymbolName: "test",
						SymbolKind: "function",
					},
				},
			}),
			intents.FromExecutable(intents.ExecutableIntent{
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
			}),
			intents.FromExecutable(intents.ExecutableIntent{
				Kind:    intents.ExecutableIntentKindCommand,
				Command: stubEchoRunCommand(),
			}),
		},
	}, nil
}

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
