package rpc

import (
	"encoding/json"
	"strings"

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

		result, ok, failureReason := voiceService.AcceptTranscript(params)
		if !ok {
			return nil, NewInvalidParamsError()
		}

		if !result.Success {
			msg := "voice.transcript failed"
			if strings.TrimSpace(failureReason) != "" {
				msg = failureReason
			}
			return nil, &protocol.JSONRPCErrorObject{
				Code:    -32000,
				Message: msg,
			}
		}

		return result, nil
	}
}
