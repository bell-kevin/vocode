package edit

import (
	"strings"
	"testing"

	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
)

func TestDispatchEditSuccess(t *testing.T) {
	t.Parallel()

	engine := NewEngine()
	fileText := readFixture(t, "anchored-block.ts")
	params := EditExecutionContext{
		Instruction: "replace inner",
		ActiveFile:  "/tmp/anchored-block.ts",
		FileText:    fileText,
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

	result, err := engine.DispatchEdit(params, intent)
	if err != nil {
		t.Fatalf("DispatchEdit returned error: %v", err)
	}
	if result.Kind != "success" {
		t.Fatalf("expected success result, got %q", result.Kind)
	}
	if len(result.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(result.Actions))
	}
	if !strings.Contains(result.Actions[0].Anchor.Before, "firstBraceAnchors") {
		t.Fatalf("expected anchored action, got %+v", result.Actions[0].Anchor)
	}
}

func TestDispatchEditNoopWhenImportAlreadyPresent(t *testing.T) {
	t.Parallel()

	engine := NewEngine()
	params := EditExecutionContext{
		Instruction: "add fmt",
		ActiveFile:  "/tmp/x.go",
		FileText: `package main

import "fmt"

func main() {
	fmt.Println("ok")
}
`,
	}
	intent := intents.EditIntent{
		Kind: intents.EditIntentKindInsertImport,
		InsertImport: &intents.InsertImportEditIntent{
			Import: `import "fmt"`,
		},
	}

	result, err := engine.DispatchEdit(params, intent)
	if err != nil {
		t.Fatalf("DispatchEdit returned error: %v", err)
	}
	if result.Kind != "noop" {
		t.Fatalf("expected noop result, got %q", result.Kind)
	}
	if result.Reason == "" {
		t.Fatal("expected noop reason")
	}
}
