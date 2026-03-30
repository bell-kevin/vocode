package agentcontext

import (
	"fmt"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Directive apply item status values (wire: voice-transcript.directive-apply-item.schema.json).
const (
	ApplyItemStatusOK      = "ok"
	ApplyItemStatusFailed  = "failed"
	ApplyItemStatusSkipped = "skipped"
)

// IntentApplyStatus is the daemon-side status for one intent in [IntentApplyRecord].
type IntentApplyStatus string

const (
	IntentApplyStatusOK      IntentApplyStatus = "ok"
	IntentApplyStatusFailed  IntentApplyStatus = "failed"
	IntentApplyStatusSkipped IntentApplyStatus = "skipped"
)

// IntentApplyRecord is one intent from a host apply batch with its outcome (cumulative transcript context).
type IntentApplyRecord struct {
	BatchOrdinal int
	IndexInBatch int
	Intent       intents.Intent
	Status       IntentApplyStatus
	Message      string
}

// AppendIntentApplyHistory appends one record per directive in a reported batch (same order as SourceIntents).
func AppendIntentApplyHistory(
	history []IntentApplyRecord,
	batchOrdinal int,
	source []intents.Intent,
	items []protocol.VoiceTranscriptDirectiveApplyItem,
) ([]IntentApplyRecord, error) {
	if len(source) != len(items) {
		return nil, fmt.Errorf("intent apply history: source and apply items length mismatch")
	}
	for i := range source {
		st, err := parseApplyItemStatus(items[i].Status)
		if err != nil {
			return nil, err
		}
		history = append(history, IntentApplyRecord{
			BatchOrdinal: batchOrdinal,
			IndexInBatch: i,
			Intent:       source[i],
			Status:       st,
			Message:      strings.TrimSpace(items[i].Message),
		})
	}
	return history, nil
}

func parseApplyItemStatus(s string) (IntentApplyStatus, error) {
	switch strings.TrimSpace(s) {
	case ApplyItemStatusOK:
		return IntentApplyStatusOK, nil
	case ApplyItemStatusFailed:
		return IntentApplyStatusFailed, nil
	case ApplyItemStatusSkipped:
		return IntentApplyStatusSkipped, nil
	default:
		return "", fmt.Errorf("intent apply history: unknown apply status %q", s)
	}
}
