package undo

import (
	"fmt"

	"vocoding.net/vocode/v2/apps/daemon/internal/intent"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Service validates undo intents and returns protocol undo directives (host applies).
type Service struct{}

func NewService() *Service {
	return &Service{}
}

// DispatchIntent maps a planner undo intent to wire payload for the extension.
func (s *Service) DispatchIntent(u intent.UndoIntent) (protocol.UndoDirective, error) {
	if err := intent.ValidateUndoIntent(u); err != nil {
		return protocol.UndoDirective{}, fmt.Errorf("undo intent: %w", err)
	}
	return protocol.UndoDirective{Scope: string(u.Scope)}, nil
}
