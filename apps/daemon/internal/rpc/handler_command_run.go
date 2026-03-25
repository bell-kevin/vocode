package rpc

import (
	"encoding/json"
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func NewCommandRunHandler(commandService CommandRunService) Handler {
	return func(
		req protocol.JSONRPCRequest[json.RawMessage],
	) (any, *protocol.JSONRPCErrorObject) {
		params, rpcErr := DecodeParams[protocol.CommandRunParams](req.Params)
		if rpcErr != nil {
			return nil, rpcErr
		}

		if strings.TrimSpace(params.Command) == "" {
			return nil, NewInvalidParamsError()
		}

		return commandService.Run(params), nil
	}
}

