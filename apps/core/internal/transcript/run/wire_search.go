package run

import (
	"time"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// WireSearchEngine attaches batch/navigation callbacks required by [TranscriptSearch].
func WireSearchEngine(e *Env) {
	if e == nil {
		return
	}
	if e.Search == nil {
		e.Search = &TranscriptSearch{}
	}
	e.Search.HostApply = e.HostApply
	e.Search.NewBatchID = newApplyBatchID
	e.Search.NavigateHitDirectives = hitNavigateDirectives
}

// IdleResetForParams mirrors session idle eviction tuning from RPC params.
func IdleResetForParams(params protocol.VoiceTranscriptParams) time.Duration {
	return idleReset(params)
}
