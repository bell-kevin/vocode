package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"os"
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type rawRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	// Allow larger JSON messages later without instantly choking.
	const maxScanTokenSize = 1024 * 1024
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxScanTokenSize)

	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req rawRequest
		if err := json.Unmarshal(line, &req); err != nil {
			log.Printf("failed to parse request: %v", err)
			_ = encoder.Encode(protocol.JSONRPCErrorResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error: protocol.JSONRPCErrorObject{
					Code:    -32700,
					Message: "Parse error",
				},
			})
			continue
		}

		switch req.Method {
		case "ping":
			handlePing(encoder, req)
		case "edit/apply":
			handleEditApply(encoder, req)
		default:
			id := req.ID
			_ = encoder.Encode(protocol.JSONRPCErrorResponse{
				JSONRPC: "2.0",
				ID:      &id,
				Error: protocol.JSONRPCErrorObject{
					Code:    -32601,
					Message: "Method not found",
				},
			})
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		log.Printf("stdin scanner error: %v", err)
	}
}

func handlePing(encoder *json.Encoder, req rawRequest) {
	var params protocol.PingParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			id := req.ID
			_ = encoder.Encode(protocol.JSONRPCErrorResponse{
				JSONRPC: "2.0",
				ID:      &id,
				Error: protocol.JSONRPCErrorObject{
					Code:    -32602,
					Message: "Invalid params",
				},
			})
			return
		}
	}

	resp := protocol.JSONRPCResponse[protocol.PingResult]{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: protocol.PingResult{
			Message: "pong",
		},
	}

	_ = encoder.Encode(resp)
}

func handleEditApply(encoder *json.Encoder, req rawRequest) {
	var params protocol.EditApplyParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		id := req.ID
		_ = encoder.Encode(protocol.JSONRPCErrorResponse{
			JSONRPC: "2.0",
			ID:      &id,
			Error: protocol.JSONRPCErrorObject{
				Code:    -32602,
				Message: "Invalid params",
			},
		})
		return
	}

	log.Printf("edit/apply instruction=%q activeFile=%q", params.Instruction, params.ActiveFile)

	before, after, ok := firstBraceAnchors(params.FileText)
	if !ok {
		resp := protocol.JSONRPCResponse[protocol.EditApplyResult]{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: protocol.EditApplyResult{
				Actions: []protocol.EditAction{},
			},
		}
		_ = encoder.Encode(resp)
		return
	}

	action := protocol.ReplaceBetweenAnchorsAction{
		Kind: "replace_between_anchors",
		Path: params.ActiveFile,
		Anchor: protocol.Anchor{
			Before: before,
			After:  after,
		},
		NewText: "\n  console.log(\"hi from vocode\");\n",
	}

	resp := protocol.JSONRPCResponse[protocol.EditApplyResult]{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: protocol.EditApplyResult{
			Actions: []protocol.EditAction{action},
		},
	}

	_ = encoder.Encode(resp)
}

func firstBraceAnchors(fileText string) (before string, after string, ok bool) {
	lines := strings.Split(fileText, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(line, "{") && trimmed != "{" {
			return line, "}", true
		}
	}

	return "", "", false
}
