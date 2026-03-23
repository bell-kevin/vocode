package rpc

import (
	"encoding/json"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func NewEditApplyHandler(editService EditApplyService) Handler {
	return func(
		req protocol.JSONRPCRequest[json.RawMessage],
	) (any, *protocol.JSONRPCErrorObject) {
		params, rpcErr := DecodeParams[protocol.EditApplyParams](req.Params)
		if rpcErr != nil {
			return nil, rpcErr
		}

		result, err := editService.Apply(params)
		if err != nil {
			return nil, NewInternalError(err)
		}

		return result, nil
	}
}
