package run

import (
	"vocoding.net/vocode/v2/apps/core/internal/agent"
	"vocoding.net/vocode/v2/apps/core/internal/flows/router"
	"vocoding.net/vocode/v2/apps/core/internal/rpc"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/searchapply"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
)

// Env holds wiring for transcript execution (sessions, host, router, search apply).
type Env struct {
	Sessions      *session.VoiceSessionStore
	Ephemeral     *session.VoiceSession
	ExtensionHost rpc.ExtensionHost
	EditModel     agent.ModelClient
	FlowRouter    *router.FlowRouter
	Search        *searchapply.TranscriptSearch
}
