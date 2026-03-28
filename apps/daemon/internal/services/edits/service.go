package edits

import (
	"fmt"

	"vocoding.net/vocode/v2/apps/daemon/internal/intent"
	"vocoding.net/vocode/v2/apps/daemon/internal/symbols"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type Service struct {
	actionBuilder *ActionBuilder
}

func NewService() *Service {
	return &Service{actionBuilder: NewActionBuilder()}
}

func NewServiceWithResolver(resolver symbols.Resolver) *Service {
	return &Service{actionBuilder: NewActionBuilderWithResolver(resolver)}
}

func (s *Service) BuildActions(ctx EditExecutionContext, editIntent intent.EditIntent) ([]protocol.EditAction, *EditBuildFailure) {
	return s.actionBuilder.BuildActions(ctx, editIntent)
}

// DispatchIntent validates [intent.EditIntent], runs [Service.BuildActions], and
// returns a protocol-level [protocol.EditDirective] (success or noop).
func (s *Service) DispatchIntent(ctx EditExecutionContext, editIntent intent.EditIntent) (protocol.EditDirective, error) {
	if err := intent.ValidateEditIntent(editIntent); err != nil {
		return protocol.EditDirective{}, err
	}
	actions, failure := s.BuildActions(ctx, editIntent)
	if failure != nil {
		if failure.Code == "no_change_needed" {
			result := protocol.NewEditDirectiveNoop(failure.Message)
			return result, result.Validate()
		}
		return protocol.EditDirective{}, fmt.Errorf("edit dispatch failed: %s", failure.Message)
	}
	result := protocol.NewEditDirectiveSuccess(actions)
	return result, result.Validate()
}
