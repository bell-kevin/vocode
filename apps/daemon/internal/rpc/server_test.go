package rpc

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"strings"
	"testing"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type editApplyServiceStub struct {
	result protocol.EditApplyResult
	err    error
}

func (s *editApplyServiceStub) Apply(_ protocol.EditApplyParams) (protocol.EditApplyResult, error) {
	return s.result, s.err
}

func runSingleRequest(t *testing.T, service EditApplyService, requestLine string) map[string]any {
	t.Helper()

	router := NewRouter(log.New(io.Discard, "", 0))
	for _, def := range BuildHandlers(service) {
		router.Register(def.Method, def.Handler)
	}

	stdin := bytes.NewBufferString(requestLine + "\n")
	stdout := &bytes.Buffer{}
	server := NewServer(ServerOptions{
		Logger: log.New(io.Discard, "", 0),
		Stdin:  stdin,
		Stdout: stdout,
		Router: router,
	})

	if err := server.Run(); err != nil {
		t.Fatalf("server.Run() returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected exactly one response line, got %d: %q", len(lines), stdout.String())
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &response); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}
	return response
}

func validEditApplyRequestLine(t *testing.T) string {
	t.Helper()
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "edit/apply",
		"params": map[string]any{
			"instruction": "insert statement `console.log(\"done\")` inside current function",
			"activeFile":  "/tmp/example.ts",
			"fileText":    "function a() {\n}\n",
		},
	}
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	return string(raw)
}

func TestServerEditApplySuccessResult(t *testing.T) {
	t.Parallel()

	response := runSingleRequest(t, &editApplyServiceStub{
		result: protocol.NewEditApplySuccess([]protocol.EditAction{
			{
				Kind: "replace_between_anchors",
				Path: "/tmp/example.ts",
				Anchor: protocol.Anchor{
					Before: "function a() {",
					After:  "}",
				},
				NewText: "\n  console.log(\"done\");\n",
			},
		}),
	}, validEditApplyRequestLine(t))

	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result object, got: %#v", response["result"])
	}
	if got := result["kind"]; got != "success" {
		t.Fatalf("expected success kind, got %#v", got)
	}
}

func TestServerEditApplyFailureResult(t *testing.T) {
	t.Parallel()

	response := runSingleRequest(t, &editApplyServiceStub{
		result: protocol.NewEditApplyFailure(protocol.EditFailure{
			Code:    "validation_failed",
			Message: "unsafe edit",
		}),
	}, validEditApplyRequestLine(t))

	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result object, got: %#v", response["result"])
	}
	if got := result["kind"]; got != "failure" {
		t.Fatalf("expected failure kind, got %#v", got)
	}
}

func TestServerEditApplyNoopResult(t *testing.T) {
	t.Parallel()

	response := runSingleRequest(t, &editApplyServiceStub{
		result: protocol.NewEditApplyNoop("No change needed."),
	}, validEditApplyRequestLine(t))

	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result object, got: %#v", response["result"])
	}
	if got := result["kind"]; got != "noop" {
		t.Fatalf("expected noop kind, got %#v", got)
	}
}

func TestServerEditApplyRejectsInvalidMixedResult(t *testing.T) {
	t.Parallel()

	response := runSingleRequest(t, &editApplyServiceStub{
		result: protocol.EditApplyResult{
			Kind: "success",
		},
	}, validEditApplyRequestLine(t))

	errorObject, ok := response["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got: %#v", response)
	}

	if got := errorObject["code"]; got != float64(-32000) {
		t.Fatalf("expected internal error code -32000, got %#v", got)
	}
}
