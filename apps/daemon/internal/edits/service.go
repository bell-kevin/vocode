package edits

import (
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

func (s *Service) BuildActions(ctx EditExecutionContext, editIntent intent.EditIntent) ([]protocol.EditAction, *protocol.EditFailure) {
	return s.actionBuilder.BuildActions(ctx, editIntent)
}

// ApplyIntent validates [intent.EditIntent], runs [Service.BuildActions], and
// returns a protocol-level [protocol.EditApplyResult] (success, failure, or noop).
func (s *Service) ApplyIntent(ctx EditExecutionContext, editIntent intent.EditIntent) (protocol.EditApplyResult, error) {
	if err := intent.ValidateEditIntent(editIntent); err != nil {
		f := protocol.EditFailure{Code: "unsupported_instruction", Message: err.Error()}
		result := protocol.NewEditApplyFailure(f)
		return result, result.Validate()
	}
	actions, failure := s.BuildActions(ctx, editIntent)
	if failure != nil {
		if failure.Code == "no_change_needed" {
			result := protocol.NewEditApplyNoop(failure.Message)
			return result, result.Validate()
		}
		result := protocol.NewEditApplyFailure(*failure)
		return result, result.Validate()
	}
	result := protocol.NewEditApplySuccess(actions)
	return result, result.Validate()
}

func editFailure(code string, message string) *protocol.EditFailure {
	return &protocol.EditFailure{Code: code, Message: message}
}
