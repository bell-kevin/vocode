package actionplan

import "testing"

func TestValidateNextActionDone(t *testing.T) {
	t.Parallel()
	if err := ValidateNextAction(NextAction{Kind: NextActionKindDone}); err != nil {
		t.Fatalf("expected done action to be valid: %v", err)
	}
}

func TestNextActionToStepEdit(t *testing.T) {
	t.Parallel()
	a := NextAction{
		Kind: NextActionKindEdit,
		Edit: &EditIntent{
			Kind: EditIntentKindReplace,
			Replace: &ReplaceEditIntent{
				Target: EditTarget{
					Kind:     EditTargetKindSymbolID,
					SymbolID: &SymbolIDTarget{ID: "v1|Zm9vLnRz|1|ZnVuY3Rpb24|YmFy"},
				},
				NewText: "x",
			},
		},
	}
	step, done, err := NextActionToStep(a)
	if err != nil {
		t.Fatalf("expected no err: %v", err)
	}
	if done {
		t.Fatal("expected done=false")
	}
	if step.Kind != StepKindEdit || step.Edit == nil {
		t.Fatalf("unexpected step: %+v", step)
	}
}
