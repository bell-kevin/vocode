package app

import (
	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/edits"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type EditOrchestrator struct {
	agent editPlanner
	edits actionBuilder
}

type editPlanner interface {
	PlanEdit(params protocol.EditApplyParams) agent.EditPlanResult
}

type actionBuilder interface {
	BuildActions(params protocol.EditApplyParams, plan agent.EditPlan) ([]protocol.EditAction, *protocol.EditFailure)
}

func NewEditOrchestrator(agentService *agent.Service, editService *edits.Service) *EditOrchestrator {
	return &EditOrchestrator{
		agent: agentService,
		edits: editService,
	}
}

func (o *EditOrchestrator) Apply(params protocol.EditApplyParams) (protocol.EditApplyResult, error) {
	planResult := o.agent.PlanEdit(params)
	if planResult.Failure != nil {
		result := protocol.NewEditApplyFailure(*planResult.Failure)
		return result, result.Validate()
	}

	actions, failure := o.edits.BuildActions(params, *planResult.Plan)
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
