package agent

import (
	"testing"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func TestIntentPlannerParsesInsertStatement(t *testing.T) {
	t.Parallel()

	planner := NewIntentPlanner()
	result := planner.Plan(protocol.EditApplyParams{
		Instruction: "insert statement `console.log(\"done\")` inside current function",
	})

	if result.Failure != nil {
		t.Fatalf("expected success plan, got failure: %+v", *result.Failure)
	}
	if result.Plan == nil {
		t.Fatal("expected plan to be present")
	}
	if result.Plan.Intent.Kind != EditIntentInsertStatementInCurrentFunction {
		t.Fatalf("unexpected intent kind: %q", result.Plan.Intent.Kind)
	}
	if result.Plan.Intent.Statement != `console.log("done")` {
		t.Fatalf("unexpected statement: %q", result.Plan.Intent.Statement)
	}
}

func TestIntentPlannerParsesReplaceAnchoredBlock(t *testing.T) {
	t.Parallel()

	planner := NewIntentPlanner()
	result := planner.Plan(protocol.EditApplyParams{
		Instruction: "replace block after \"before\" before \"after\" with `updated`",
	})

	if result.Failure != nil {
		t.Fatalf("expected success plan, got failure: %+v", *result.Failure)
	}
	if result.Plan == nil {
		t.Fatal("expected plan to be present")
	}
	if result.Plan.Intent.Kind != EditIntentReplaceAnchoredBlock {
		t.Fatalf("unexpected intent kind: %q", result.Plan.Intent.Kind)
	}
	if result.Plan.Intent.Before != "before" || result.Plan.Intent.After != "after" {
		t.Fatalf("unexpected anchors: before=%q after=%q", result.Plan.Intent.Before, result.Plan.Intent.After)
	}
	if result.Plan.Intent.NewText != "updated" {
		t.Fatalf("unexpected replacement text: %q", result.Plan.Intent.NewText)
	}
}

func TestIntentPlannerParsesAppendImportIfMissing(t *testing.T) {
	t.Parallel()

	planner := NewIntentPlanner()
	result := planner.Plan(protocol.EditApplyParams{
		Instruction: "append import `import \"fmt\"` if missing",
	})

	if result.Failure != nil {
		t.Fatalf("expected success plan, got failure: %+v", *result.Failure)
	}
	if result.Plan == nil {
		t.Fatal("expected plan to be present")
	}
	if result.Plan.Intent.Kind != EditIntentAppendImportIfMissing {
		t.Fatalf("unexpected intent kind: %q", result.Plan.Intent.Kind)
	}
	if result.Plan.Intent.Import != `import "fmt"` {
		t.Fatalf("unexpected import statement: %q", result.Plan.Intent.Import)
	}
}

func TestIntentPlannerRejectsEmptyInstruction(t *testing.T) {
	t.Parallel()

	planner := NewIntentPlanner()
	result := planner.Plan(protocol.EditApplyParams{
		Instruction: "   ",
	})

	if result.Failure == nil {
		t.Fatal("expected failure for empty instruction")
	}
	if result.Failure.Code != "unsupported_instruction" {
		t.Fatalf("unexpected failure code: %q", result.Failure.Code)
	}
}

func TestIntentPlannerRejectsUnsupportedInstruction(t *testing.T) {
	t.Parallel()

	planner := NewIntentPlanner()
	result := planner.Plan(protocol.EditApplyParams{
		Instruction: "rewrite the whole file",
	})

	if result.Failure == nil {
		t.Fatal("expected failure for unsupported instruction")
	}
	if result.Failure.Code != "unsupported_instruction" {
		t.Fatalf("unexpected failure code: %q", result.Failure.Code)
	}
}

func TestIntentPlannerRejectsAppendImportWithoutImportPrefix(t *testing.T) {
	t.Parallel()

	planner := NewIntentPlanner()
	result := planner.Plan(protocol.EditApplyParams{
		Instruction: "append import `fmt` if missing",
	})

	if result.Failure == nil {
		t.Fatal("expected failure for malformed import instruction")
	}
	if result.Failure.Code != "unsupported_instruction" {
		t.Fatalf("unexpected failure code: %q", result.Failure.Code)
	}
}
