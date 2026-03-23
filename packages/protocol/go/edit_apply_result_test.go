package protocol

import "testing"

func TestEditApplyResultValidate(t *testing.T) {
	t.Parallel()

	success := NewEditApplySuccess([]EditAction{{
		Kind: "replace_between_anchors",
		Path: "/tmp/file.ts",
		Anchor: Anchor{
			Before: "before",
			After:  "after",
		},
		NewText: "updated",
	}})
	if err := success.Validate(); err != nil {
		t.Fatalf("expected success to validate, got %v", err)
	}

	failure := NewEditApplyFailure(EditFailure{
		Code:    "validation_failed",
		Message: "bad edit",
	})
	if err := failure.Validate(); err != nil {
		t.Fatalf("expected failure to validate, got %v", err)
	}

	noop := NewEditApplyNoop("No change needed.")
	if err := noop.Validate(); err != nil {
		t.Fatalf("expected noop to validate, got %v", err)
	}
}

func TestEditApplyResultValidateRejectsInvalidCombinations(t *testing.T) {
	t.Parallel()

	invalid := []EditApplyResult{
		{Kind: "success"},
		{
			Kind: "success",
			Actions: []EditAction{},
			Failure: &EditFailure{Code: "validation_failed", Message: "bad"},
		},
		{
			Kind: "failure",
			Failure: &EditFailure{Code: "validation_failed", Message: "bad"},
			Actions: []EditAction{{Kind: "replace_between_anchors"}},
		},
		{Kind: "noop"},
		{Kind: "unknown"},
	}

	for i, candidate := range invalid {
		if err := candidate.Validate(); err == nil {
			t.Fatalf("expected invalid result %d to fail validation", i)
		}
	}
}
