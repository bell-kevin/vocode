package rpc

import (
	"encoding/json"
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type VoiceTranscriptParams struct {
	Text string `json:"text"`
}

type VoiceTranscriptResult struct {
	Accepted bool `json:"accepted"`
}

func NewVoiceTranscriptHandler() Handler {
	return func(
		req protocol.JSONRPCRequest[json.RawMessage],
	) (any, *protocol.JSONRPCErrorObject) {
		params, rpcErr := DecodeParams[VoiceTranscriptParams](req.Params)
		if rpcErr != nil {
			return nil, rpcErr
		}

		if strings.TrimSpace(params.Text) == "" {
			return nil, NewInvalidParamsError()
		}

		return VoiceTranscriptResult{Accepted: true}, nil
	}
}
