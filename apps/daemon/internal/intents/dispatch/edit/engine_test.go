package edit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	"vocoding.net/vocode/v2/apps/daemon/internal/symbols"
)

func TestApplyInsertStatementInCurrentFunction(t *testing.T) {
	t.Parallel()

	engine := NewEngine()
	fileText := readFixture(t, "single-function.ts")
	params := EditExecutionContext{
		ActiveFile: "/tmp/single-function.ts",
		FileText:   fileText,
	}
	intent := intents.EditIntent{
		Kind: intents.EditIntentKindInsert,
		Insert: &intents.InsertEditIntent{
			Target: intents.EditTarget{
				Kind: intents.EditTargetKindSymbolID,
				SymbolID: &intents.SymbolIDTarget{
					ID: symbols.BuildSymbolID(symbols.SymbolRef{
						Name: "current_function",
						Path: "/tmp/single-function.ts",
						Line: 1,
						Kind: "function",
					}),
				},
			},
			Text: `console.log("done")`,
		},
	}

	result, failure := engine.BuildActions(params, intent)
	if failure != nil {
		t.Fatalf("BuildActions returned failure: %+v", *failure)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 action, got %d", len(result))
	}

	action := result[0]
	if !strings.Contains(action.NewText, `console.log("done");`) {
		t.Fatalf("expected generated statement, got %q", action.NewText)
	}
	if strings.Contains(action.NewText, "hi from vocode") {
		t.Fatalf("legacy demo insertion leaked into action: %q", action.NewText)
	}
}

func TestApplyReplaceCurrentFunctionBody(t *testing.T) {
	t.Parallel()

	engine := NewEngine()
	fileText := readFixture(t, "single-function.ts")
	params := EditExecutionContext{
		ActiveFile: "/tmp/single-function.ts",
		FileText:   fileText,
	}
	intent := intents.EditIntent{
		Kind: intents.EditIntentKindReplace,
		Replace: &intents.ReplaceEditIntent{
			Target: intents.EditTarget{
				Kind: intents.EditTargetKindSymbolID,
				SymbolID: &intents.SymbolIDTarget{
					ID: symbols.BuildSymbolID(symbols.SymbolRef{
						Name: "current_function",
						Path: "/tmp/single-function.ts",
						Line: 1,
						Kind: "function",
					}),
				},
			},
			NewText: `console.log("hello from vocode");`,
		},
	}

	result, failure := engine.BuildActions(params, intent)
	if failure != nil {
		t.Fatalf("BuildActions returned failure: %+v", *failure)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 action, got %d", len(result))
	}
	if !strings.Contains(result[0].NewText, `console.log("hello from vocode");`) {
		t.Fatalf("expected replacement body, got %q", result[0].NewText)
	}
	if strings.Contains(result[0].NewText, "hello ${name}") {
		t.Fatalf("expected old return to be replaced, got %q", result[0].NewText)
	}
}

func TestApplyFailsForAmbiguousCurrentFunction(t *testing.T) {
	t.Parallel()

	engine := NewEngine()
	fileText := readFixture(t, "multi-function.ts")
	params := EditExecutionContext{
		ActiveFile: "/tmp/multi-function.ts",
		FileText:   fileText,
	}
	intent := intents.EditIntent{
		Kind: intents.EditIntentKindInsert,
		Insert: &intents.InsertEditIntent{
			Target: intents.EditTarget{
				Kind: intents.EditTargetKindSymbolID,
				SymbolID: &intents.SymbolIDTarget{
					ID: symbols.BuildSymbolID(symbols.SymbolRef{
						Name: "current_function",
						Path: "/tmp/multi-function.ts",
						Line: 1,
						Kind: "function",
					}),
				},
			},
			Text: `console.log("done")`,
		},
	}

	_, failure := engine.BuildActions(params, intent)
	if failure == nil {
		t.Fatal("expected failure but got success")
	}
	if failure.Code != "ambiguous_target" {
		t.Fatalf("expected ambiguous_target failure, got %+v", *failure)
	}
}

func TestApplyReplaceAnchoredBlock(t *testing.T) {
	t.Parallel()

	engine := NewEngine()
	fileText := readFixture(t, "anchored-block.ts")
	params := EditExecutionContext{
		ActiveFile: "/tmp/anchored-block.ts",
		FileText:   fileText,
	}
	intent := intents.EditIntent{
		Kind: intents.EditIntentKindReplace,
		Replace: &intents.ReplaceEditIntent{
			Target: intents.EditTarget{
				Kind: intents.EditTargetKindAnchor,
				Anchor: &intents.AnchorTarget{
					Before: "export function firstBraceAnchors() {",
					After:  "}",
				},
			},
			NewText: "\n  return \"updated\";\n",
		},
	}

	result, failure := engine.BuildActions(params, intent)
	if failure != nil {
		t.Fatalf("unexpected failure: %+v", *failure)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 action, got %d", len(result))
	}
	if !strings.Contains(result[0].Anchor.Before, "firstBraceAnchors") {
		t.Fatalf("expected anchored action to target firstBraceAnchors, got %+v", result[0].Anchor)
	}
}

func TestApplyReplaceBySymbolIDFunction(t *testing.T) {
	t.Parallel()

	engine := NewEngine()
	fileText := readFixture(t, "multi-function.ts")
	params := EditExecutionContext{
		ActiveFile: "/tmp/multi-function.ts",
		FileText:   fileText,
	}
	intent := intents.EditIntent{
		Kind: intents.EditIntentKindReplace,
		Replace: &intents.ReplaceEditIntent{
			Target: intents.EditTarget{
				Kind: intents.EditTargetKindSymbolID,
				SymbolID: &intents.SymbolIDTarget{
					ID: symbols.BuildSymbolID(symbols.SymbolRef{
						Name: "secondBraceAnchors",
						Path: "/tmp/multi-function.ts",
						Line: 5,
						Kind: "function",
					}),
				},
			},
			NewText: "\n  return 42;\n",
		},
	}

	result, failure := engine.BuildActions(params, intent)
	if failure != nil {
		t.Fatalf("unexpected failure: %+v", *failure)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 action, got %d", len(result))
	}
	if !strings.Contains(result[0].Anchor.Before, "secondBraceAnchors") {
		t.Fatalf("expected secondBraceAnchors target, got %+v", result[0].Anchor)
	}
}

func TestApplyAppendImportIfMissing(t *testing.T) {
	t.Parallel()

	engine := NewEngine()
	fileText := readFixture(t, "imports.go")
	params := EditExecutionContext{
		ActiveFile: "/tmp/imports.go",
		FileText:   fileText,
	}
	intent := intents.EditIntent{
		Kind: intents.EditIntentKindInsertImport,
		InsertImport: &intents.InsertImportEditIntent{
			Import: `import "fmt"`,
		},
	}

	result, failure := engine.BuildActions(params, intent)
	if failure != nil {
		t.Fatalf("unexpected failure: %+v", *failure)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 action, got %d", len(result))
	}
	if !strings.Contains(result[0].NewText, `import "fmt"`) {
		t.Fatalf("expected import insertion, got %q", result[0].NewText)
	}
}

func readFixture(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("fixtures", name)
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}

	return string(bytes)
}
