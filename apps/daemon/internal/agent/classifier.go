package agent

import (
	"fmt"
	"strings"
)

type TranscriptKind string

const (
	TranscriptInstruction TranscriptKind = "instruction"
	TranscriptSearch      TranscriptKind = "search"
	TranscriptQuestion    TranscriptKind = "question"
	TranscriptIrrelevant  TranscriptKind = "irrelevant"
)

type TranscriptClassifierResult struct {
	Kind       TranscriptKind
	SearchQuery string
	AnswerText string
}

func (r TranscriptClassifierResult) Validate() error {
	switch r.Kind {
	case TranscriptInstruction, TranscriptIrrelevant:
		if strings.TrimSpace(r.SearchQuery) != "" || strings.TrimSpace(r.AnswerText) != "" {
			return fmt.Errorf("classifier: %s must not set searchQuery/answerText", r.Kind)
		}
		return nil
	case TranscriptSearch:
		if strings.TrimSpace(r.SearchQuery) == "" {
			return fmt.Errorf("classifier: search requires searchQuery")
		}
		if strings.TrimSpace(r.AnswerText) != "" {
			return fmt.Errorf("classifier: search must not set answerText")
		}
		return nil
	case TranscriptQuestion:
		if strings.TrimSpace(r.AnswerText) == "" {
			return fmt.Errorf("classifier: question requires answerText")
		}
		if strings.TrimSpace(r.SearchQuery) != "" {
			return fmt.Errorf("classifier: question must not set searchQuery")
		}
		return nil
	default:
		return fmt.Errorf("classifier: unknown kind %q", r.Kind)
	}
}

