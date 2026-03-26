package actionplan

import "testing"

func TestValidateActionPlanSingleEdit(t *testing.T) {
	err := ValidateActionPlan(ActionPlan{
		Steps: []Step{
			{
				Kind: StepKindEdit,
				Edit: &EditIntent{
					Kind:      EditIntentInsertStatementInCurrentFunction,
					Statement: "x++",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected valid: %v", err)
	}
}

func TestValidateActionPlanReplaceCurrentFunctionBody(t *testing.T) {
	err := ValidateActionPlan(ActionPlan{
		Steps: []Step{
			{
				Kind: StepKindEdit,
				Edit: &EditIntent{
					Kind:    EditIntentReplaceCurrentFunctionBody,
					NewText: `console.log("x");`,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected valid: %v", err)
	}
}

func TestValidateActionPlanEditThenCommand(t *testing.T) {
	err := ValidateActionPlan(ActionPlan{
		Steps: []Step{
			{
				Kind: StepKindEdit,
				Edit: &EditIntent{
					Kind:      EditIntentInsertStatementInCurrentFunction,
					Statement: "x++",
				},
			},
			{
				Kind:       StepKindRunCommand,
				RunCommand: &RunCommandIntent{Command: "pnpm", Args: []string{"test"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected valid: %v", err)
	}
}

func TestValidateActionPlanRejectsBadEditStep(t *testing.T) {
	err := ValidateActionPlan(ActionPlan{
		Steps: []Step{
			{
				Kind: StepKindEdit,
				Edit: &EditIntent{
					Kind:      EditIntentInsertStatementInCurrentFunction,
					Statement: "   ",
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid edit intent")
	}
}

func TestValidateStepRejectsMismatchedPayload(t *testing.T) {
	err := ValidateStep(Step{
		Kind: StepKindEdit,
		Edit: &EditIntent{
			Kind:      EditIntentInsertStatementInCurrentFunction,
			Statement: "ok",
		},
		RunCommand: &RunCommandIntent{Command: "x"},
	})
	if err == nil {
		t.Fatal("expected error for extra runCommand on edit step")
	}
}

func TestValidateActionPlanEmptySteps(t *testing.T) {
	err := ValidateActionPlan(ActionPlan{Steps: nil})
	if err != nil {
		t.Fatalf("empty plan should be valid: %v", err)
	}
}

