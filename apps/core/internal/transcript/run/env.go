package run

import (
	"vocoding.net/vocode/v2/apps/core/internal/flows/router"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Env holds dependencies for transcript execution (sessions, host, router, search).
type Env struct {
	Sessions   *session.VoiceSessionStore
	Ephemeral  *session.VoiceSession
	HostApply  hostApplyClient
	FlowRouter *router.FlowRouter
	Search     *TranscriptSearch
}

type hostApplyClient interface {
	ApplyDirectives(protocol.HostApplyParams) (protocol.HostApplyResult, error)
}
