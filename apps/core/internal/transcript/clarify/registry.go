package clarify

import "strings"

// Clarify target resolutions are the protocol strings the controller passes through.
// These must match the agent/executor semantics in later porting.
const (
	ClarifyTargetQuestion        = "question"
	ClarifyTargetWorkspaceSelect = "workspace_select"
	ClarifyTargetSelectFile      = "select_file"

	ClarifyTargetEdit = "edit"

	ClarifyTargetRename       = "rename"
	ClarifyTargetMove         = "move"
	ClarifyTargetOpen         = "open"
	ClarifyTargetCreateFile   = "create_file"
	ClarifyTargetCreateFolder = "create_folder"
)

type BaseFlowKind string

const (
	BaseFlowMain            BaseFlowKind = "main"
	BaseFlowWorkspaceSelect BaseFlowKind = "workspace_select"
	BaseFlowSelectFile      BaseFlowKind = "select_file"
)

// ClarifyTargetAllowed is true iff target names a resolution allowed to be clarified under parent/base phase.
func ClarifyTargetAllowed(parentFlowKind BaseFlowKind, target string) bool {
	t := strings.TrimSpace(target)
	switch parentFlowKind {
	case BaseFlowMain:
		switch t {
		case ClarifyTargetQuestion, ClarifyTargetWorkspaceSelect, ClarifyTargetSelectFile:
			return true
		}
	case BaseFlowWorkspaceSelect:
		return t == ClarifyTargetEdit
	case BaseFlowSelectFile:
		switch t {
		case ClarifyTargetRename, ClarifyTargetMove, ClarifyTargetOpen, ClarifyTargetCreateFile, ClarifyTargetCreateFolder:
			return true
		}
	}
	return false
}

func ValidateClarifyTargetResolution(parentFlowKind BaseFlowKind, target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return &validationError{msg: "clarify requires non-empty clarifyTargetResolution"}
	}
	if !ClarifyTargetAllowed(parentFlowKind, target) {
		return &validationError{msg: "clarify targetResolution is not allowed for this base phase"}
	}
	return nil
}

type validationError struct{ msg string }

func (e *validationError) Error() string { return e.msg }
