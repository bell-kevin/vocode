package edits

import (
	"strings"
	"testing"

	"vocoding.net/vocode/v2/apps/daemon/internal/actionplan"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func TestApplyIntentSuccess(t *testing.T) {
	t.Parallel()

	service := NewService()
	fileText := readFixture(t, "anchored-block.ts")
	params := protocol.EditApplyParams{
		Instruction: "replace inner",
		ActiveFile:  "/tmp/anchored-block.ts",
		FileText:    fileText,
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

	result, err := service.ApplyIntent(params, intent)
	if err != nil {
		t.Fatalf("ApplyIntent returned error: %v", err)
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

func TestApplyIntentNoopWhenImportAlreadyPresent(t *testing.T) {
	t.Parallel()

	service := NewService()
	params := protocol.EditApplyParams{
		Instruction: "add fmt",
		ActiveFile:  "/tmp/x.go",
		FileText: `package main

import "fmt"

func main() {
	fmt.Println("ok")
}
`,
	}
	intent := actionplan.EditIntent{
		Kind: actionplan.EditIntentKindInsertImport,
		InsertImport: &actionplan.InsertImportEditIntent{
			Import: `import "fmt"`,
		},
	}

	result, err := service.ApplyIntent(params, intent)
	if err != nil {
		t.Fatalf("ApplyIntent returned error: %v", err)
	}
	if result.Kind != "noop" {
		t.Fatalf("expected noop result, got %q", result.Kind)
	}
	if result.Reason == "" {
		t.Fatal("expected noop reason")
	}
}
