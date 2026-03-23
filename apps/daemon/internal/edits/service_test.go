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

	agentService := agent.NewService()
	service := NewService()
	fileText := readFixture(t, "single-function.ts")
	params := protocol.EditApplyParams{
		Instruction: "insert statement `console.log(\"done\")` inside current function",
		ActiveFile:  "/tmp/single-function.ts",
		FileText:    fileText,
	}

	planResult := agentService.PlanEdit(params)
	if planResult.Failure != nil {
		t.Fatalf("unexpected planning failure: %+v", *planResult.Failure)
	}
	result, failure := service.BuildActions(params, *planResult.Plan)
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

func TestApplyFailsForAmbiguousCurrentFunction(t *testing.T) {
	t.Parallel()

	agentService := agent.NewService()
	service := NewService()
	fileText := readFixture(t, "multi-function.ts")
	params := protocol.EditApplyParams{
		Instruction: "insert statement `console.log(\"done\")` inside current function",
		ActiveFile:  "/tmp/multi-function.ts",
		FileText:    fileText,
	}

	planResult := agentService.PlanEdit(params)
	if planResult.Failure != nil {
		t.Fatalf("unexpected planning failure: %+v", *planResult.Failure)
	}
	_, failure := service.BuildActions(params, *planResult.Plan)
	if failure == nil {
		t.Fatal("expected failure but got success")
	}
	if failure.Code != "ambiguous_target" {
		t.Fatalf("expected ambiguous_target failure, got %+v", *failure)
	}
}

func TestApplyReplaceAnchoredBlock(t *testing.T) {
	t.Parallel()

	agentService := agent.NewService()
	service := NewService()
	fileText := readFixture(t, "anchored-block.ts")
	params := protocol.EditApplyParams{
		Instruction: "replace block after \"export function firstBraceAnchors() {\" before \"}\" with `\\n  return \\\"updated\\\";\\n`",
		ActiveFile:  "/tmp/anchored-block.ts",
		FileText:    fileText,
	}

	planResult := agentService.PlanEdit(params)
	if planResult.Failure != nil {
		t.Fatalf("unexpected planning failure: %+v", *planResult.Failure)
	}
	result, failure := service.BuildActions(params, *planResult.Plan)
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

	agentService := agent.NewService()
	service := NewService()
	fileText := readFixture(t, "imports.go")
	params := protocol.EditApplyParams{
		Instruction: "append import `import \"fmt\"` if missing",
		ActiveFile:  "/tmp/imports.go",
		FileText:    fileText,
	}

	planResult := agentService.PlanEdit(params)
	if planResult.Failure != nil {
		t.Fatalf("unexpected planning failure: %+v", *planResult.Failure)
	}
	result, failure := service.BuildActions(params, *planResult.Plan)
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

	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(data)
}
