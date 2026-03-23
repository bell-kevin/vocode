package agent

import protocol "vocoding.net/vocode/v2/packages/protocol/go"

type EditPlan struct {
	Intent EditIntent
}

type EditPlanResult struct {
	Plan    *EditPlan
	Failure *protocol.EditFailure
}
