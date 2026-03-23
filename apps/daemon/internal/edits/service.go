package edits

import (
	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type Service struct {
	actionBuilder *ActionBuilder
}

func NewService() *Service {
	return &Service{
		actionBuilder: NewActionBuilder(),
	}
}

func (s *Service) BuildActions(params protocol.EditApplyParams, plan agent.EditPlan) ([]protocol.EditAction, *protocol.EditFailure) {
	return s.actionBuilder.BuildActions(params, plan)
}

func editFailure(code string, message string) *protocol.EditFailure {
	return &protocol.EditFailure{Code: code, Message: message}
}
