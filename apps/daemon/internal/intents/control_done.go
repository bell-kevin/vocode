package intents

import "fmt"

// DoneIntent is optional payload for the done control intent: a human-readable summary of
// what the agent did, shown by the extension.
type DoneIntent struct {
	Summary string `json:"summary,omitempty"`
}

const maxDoneSummaryRunes = 8192

func validateDoneIntent(d *DoneIntent) error {
	if d == nil {
		return nil
	}
	if len([]rune(d.Summary)) > maxDoneSummaryRunes {
		return fmt.Errorf("intent: done summary exceeds %d characters", maxDoneSummaryRunes)
	}
	return nil
}
