package actionplan

import "testing"

func TestValidateActionPlanSingleEdit(t *testing.T) {
	err := ValidateActionPlan(ActionPlan{
		Steps: []Step{
			{
				Kind: StepKindEdit,
				Edit: &EditIntent{
					Kind: EditIntentKindInsert,
					Insert: &InsertEditIntent{
						Target: EditTarget{
							Kind:     EditTargetKindSymbolID,
							SymbolID: &SymbolIDTarget{ID: "v1|Zm9vLnRz|1|ZnVuY3Rpb24|Y3VycmVudF9mdW5jdGlvbg"},
						},
						Text: "x++",
					},
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
					Kind: EditIntentKindReplace,
					Replace: &ReplaceEditIntent{
						Target: EditTarget{
							Kind:     EditTargetKindSymbolID,
							SymbolID: &SymbolIDTarget{ID: "v1|Zm9vLnRz|1|ZnVuY3Rpb24|Y3VycmVudF9mdW5jdGlvbg"},
						},
						NewText: `console.log("x");`,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected valid: %v", err)
	}
}

func TestValidateActionPlanReplaceBySymbolID(t *testing.T) {
	err := ValidateActionPlan(ActionPlan{
		Steps: []Step{
			{
				Kind: StepKindEdit,
				Edit: &EditIntent{
					Kind: EditIntentKindReplace,
					Replace: &ReplaceEditIntent{
						Target: EditTarget{
							Kind:     EditTargetKindSymbolID,
							SymbolID: &SymbolIDTarget{ID: "v1|Zm9v|1|ZnVuY3Rpb24|YmFy"},
						},
						NewText: `console.log("x");`,
					},
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
					Kind: EditIntentKindInsert,
					Insert: &InsertEditIntent{
						Target: EditTarget{
							Kind:     EditTargetKindSymbolID,
							SymbolID: &SymbolIDTarget{ID: "v1|Zm9vLnRz|1|ZnVuY3Rpb24|Y3VycmVudF9mdW5jdGlvbg"},
						},
						Text: "x++",
					},
				},
			},
			{
				Kind:       StepKindRunCommand,
				RunCommand: &CommandIntent{Command: "pnpm", Args: []string{"test"}},
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
					Kind: EditIntentKindInsert,
					Insert: &InsertEditIntent{
						Target: EditTarget{
							Kind:     EditTargetKindSymbolID,
							SymbolID: &SymbolIDTarget{ID: "v1|Zm9vLnRz|1|ZnVuY3Rpb24|Y3VycmVudF9mdW5jdGlvbg"},
						},
						Text: "   ",
					},
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
			Kind: EditIntentKindInsert,
			Insert: &InsertEditIntent{
				Target: EditTarget{
					Kind:     EditTargetKindSymbolID,
					SymbolID: &SymbolIDTarget{ID: "v1|Zm9vLnRz|1|ZnVuY3Rpb24|Y3VycmVudF9mdW5jdGlvbg"},
				},
				Text: "ok",
			},
		},
		RunCommand: &CommandIntent{Command: "x"},
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
