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
							Kind:   EditTargetKindSymbol,
							Symbol: &SymbolTarget{SymbolName: "current_function", SymbolKind: "function"},
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
							Kind:   EditTargetKindSymbol,
							Symbol: &SymbolTarget{SymbolName: "current_function", SymbolKind: "function"},
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
							Kind:   EditTargetKindSymbol,
							Symbol: &SymbolTarget{SymbolName: "current_function", SymbolKind: "function"},
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
							Kind:   EditTargetKindSymbol,
							Symbol: &SymbolTarget{SymbolName: "current_function", SymbolKind: "function"},
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
					Kind:   EditTargetKindSymbol,
					Symbol: &SymbolTarget{SymbolName: "current_function", SymbolKind: "function"},
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
