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
	e.Search.HostApply = e.ExtensionHost
	e.Search.ExtensionHost = e.ExtensionHost
	e.Search.NewBatchID = hostdirectives.NewApplyBatchID
	e.Search.NavigateHitDirectives = func(params protocol.VoiceTranscriptParams, path string, line0, char0, length int) []protocol.VoiceTranscriptDirective {
		syms := hostdirectives.DocumentSymbolsForPath(e.ExtensionHost, params, path)
		return hostdirectives.HitNavigateDirectivesExpandWithSymbols(path, line0, char0, length, syms)
	}
}

// IdleResetForParams mirrors session idle eviction tuning from RPC params.
func IdleResetForParams(params protocol.VoiceTranscriptParams) time.Duration {
	return idle.SessionResetDuration(params)
}
