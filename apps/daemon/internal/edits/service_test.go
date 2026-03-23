package edits

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func TestApplyInsertStatementInCurrentFunction(t *testing.T) {
	t.Parallel()

	service := NewService(agent.NewService())
	fileText := readFixture(t, "single-function.ts")

	result, err := service.Apply(protocol.EditApplyParams{
		Instruction: "insert statement `console.log(\"done\")` inside current function",
		ActiveFile:  "/tmp/single-function.ts",
		FileText:    fileText,
	})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if result.Failure != nil {
		t.Fatalf("Apply returned failure: %+v", *result.Failure)
	}
	if len(result.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(result.Actions))
	}

	action := result.Actions[0]
	if !strings.Contains(action.NewText, `console.log("done");`) {
		t.Fatalf("expected generated statement, got %q", action.NewText)
	}
	if strings.Contains(action.NewText, "hi from vocode") {
		t.Fatalf("legacy demo insertion leaked into action: %q", action.NewText)
	}
}

func TestApplyFailsForAmbiguousCurrentFunction(t *testing.T) {
	t.Parallel()

	service := NewService(agent.NewService())
	fileText := readFixture(t, "multi-function.ts")

	result, err := service.Apply(protocol.EditApplyParams{
		Instruction: "insert statement `console.log(\"done\")` inside current function",
		ActiveFile:  "/tmp/multi-function.ts",
		FileText:    fileText,
	})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if result.Failure == nil {
		t.Fatal("expected failure but got success")
	}
	if result.Failure.Code != "ambiguous_target" {
		t.Fatalf("expected ambiguous_target failure, got %+v", *result.Failure)
	}
}

func TestApplyReplaceAnchoredBlock(t *testing.T) {
	t.Parallel()

	service := NewService(agent.NewService())
	fileText := readFixture(t, "anchored-block.ts")

	result, err := service.Apply(protocol.EditApplyParams{
		Instruction: "replace block after \"export function firstBraceAnchors() {\" before \"}\" with `\\n  return \\\"updated\\\";\\n`",
		ActiveFile:  "/tmp/anchored-block.ts",
		FileText:    fileText,
	})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if result.Failure != nil {
		t.Fatalf("unexpected failure: %+v", *result.Failure)
	}
	if len(result.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(result.Actions))
	}
	if !strings.Contains(result.Actions[0].Anchor.Before, "firstBraceAnchors") {
		t.Fatalf("expected anchored action to target firstBraceAnchors, got %+v", result.Actions[0].Anchor)
	}
}

func TestApplyAppendImportIfMissing(t *testing.T) {
	t.Parallel()

	service := NewService(agent.NewService())
	fileText := readFixture(t, "imports.go")

	result, err := service.Apply(protocol.EditApplyParams{
		Instruction: "append import `import \"fmt\"` if missing",
		ActiveFile:  "/tmp/imports.go",
		FileText:    fileText,
	})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if result.Failure != nil {
		t.Fatalf("unexpected failure: %+v", *result.Failure)
	}
	if len(result.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(result.Actions))
	}
	if !strings.Contains(result.Actions[0].NewText, `import "fmt"`) {
		t.Fatalf("expected import insertion, got %q", result.Actions[0].NewText)
	}
}

func readFixture(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(data)
}
