package rpc

import protocol "vocoding.net/vocode/v2/packages/protocol/go"

type HandlerDefinition struct {
	Method  string
	Handler Handler
}

type VoiceTranscriptService interface {
	// AcceptTranscript returns ok=false for semantically invalid params.
	// If ok=true and failureReason is non-empty, the handler will return a JSON-RPC error
	// so the extension can show a useful failure message.
	AcceptTranscript(params protocol.VoiceTranscriptParams) (protocol.VoiceTranscriptCompletion, bool, string)
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
