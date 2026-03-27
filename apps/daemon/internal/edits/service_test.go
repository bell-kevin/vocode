package edits

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vocoding.net/vocode/v2/apps/daemon/internal/actionplan"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func TestApplyInsertStatementInCurrentFunction(t *testing.T) {
	t.Parallel()

	service := NewService()
	fileText := readFixture(t, "single-function.ts")
	params := protocol.EditApplyParams{
		ActiveFile: "/tmp/single-function.ts",
		FileText:   fileText,
	}
	intent := actionplan.EditIntent{
		Kind: actionplan.EditIntentKindInsert,
		Insert: &actionplan.InsertEditIntent{
			Target: actionplan.EditTarget{
				Kind: actionplan.EditTargetKindSymbol,
				Symbol: &actionplan.SymbolTarget{
					SymbolName: "current_function",
					SymbolKind: "function",
				},
			},
			Text: `console.log("done")`,
		},
	}

	result, failure := service.BuildActions(params, intent)
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

	service := NewService()
	fileText := readFixture(t, "single-function.ts")
	params := protocol.EditApplyParams{
		ActiveFile: "/tmp/single-function.ts",
		FileText:   fileText,
	}
	intent := actionplan.EditIntent{
		Kind: actionplan.EditIntentKindReplace,
		Replace: &actionplan.ReplaceEditIntent{
			Target: actionplan.EditTarget{
				Kind: actionplan.EditTargetKindSymbol,
				Symbol: &actionplan.SymbolTarget{
					SymbolName: "current_function",
					SymbolKind: "function",
				},
			},
			NewText: `console.log("hello from vocode");`,
		},
	}

	result, failure := service.BuildActions(params, intent)
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

	service := NewService()
	fileText := readFixture(t, "multi-function.ts")
	params := protocol.EditApplyParams{
		ActiveFile: "/tmp/multi-function.ts",
		FileText:   fileText,
	}
	intent := actionplan.EditIntent{
		Kind: actionplan.EditIntentKindInsert,
		Insert: &actionplan.InsertEditIntent{
			Target: actionplan.EditTarget{
				Kind: actionplan.EditTargetKindSymbol,
				Symbol: &actionplan.SymbolTarget{
					SymbolName: "current_function",
					SymbolKind: "function",
				},
			},
			Text: `console.log("done")`,
		},
	}

	_, failure := service.BuildActions(params, intent)
	if failure == nil {
		t.Fatal("expected failure but got success")
	}
	if failure.Code != "ambiguous_target" {
		t.Fatalf("expected ambiguous_target failure, got %+v", *failure)
	}
}

func TestApplyReplaceAnchoredBlock(t *testing.T) {
	t.Parallel()

	service := NewService()
	fileText := readFixture(t, "anchored-block.ts")
	params := protocol.EditApplyParams{
		ActiveFile: "/tmp/anchored-block.ts",
		FileText:   fileText,
	}
	intent := actionplan.EditIntent{
		Kind: actionplan.EditIntentKindReplace,
		Replace: &actionplan.ReplaceEditIntent{
			Target: actionplan.EditTarget{
				Kind: actionplan.EditTargetKindAnchor,
				Anchor: &actionplan.AnchorTarget{
					Before: "export function firstBraceAnchors() {",
					After:  "}",
				},
			},
			NewText: "\n  return \"updated\";\n",
		},
	}

	result, failure := service.BuildActions(params, intent)
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

func TestApplyAppendImportIfMissing(t *testing.T) {
	t.Parallel()

	service := NewService()
	fileText := readFixture(t, "imports.go")
	params := protocol.EditApplyParams{
		ActiveFile: "/tmp/imports.go",
		FileText:   fileText,
	}
	intent := actionplan.EditIntent{
		Kind: actionplan.EditIntentKindInsertImport,
		InsertImport: &actionplan.InsertImportEditIntent{
			Import: `import "fmt"`,
		},
	}

	result, failure := service.BuildActions(params, intent)
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
