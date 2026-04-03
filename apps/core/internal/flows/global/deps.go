package globalflow

import (
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// WorkspaceSearchApply runs workspace search for voice flows and applies host navigation.
type WorkspaceSearchApply interface {
	SearchFromQuery(params protocol.VoiceTranscriptParams, q string, vs *session.VoiceSession) (protocol.VoiceTranscriptCompletion, bool, string)
	FileSearchFromQuery(params protocol.VoiceTranscriptParams, q string, vs *session.VoiceSession) (protocol.VoiceTranscriptCompletion, bool, string)
}

// RouteDeps carries dependencies shared by global route handlers (select, select_file, …).
type RouteDeps struct {
	Search WorkspaceSearchApply
}
