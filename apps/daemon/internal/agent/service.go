package agent

import protocol "vocoding.net/vocode/v2/packages/protocol/go"

type Service struct {
	planner *Planner
}

func NewService() *Service {
	return &Service{planner: NewPlanner()}
}

func (s *Service) PlanEdit(params protocol.EditApplyParams) EditPlanResult {
	return s.planner.Plan(params)
}
