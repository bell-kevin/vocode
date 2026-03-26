package orchestration

import (
	"testing"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type editsStub struct {
	actions []protocol.EditAction
	failure *protocol.EditFailure
	called  bool
}

func (s *editsStub) BuildActions(_ protocol.EditApplyParams, _ agent.EditIntent) ([]protocol.EditAction, *protocol.EditFailure) {
	s.called = true
	return s.actions, s.failure
}

func TestEditApplyPipelineApplyWithIntentBuildsActions(t *testing.T) {
	t.Parallel()

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
	pipeline := &EditApplyPipeline{edits: editsService}

	params := protocol.EditApplyParams{
		Instruction: "replace block after \"before\" before \"after\" with \"updated\"",
		ActiveFile:  "/tmp/example.ts",
		FileText:    "beforeoldafter",
	}
	intent := agent.EditIntent{
		Kind:    agent.EditIntentReplaceAnchoredBlock,
		Before:  "before",
		After:   "after",
		NewText: "updated",
	}

	result, err := pipeline.ApplyWithIntent(params, intent)
	if err != nil {
		t.Fatalf("ApplyWithIntent returned error: %v", err)
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

func TestEditApplyPipelineApplyWithIntentReturnsNoopForNoChangeNeeded(t *testing.T) {
	t.Parallel()

	editsService := &editsStub{
		failure: &protocol.EditFailure{
			Code:    "no_change_needed",
			Message: "Import is already present.",
		},
	}
	pipeline := &EditApplyPipeline{edits: editsService}

	result, err := pipeline.ApplyWithIntent(
		protocol.EditApplyParams{
			Instruction: "add import",
			ActiveFile:    "/tmp/x.go",
			FileText:      "package p\n",
		},
		agent.EditIntent{
			Kind:   agent.EditIntentAppendImportIfMissing,
			Import: `import "fmt"`,
		},
	)
	if err != nil {
		t.Fatalf("ApplyWithIntent returned error: %v", err)
	}
	if result.Kind != "noop" {
		t.Fatalf("expected noop result, got %q", result.Kind)
	}
	if result.Reason == "" {
		t.Fatal("expected noop reason")
	}
}
