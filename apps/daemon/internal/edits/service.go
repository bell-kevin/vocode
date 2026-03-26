package edits

import (
	"vocoding.net/vocode/v2/apps/daemon/internal/actionplan"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type Service struct {
	actionBuilder *ActionBuilder
}

func NewService() *Service {
	return &Service{actionBuilder: NewActionBuilder()}
}

func (s *Service) BuildActions(params protocol.EditApplyParams, intent actionplan.EditIntent) ([]protocol.EditAction, *protocol.EditFailure) {
	return s.actionBuilder.BuildActions(params, intent)
}

// ApplyIntent validates [actionplan.EditIntent], runs [Service.BuildActions], and
// returns a protocol-level [protocol.EditApplyResult] (success, failure, or noop).
func (s *Service) ApplyIntent(params protocol.EditApplyParams, intent actionplan.EditIntent) (protocol.EditApplyResult, error) {
	if err := actionplan.ValidateEditIntent(intent); err != nil {
		f := protocol.EditFailure{Code: "unsupported_instruction", Message: err.Error()}
		result := protocol.NewEditApplyFailure(f)
		return result, result.Validate()
	}
	actions, failure := s.BuildActions(params, intent)
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
