package agent

import protocol "vocoding.net/vocode/v2/packages/protocol/go"

type Service struct {
	intentPlanner *IntentPlanner
}

func NewService() *Service {
	return &Service{intentPlanner: NewIntentPlanner()}
}

func (s *Service) PlanEdit(params protocol.EditApplyParams) EditPlanResult {
	return s.intentPlanner.Plan(params)
}
