package transcript

import (
	"strings"
	"time"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/config"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/voicesession"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func (s *TranscriptService) runExecute(params protocol.VoiceTranscriptParams) (protocol.VoiceTranscriptResult, bool) {
	s.executeMu.Lock()
	defer s.executeMu.Unlock()

	key := strings.TrimSpace(params.ContextSessionId)
	dc := params.DaemonConfig

	idleReset := config.DefaultSessionIdleReset()
	if dc != nil && dc.SessionIdleResetMs != nil {
		ms := *dc.SessionIdleResetMs
		if ms == 0 {
			idleReset = 0
		} else if ms > 0 {
			idleReset = time.Duration(ms) * time.Millisecond
		} else {
			idleReset = config.DefaultSessionIdleReset()
		}
	}
	var vs agentcontext.VoiceSession
	if strings.TrimSpace(key) == "" {
		vs = voicesession.Load(s.sessions, key, idleReset, &s.ephemeralVoiceSession)
	} else {
		vs = voicesession.Load(s.sessions, key, idleReset, nil)
	}

	extSucc, extFail, extSkipped, err := voicesession.ConsumeIncomingApplyReport(&params, &vs)
	if err != nil {
		return protocol.VoiceTranscriptResult{Success: false}, true
	}

	activeFile := strings.TrimSpace(params.ActiveFile)
	maxGatheredBytes := config.DefaultGatheredMaxBytes
	maxGatheredExcerpts := config.DefaultGatheredMaxExcerpts
	if dc != nil {
		if dc.MaxGatheredBytes != nil {
			maxGatheredBytes = int(*dc.MaxGatheredBytes)
		}
		if dc.MaxGatheredExcerpts != nil {
			maxGatheredExcerpts = int(*dc.MaxGatheredExcerpts)
		}
	}

	// In duplex mode, the daemon applies directives immediately and feeds the
	// host's per-directive outcomes back into the next planning iteration.
	maxRepairSteps := config.DefaultMaxRepairSteps
	if dc != nil && dc.MaxTranscriptRepairRpcs != nil {
		maxRepairSteps = int(*dc.MaxTranscriptRepairRpcs)
	}
	if maxRepairSteps < 1 {
		maxRepairSteps = 1
	}

	appliedOkTotal := 0
	appliedFailTotal := 0
	appliedSkippedTotal := 0
	appliedBatchesTotal := 0

	for stepI := 0; stepI < maxRepairSteps; stepI++ {
		vs.Gathered = agentcontext.ApplyGatheredRollingCap(
			vs.Gathered,
			activeFile,
			maxGatheredBytes,
			maxGatheredExcerpts,
		)

		res, g1, pending, ok := s.executor.Execute(params, vs.Gathered, vs.IntentApplyHistory, extSucc, extFail, extSkipped)
		vs.Gathered = g1
		if strings.TrimSpace(key) != "" {
			vs.Gathered = agentcontext.ApplyGatheredRollingCap(vs.Gathered, activeFile, maxGatheredBytes, maxGatheredExcerpts)
		}

		if !ok {
			// Treat executor failure as transcript failure: host UI will show a generic error.
			if strings.TrimSpace(key) == "" {
				voicesession.StoreEphemeralVoiceSession(&s.ephemeralVoiceSession, vs)
			} else {
				voicesession.SaveKeyed(s.sessions, key, vs)
			}
			return protocol.VoiceTranscriptResult{Success: false}, false
		}

		if !res.Success || len(res.Directives) == 0 {
			vs.PendingDirectiveApply = nil
			// Persist before returning.
			if strings.TrimSpace(key) == "" {
				voicesession.StoreEphemeralVoiceSession(&s.ephemeralVoiceSession, vs)
			} else {
				voicesession.SaveKeyed(s.sessions, key, vs)
			}
			// Transcript logging is now a daemon logger concern rather than env-driven tuning.
			return res, ok
		}

		// We have directives; in duplex mode we must call the host for outcomes.
		if pending == nil || s.hostApplyClient == nil {
			vs.PendingDirectiveApply = nil
			if strings.TrimSpace(key) == "" {
				voicesession.StoreEphemeralVoiceSession(&s.ephemeralVoiceSession, vs)
			} else {
				voicesession.SaveKeyed(s.sessions, key, vs)
			}
			return protocol.VoiceTranscriptResult{Success: false}, true
		}

		vs.PendingDirectiveApply = pending

		hostRes, err := s.hostApplyClient.ApplyDirectives(protocol.HostApplyParams{
			ApplyBatchId: pending.ID,
			ActiveFile:   params.ActiveFile,
			Directives:   res.Directives,
		})
		if err != nil {
			vs.PendingDirectiveApply = nil
			if strings.TrimSpace(key) == "" {
				voicesession.StoreEphemeralVoiceSession(&s.ephemeralVoiceSession, vs)
			} else {
				voicesession.SaveKeyed(s.sessions, key, vs)
			}
			return protocol.VoiceTranscriptResult{Success: false}, true
		}

		// Consume the host report to update IntentApplyHistory and compute delta
		// succeeded/failed/skipped intents for this planning iteration.
		extSucc, extFail, extSkipped, err = voicesession.ConsumeHostApplyReport(
			pending.ID,
			hostRes.Items,
			&vs,
		)
		if err != nil {
			vs.PendingDirectiveApply = nil
			if strings.TrimSpace(key) == "" {
				voicesession.StoreEphemeralVoiceSession(&s.ephemeralVoiceSession, vs)
			} else {
				voicesession.SaveKeyed(s.sessions, key, vs)
			}
			return protocol.VoiceTranscriptResult{Success: false}, true
		}

		appliedBatchesTotal++
		appliedOkTotal += len(extSucc)
		appliedFailTotal += len(extFail)
		appliedSkippedTotal += len(extSkipped)
	}

	// Repair-cap hit: directives still outstanding.
	if strings.TrimSpace(key) == "" {
		voicesession.StoreEphemeralVoiceSession(&s.ephemeralVoiceSession, vs)
	} else {
		voicesession.SaveKeyed(s.sessions, key, vs)
	}
	return protocol.VoiceTranscriptResult{Success: false}, true
}
