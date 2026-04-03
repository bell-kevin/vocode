package router

import (
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Context is the minimal input for route classification: which flow we are in and what the user said.
// Per-route handlers build their own prompts and context when they call a model.
type Context struct {
	Flow                 flows.ID
	Instruction          string
	ActiveFile           string
	HasNonemptySelection bool
	WorkspaceRoot        string
	HostPlatform         string
	WorkspaceFolderOpen  bool
}

// ContextForClassification builds router context from host params (editor/selection awareness for the model).
func ContextForClassification(flow flows.ID, instruction string, p protocol.VoiceTranscriptParams) Context {
	return Context{
		Flow:                 flow,
		Instruction:          strings.TrimSpace(instruction),
		ActiveFile:           strings.TrimSpace(p.ActiveFile),
		HasNonemptySelection: hasNonemptyEditorSelection(p),
		WorkspaceRoot:        strings.TrimSpace(p.WorkspaceRoot),
		HostPlatform:         strings.TrimSpace(p.HostPlatform),
		WorkspaceFolderOpen:  p.WorkspaceFolderOpen,
	}
}

func hasNonemptyEditorSelection(p protocol.VoiceTranscriptParams) bool {
	sel := p.ActiveSelection
	if sel == nil {
		return false
	}
	return sel.StartLine != sel.EndLine || sel.StartChar != sel.EndChar
}
