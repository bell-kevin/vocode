package rpc

import (
	"encoding/json"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func NewVoiceTranscriptHandler(
	voiceService VoiceTranscriptService,
) Handler {
	return func(
		req protocol.JSONRPCRequest[json.RawMessage],
	) (any, *protocol.JSONRPCErrorObject) {
		params, rpcErr := DecodeParams[protocol.VoiceTranscriptParams](req.Params)
		if rpcErr != nil {
			return nil, rpcErr
		}

		result, ok := voiceService.AcceptTranscript(params)
		if !ok {
			return nil, NewInvalidParamsError()
		}

		return result, nil
	}
}
