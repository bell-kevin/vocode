package rpc

import protocol "vocoding.net/vocode/v2/packages/protocol/go"

type HandlerDefinition struct {
	Method  string
	Handler Handler
}

type EditApplyService interface {
	Apply(params protocol.EditApplyParams) (protocol.EditApplyResult, error)
}

type VoiceTranscriptService interface {
	// AcceptTranscript returns ok=false for semantically invalid params.
	AcceptTranscript(params protocol.VoiceTranscriptParams) (protocol.VoiceTranscriptResult, bool)
}

func BuildHandlers(editService EditApplyService, voiceService VoiceTranscriptService) []HandlerDefinition {
	return []HandlerDefinition{
		{
			Method:  "ping",
			Handler: NewPingHandler(),
		},
		{
			Method:  "edit/apply",
			Handler: NewEditApplyHandler(editService),
		},
		{
			Method:  "voice.transcript",
			Handler: NewVoiceTranscriptHandler(voiceService),
		},
	}
}
