package edits

import (
	"strings"
	"testing"

	intentpkg "vocoding.net/vocode/v2/apps/daemon/internal/intent"
)

func TestDispatchIntentSuccess(t *testing.T) {
	t.Parallel()

	service := NewService()
	fileText := readFixture(t, "anchored-block.ts")
	params := EditExecutionContext{
		Instruction: "replace inner",
		ActiveFile:  "/tmp/anchored-block.ts",
		FileText:    fileText,
	}
	intent := intentpkg.EditIntent{
		Kind: intentpkg.EditIntentKindReplace,
		Replace: &intentpkg.ReplaceEditIntent{
			Target: intentpkg.EditTarget{
				Kind: intentpkg.EditTargetKindAnchor,
				Anchor: &intentpkg.AnchorTarget{
					Before: "export function firstBraceAnchors() {",
					After:  "}",
				},
			},
			NewText: "\n  return \"updated\";\n",
		},
	}

	result, err := service.DispatchIntent(params, intent)
	if err != nil {
		t.Fatalf("DispatchIntent returned error: %v", err)
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

func TestDispatchIntentNoopWhenImportAlreadyPresent(t *testing.T) {
	t.Parallel()

	service := NewService()
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
	intent := intentpkg.EditIntent{
		Kind: intentpkg.EditIntentKindInsertImport,
		InsertImport: &intentpkg.InsertImportEditIntent{
			Import: `import "fmt"`,
		},
	}

	result, err := service.DispatchIntent(params, intent)
	if err != nil {
		t.Fatalf("DispatchIntent returned error: %v", err)
	}
	if result.Kind != "noop" {
		t.Fatalf("expected noop result, got %q", result.Kind)
	}
	if result.Reason == "" {
		t.Fatal("expected noop reason")
	}
}
