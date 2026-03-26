package rpc

import protocol "vocoding.net/vocode/v2/packages/protocol/go"

type HandlerDefinition struct {
	Method  string
	Handler Handler
}

type VoiceTranscriptService interface {
	// AcceptTranscript returns ok=false for semantically invalid params.
	AcceptTranscript(params protocol.VoiceTranscriptParams) (protocol.VoiceTranscriptResult, bool)
}

func BuildHandlers(voiceService VoiceTranscriptService) []HandlerDefinition {
	return []HandlerDefinition{
		{
			Method:  "ping",
			Handler: NewPingHandler(),
		},
		{
			Method:  "voice.transcript",
			Handler: NewVoiceTranscriptHandler(voiceService),
		},
	}
}
