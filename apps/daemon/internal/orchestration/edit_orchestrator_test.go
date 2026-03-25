package orchestration

import (
	"testing"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type plannerStub struct {
	result agent.EditPlanResult
	called bool
}

func (s *plannerStub) PlanEdit(_ protocol.EditApplyParams) agent.EditPlanResult {
	s.called = true
	return s.result
}

type editsStub struct {
	actions []protocol.EditAction
	failure *protocol.EditFailure
	called  bool
}

func (s *editsStub) BuildActions(_ protocol.EditApplyParams, _ agent.EditPlan) ([]protocol.EditAction, *protocol.EditFailure) {
	s.called = true
	return s.actions, s.failure
}

func TestEditOrchestratorApplyCallsPlannerThenBuildsActions(t *testing.T) {
	t.Parallel()

	planner := &plannerStub{
		result: agent.EditPlanResult{
			Plan: &agent.EditPlan{
				Intent: agent.EditIntent{Kind: agent.EditIntentReplaceAnchoredBlock},
			},
		},
	}
	editsService := &editsStub{
		actions: []protocol.EditAction{
			{
				Kind: "replace_between_anchors",
				Path: "/tmp/example.ts",
				Anchor: protocol.Anchor{
					Before: "before",
					After:  "after",
				},
				NewText: "updated",
			},
		},
	}
	orchestrator := &EditOrchestrator{
		agent: planner,
		edits: editsService,
	}

	result, err := orchestrator.Apply(protocol.EditApplyParams{
		Instruction: "replace block after \"before\" before \"after\" with \"updated\"",
		ActiveFile:  "/tmp/example.ts",
		FileText:    "beforeoldafter",
	})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if !planner.called {
		t.Fatal("expected planner to be called")
	}
	if !editsService.called {
		t.Fatal("expected edits service to be called")
	}
	if result.Kind != "success" {
		t.Fatalf("expected success result, got %q", result.Kind)
	}
	if len(result.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(result.Actions))
	}
}

func TestEditOrchestratorApplyReturnsNoopForNoChangeNeeded(t *testing.T) {
	t.Parallel()

	planner := &plannerStub{
		result: agent.EditPlanResult{
			Plan: &agent.EditPlan{
				Intent: agent.EditIntent{Kind: agent.EditIntentAppendImportIfMissing},
			},
		},
	}
	editsService := &editsStub{
		failure: &protocol.EditFailure{
			Code:    "no_change_needed",
			Message: "Import is already present.",
		},
	}
	orchestrator := &EditOrchestrator{
		agent: planner,
		edits: editsService,
	}

	result, err := orchestrator.Apply(protocol.EditApplyParams{})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if result.Kind != "noop" {
		t.Fatalf("expected noop result, got %q", result.Kind)
	}
	if result.Reason == "" {
		t.Fatal("expected noop reason")
	}
}
