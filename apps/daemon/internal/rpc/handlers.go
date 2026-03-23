package rpc

import protocol "vocoding.net/vocode/v2/packages/protocol/go"

type HandlerDefinition struct {
	Method  string
	Handler Handler
}

type EditApplyService interface {
	Apply(params protocol.EditApplyParams) (protocol.EditApplyResult, error)
}

func BuildHandlers(editService EditApplyService) []HandlerDefinition {
	return []HandlerDefinition{
		{
			Method:  "ping",
			Handler: NewPingHandler(),
		},
		{
			Method:  "edit/apply",
			Handler: NewEditApplyHandler(editService),
		},
	}
}
