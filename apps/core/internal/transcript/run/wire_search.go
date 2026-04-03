package run

import (
	"time"

	"vocoding.net/vocode/v2/apps/core/internal/transcript/hostdirectives"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/idle"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/searchapply"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func WireSearchEngine(e *Env) {
	if e == nil {
		return
	}
	if e.Search == nil {
		e.Search = &searchapply.TranscriptSearch{}
	}
	e.Search.HostApply = e.HostApply
	e.Search.NewBatchID = hostdirectives.NewApplyBatchID
	e.Search.NavigateHitDirectives = hostdirectives.HitNavigateDirectivesExpand
}

// IdleResetForParams mirrors session idle eviction tuning from RPC params.
func IdleResetForParams(params protocol.VoiceTranscriptParams) time.Duration {
	return idle.SessionResetDuration(params)
}
