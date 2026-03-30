// Package voicesession loads and saves per-context transcript state: [agentcontext.VoiceSession]
// in [agentcontext.VoiceSessionStore], plus process-local pending directive batch when
// params.contextSessionId is empty.
package voicesession

import (
	"fmt"
	"strings"
	"time"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Load returns session state for this RPC. When contextKey is empty, restores the full
// ephemeral [agentcontext.VoiceSession] (gathered, apply history, pending batch) from *ephemeral.
func Load(store *agentcontext.VoiceSessionStore, contextKey string, idleReset time.Duration, ephemeral *agentcontext.VoiceSession) agentcontext.VoiceSession {
	key := strings.TrimSpace(contextKey)
	if key == "" {
		if ephemeral == nil {
			return agentcontext.VoiceSession{}
		}
		return agentcontext.CloneVoiceSession(*ephemeral)
	}
	return store.Get(key, idleReset)
}

// SaveKeyed persists vs when contextKey is non-empty.
func SaveKeyed(store *agentcontext.VoiceSessionStore, contextKey string, vs agentcontext.VoiceSession) {
	key := strings.TrimSpace(contextKey)
	if key == "" || store == nil {
		return
	}
	store.Put(key, vs)
}

// StoreEphemeralVoiceSession copies vs into *dst for the next RPC when contextSessionId is empty.
func StoreEphemeralVoiceSession(dst *agentcontext.VoiceSession, vs agentcontext.VoiceSession) {
	*dst = agentcontext.CloneVoiceSession(vs)
}

// ConsumeIncomingApplyReport consumes the host's apply report
// without mutating params (unlike ConsumeIncomingApplyReport).
func ConsumeIncomingApplyReport(
	params *protocol.VoiceTranscriptParams,
	vs *agentcontext.VoiceSession,
) ([]intents.Intent, []agentcontext.FailedIntent, []intents.Intent, error) {
	items := params.LastBatchApply
	reportID := strings.TrimSpace(params.ReportApplyBatchId)
	return ConsumeHostApplyReport(reportID, items, vs)
}

// ConsumeHostApplyReport consumes the host's apply outcomes for the currently
// pending directive batch, updates intent apply history, and clears the pending
// batch on success.
func ConsumeHostApplyReport(
	reportID string,
	items []protocol.VoiceTranscriptDirectiveApplyItem,
	vs *agentcontext.VoiceSession,
) ([]intents.Intent, []agentcontext.FailedIntent, []intents.Intent, error) {
	if len(items) == 0 {
		vs.PendingDirectiveApply = nil
		return nil, nil, nil, nil
	}
	if vs.PendingDirectiveApply == nil {
		return nil, nil, nil, fmt.Errorf("lastBatchApply without pending directive apply batch")
	}
	batch := vs.PendingDirectiveApply
	extSucc, extFail, extSkipped, err := batch.ConsumeHostApplyReport(reportID, items)
	if err != nil {
		return nil, nil, nil, err
	}
	hist, err := agentcontext.AppendIntentApplyHistory(vs.IntentApplyHistory, vs.NextApplyBatchOrdinal, batch.SourceIntents, items)
	if err != nil {
		return nil, nil, nil, err
	}
	vs.IntentApplyHistory = hist
	vs.NextApplyBatchOrdinal++
	vs.PendingDirectiveApply = nil
	return extSucc, extFail, extSkipped, nil
}
