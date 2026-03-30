// Package stub provides a fixed-response [agent.ModelClient] for tests and dev wiring.
package stub

import (
	"context"
	"runtime"
	"strings"

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
		return agent.TurnResult{Kind: agent.TurnFinish, FinishSummary: "stub model acknowledged prior host apply"}, nil
	}

	active := strings.TrimSpace(in.Editor.ActiveFilePath)
	if active == "" {
		active = "test.js"
	}
	return agent.TurnResult{
		Kind: agent.TurnIntents,
		Intents: []intents.Intent{
			{
				Kind: intents.IntentKindNavigate,
				Navigate: &intents.NavigationIntent{
					Kind: intents.NavigationIntentKindOpenFile,
					OpenFile: &intents.OpenFileNavigationIntent{
						Path: active,
					},
				},
			},
			{
				Kind: intents.IntentKindNavigate,
				Navigate: &intents.NavigationIntent{
					Kind: intents.NavigationIntentKindRevealSymbol,
					RevealSymbol: &intents.RevealSymbolNavigationIntent{
						Path:       active,
						SymbolName: "test",
						SymbolKind: "function",
					},
				},
			},
			{
				Kind: intents.IntentKindEdit,
				Edit: &intents.EditIntent{
					Kind: intents.EditIntentKindReplace,
					Replace: &intents.ReplaceEditIntent{
						Target: intents.EditTarget{
							Kind: intents.EditTargetKindSymbolID,
							SymbolID: &intents.SymbolIDTarget{
								ID: symbols.BuildSymbolID(symbols.SymbolRef{
									Path: "",
									Line: 0,
									Kind: "function",
									Name: "test",
								}),
							},
						},
						NewText: "\n  console.log(\"updated from stub\");\n",
					},
				},
			},
			{
				Kind:    intents.IntentKindCommand,
				Command: stubEchoRunCommand(),
			},
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
