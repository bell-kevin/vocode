package agent

import (
	"fmt"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
)

// TurnKind discriminates one model completion for the transcript executor.
type TurnKind string

const (
	TurnIrrelevant     TurnKind = "irrelevant"
	TurnDone           TurnKind = "done"
	TurnRequestContext TurnKind = "request_context"
	TurnIntents        TurnKind = "intents"
)

// TurnResult is exactly one variant: irrelevant, done, request_context, or a non-empty intent batch.
type TurnResult struct {
	Kind             TurnKind
	IrrelevantReason string
	DoneSummary      string
	RequestContext   *intents.RequestContextIntent
	Intents          []intents.Intent
}

// Validate checks the turn union invariant and nested intents.
func (t TurnResult) Validate() error {
	switch t.Kind {
	case TurnIrrelevant:
		if t.RequestContext != nil || len(t.Intents) > 0 || strings.TrimSpace(t.DoneSummary) != "" {
			return fmt.Errorf("agent turn: irrelevant must not set other fields")
		}
	case TurnDone:
		if t.RequestContext != nil || len(t.Intents) > 0 || strings.TrimSpace(t.IrrelevantReason) != "" {
			return fmt.Errorf("agent turn: done must not set other fields")
		}
	case TurnRequestContext:
		if t.RequestContext == nil || len(t.Intents) > 0 ||
			strings.TrimSpace(t.IrrelevantReason) != "" || strings.TrimSpace(t.DoneSummary) != "" {
			return fmt.Errorf("agent turn: request_context requires requestContext only")
		}
		rc := intents.ControlRequestContext(t.RequestContext)
		if err := rc.Validate(); err != nil {
			return err
		}
	case TurnIntents:
		if t.RequestContext != nil || strings.TrimSpace(t.IrrelevantReason) != "" || strings.TrimSpace(t.DoneSummary) != "" {
			return fmt.Errorf("agent turn: intents must not set control-only fields")
		}
		if len(t.Intents) == 0 {
			return fmt.Errorf("agent turn: intents must be non-empty")
		}
		for i := range t.Intents {
			if err := t.Intents[i].Validate(); err != nil {
				return fmt.Errorf("agent turn: intents[%d]: %w", i, err)
			}
		}
	default:
		return fmt.Errorf("agent turn: unknown kind %q", t.Kind)
	}
	return nil
}
