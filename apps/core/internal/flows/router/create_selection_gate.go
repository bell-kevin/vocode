package router

import (
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// RejectCreateWhenEditorSelection blocks file-create while the editor has a non-empty selection.
// Create only inserts at line boundaries; changing highlighted code is the edit route.
func RejectCreateWhenEditorSelection(p protocol.VoiceTranscriptParams) (protocol.VoiceTranscriptCompletion, string, bool) {
	if !HasNonemptyEditorSelection(p) {
		return protocol.VoiceTranscriptCompletion{}, "", false
	}
	return protocol.VoiceTranscriptCompletion{
			Success: false,
			Summary: "Create is not available while text is selected. Clear the selection to insert new content at a line, or use edit to change the highlighted code.",
		},
		"create: not available while text is selected",
		true
}
