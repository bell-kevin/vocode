package transcript

import (
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/config"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/voicesession"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func (s *TranscriptService) runExecute(params protocol.VoiceTranscriptParams) (protocol.VoiceTranscriptResult, bool) {
	s.executeMu.Lock()
	defer s.executeMu.Unlock()

	key := strings.TrimSpace(params.ContextSessionId)
	idleReset := config.SessionIdleReset()
	vs := voicesession.Load(s.sessions, key, idleReset, s.ephemeralPendingDirectiveApply)

	extSucc, extFail, err := voicesession.ConsumeIncomingApplyReport(&params, &vs)
	if err != nil {
		return protocol.VoiceTranscriptResult{Success: false}, true
	}

	activeFile := strings.TrimSpace(params.ActiveFile)
	maxGatheredBytes := config.Int("VOCODE_DAEMON_GATHERED_MAX_BYTES", 120_000)
	maxGatheredExcerpts := config.Int("VOCODE_DAEMON_GATHERED_MAX_EXCERPTS", 12)
	vs.Gathered = agentcontext.ApplyGatheredRollingCap(vs.Gathered, activeFile, maxGatheredBytes, maxGatheredExcerpts)

	res, g1, pending, ok := s.executor.Execute(params, vs.Gathered, extSucc, extFail)
	vs.Gathered = g1
	if strings.TrimSpace(key) != "" {
		vs.Gathered = agentcontext.ApplyGatheredRollingCap(vs.Gathered, activeFile, maxGatheredBytes, maxGatheredExcerpts)
	}

	if ok && res.Success && len(res.Directives) > 0 && pending != nil {
		vs.PendingDirectiveApply = pending
	} else {
		vs.PendingDirectiveApply = nil
	}

	if strings.TrimSpace(key) == "" {
		voicesession.StoreEphemeralPending(&s.ephemeralPendingDirectiveApply, vs)
	} else {
		voicesession.SaveKeyed(s.sessions, key, vs)
	}

	return res, ok
}
