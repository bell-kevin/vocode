package agentcontext

import (
	"os"
	"strings"
	"unicode/utf8"

	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// maxActiveFileExcerptRunes limits on-disk active-file text merged into [Gathered.Excerpts].
const maxActiveFileExcerptRunes = 12000

// ReadActiveFileExcerpt returns a rune-capped prefix of the file at path, or empty if unreadable.
func ReadActiveFileExcerpt(absPath string) string {
	p := strings.TrimSpace(absPath)
	if p == "" {
		return ""
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	s := string(b)
	if utf8.RuneCountInString(s) <= maxActiveFileExcerptRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxActiveFileExcerptRunes])
}

// ComposeTurnContext builds a [TurnContext] for one [agent.Agent.NextTurn] call.
func ComposeTurnContext(
	params protocol.VoiceTranscriptParams,
	transcript string,
	succeeded []intents.Intent,
	failed []FailedIntent,
	skipped []intents.Intent,
	intentApplyHistory []IntentApplyRecord,
	gathered Gathered,
	cursor *CursorSymbol,
) TurnContext {
	return TurnContext{
		TranscriptText:     transcript,
		SucceededIntents:   append([]intents.Intent(nil), succeeded...),
		FailedIntents:      append([]FailedIntent(nil), failed...),
		SkippedIntents:     append([]intents.Intent(nil), skipped...),
		IntentApplyHistory: append([]IntentApplyRecord(nil), intentApplyHistory...),
		Editor:             EditorSnapshotFromParams(params, cursor),
		Gathered:           gathered,
	}
}

// EstimatedGatheredBytes approximates wire-ish size for gathered context caps.
func EstimatedGatheredBytes(g Gathered) int {
	total := 0
	for _, sref := range g.Symbols {
		total += len(sref.ID) + len(sref.Name) + len(sref.Path) + len(sref.Kind) + 16
	}
	for _, ex := range g.Excerpts {
		total += len(ex.Path) + len(ex.Content)
	}
	for _, note := range g.Notes {
		total += len(note)
	}
	return total
}
