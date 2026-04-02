package run

import (
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/executor"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/voicesession"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// HostApply is the extension callback for directive batches (same contract as transcript service).
type HostApply interface {
	ApplyDirectives(protocol.HostApplyParams) (protocol.HostApplyResult, error)
}

// Env carries dependencies for [Execute]. The transcript service constructs this per locked call.
type Env struct {
	Sessions  *agentcontext.VoiceSessionStore
	Ephemeral *agentcontext.VoiceSession
	Executor  *executor.Executor
	HostApply HostApply
}

func (e *Env) persistSession(contextKey string, vs agentcontext.VoiceSession) {
	if strings.TrimSpace(contextKey) == "" {
		voicesession.StoreEphemeralVoiceSession(e.Ephemeral, vs)
		return
	}
	voicesession.SaveKeyed(e.Sessions, contextKey, vs)
}
