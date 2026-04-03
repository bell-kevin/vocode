package agentcontext

import (
	"fmt"
	"strings"
)

// Wire names for clarifyTargetResolution (per-flow registry; must match executor/service).
const (
	ClarifyTargetQuestion = "question"
	// ClarifyTargetWorkspaceSelect / ClarifyTargetSelectFile match apps/core/transcript/clarify registry.
	ClarifyTargetWorkspaceSelect = "workspace_select"
	ClarifyTargetSelectFile      = "select_file"
	// ClarifyTargetInstruction is Main-flow scoped edit (classifier instruction + scope intent).
	ClarifyTargetInstruction = "instruction"
	// ClarifyTargetEdit is Selection-flow locked-match edit.
	ClarifyTargetEdit         = "edit"
	ClarifyTargetRename       = "rename"
	ClarifyTargetMove         = "move"
	ClarifyTargetOpen         = "open"
	ClarifyTargetCreateFile   = "create_file"
	ClarifyTargetCreateFolder = "create_folder"
)

// ClarifyTargetAllowed is true iff target names a resolution with can_clarify in parentFlowKind.
func ClarifyTargetAllowed(parentFlowKind, target string) bool {
	t := strings.TrimSpace(target)
	switch parentFlowKind {
	case FlowKindMain:
		switch t {
		case ClarifyTargetQuestion, ClarifyTargetWorkspaceSelect, ClarifyTargetSelectFile, ClarifyTargetInstruction:
			return true
		}
	case FlowKindSelection:
		return t == ClarifyTargetEdit
	case FlowKindFileSelection:
		switch t {
		case ClarifyTargetRename, ClarifyTargetMove, ClarifyTargetOpen, ClarifyTargetCreateFile, ClarifyTargetCreateFolder:
			return true
		}
	}
	return false
}

// ValidateClarifyTargetResolution rejects invalid or non–can_clarify targets for the parent flow.
func ValidateClarifyTargetResolution(parentFlowKind, target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("clarify requires non-empty clarifyTargetResolution for flow %q", parentFlowKind)
	}
	if !ClarifyTargetAllowed(parentFlowKind, target) {
		return fmt.Errorf("clarify targetResolution %q is not allowed (can_clarify) for flow %q", target, parentFlowKind)
	}
	return nil
}
