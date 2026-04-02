package agentcontext

import "testing"

func TestClarifyTargetAllowedMain(t *testing.T) {
	for _, target := range []string{
		ClarifyTargetQuestion,
		ClarifyTargetSelection,
		ClarifyTargetFileSelection,
		ClarifyTargetInstruction,
	} {
		if !ClarifyTargetAllowed(FlowKindMain, target) {
			t.Errorf("main should allow %q", target)
		}
	}
	if ClarifyTargetAllowed(FlowKindMain, ClarifyTargetEdit) {
		t.Error("main should not allow edit")
	}
	if ClarifyTargetAllowed(FlowKindMain, ClarifyTargetRename) {
		t.Error("main should not allow rename")
	}
}

func TestClarifyTargetAllowedSelection(t *testing.T) {
	if !ClarifyTargetAllowed(FlowKindSelection, ClarifyTargetEdit) {
		t.Error("selection should allow edit")
	}
	if ClarifyTargetAllowed(FlowKindSelection, ClarifyTargetInstruction) {
		t.Error("selection should not allow instruction")
	}
}

func TestClarifyTargetAllowedFileSelection(t *testing.T) {
	for _, target := range []string{
		ClarifyTargetRename,
		ClarifyTargetMove,
		ClarifyTargetOpen,
		ClarifyTargetCreateFile,
		ClarifyTargetCreateFolder,
	} {
		if !ClarifyTargetAllowed(FlowKindFileSelection, target) {
			t.Errorf("file_selection should allow %q", target)
		}
	}
	if ClarifyTargetAllowed(FlowKindFileSelection, ClarifyTargetInstruction) {
		t.Error("file_selection should not allow instruction")
	}
}

func TestValidateClarifyTargetResolution(t *testing.T) {
	if err := ValidateClarifyTargetResolution(FlowKindMain, ""); err == nil {
		t.Error("empty target should fail")
	}
	if err := ValidateClarifyTargetResolution(FlowKindMain, "bogus"); err == nil {
		t.Error("bogus main target should fail")
	}
	if err := ValidateClarifyTargetResolution(FlowKindMain, ClarifyTargetInstruction); err != nil {
		t.Errorf("valid target: %v", err)
	}
}
