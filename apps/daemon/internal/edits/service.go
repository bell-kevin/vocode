package edits

import (
	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type Service struct {
	agentService *agent.Service
	planner      *Planner
}

func NewService(agentService *agent.Service) *Service {
	return &Service{
		agentService: agentService,
		planner:      NewPlanner(),
	}
}

func (s *Service) Apply(params protocol.EditApplyParams) (protocol.EditApplyResult, error) {
	planResult := s.agentService.PlanEdit(params)
	if planResult.Failure != nil {
		return protocol.EditApplyResult{
			Actions: []protocol.EditAction{},
			Failure: planResult.Failure,
		}, nil
	}

	actions, failure := s.planner.BuildActions(params, *planResult.Plan)
	if failure != nil {
		return protocol.EditApplyResult{
			Actions: []protocol.EditAction{},
			Failure: failure,
		}, nil
	}

	return protocol.EditApplyResult{Actions: actions}, nil
}

func editFailure(code string, message string) *protocol.EditFailure {
	return &protocol.EditFailure{Code: code, Message: message}
}
