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

type voiceTranscriptServiceStub struct{}

func (s *voiceTranscriptServiceStub) AcceptTranscript(
	params protocol.VoiceTranscriptParams,
) (protocol.VoiceTranscriptCompletion, bool, string) {
	if strings.TrimSpace(params.Text) == "" {
		return protocol.VoiceTranscriptCompletion{}, false, ""
	}

	return protocol.VoiceTranscriptCompletion{Success: true}, true, ""
}

type voiceTranscriptServiceFunc func(params protocol.VoiceTranscriptParams) (protocol.VoiceTranscriptCompletion, bool, string)

func (f voiceTranscriptServiceFunc) AcceptTranscript(
	params protocol.VoiceTranscriptParams,
) (protocol.VoiceTranscriptCompletion, bool, string) {
	return f(params)
}

func runSingleRequest(
	t *testing.T,
	voiceService VoiceTranscriptService,
	requestLine string,
) map[string]any {
	t.Helper()

	router := NewRouter(log.New(io.Discard, "", 0))
	for _, def := range BuildHandlers(voiceService) {
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

func TestServerVoiceTranscriptSuccess(t *testing.T) {
	t.Parallel()

	request := `{"jsonrpc":"2.0","id":1,"method":"voice.transcript","params":{"text":"hello world"}}`
	response := runSingleRequest(
		t,
		&voiceTranscriptServiceStub{},
		request,
	)

	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result object, got: %#v", response["result"])
	}

	if got := result["success"]; got != true {
		t.Fatalf("expected success=true, got %#v", got)
	}
}

func TestServerVoiceTranscriptFailureReturnsError(t *testing.T) {
	t.Parallel()

	request := `{"jsonrpc":"2.0","id":1,"method":"voice.transcript","params":{"text":"hello world"}}`

	response := runSingleRequest(
		t,
		voiceTranscriptServiceFunc(func(params protocol.VoiceTranscriptParams) (protocol.VoiceTranscriptCompletion, bool, string) {
			return protocol.VoiceTranscriptCompletion{Success: false}, true, "openai: HTTP 401: invalid api key"
		}),
		request,
	)

	errorObject, ok := response["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got: %#v", response)
	}
	if got := errorObject["message"]; got != "openai: HTTP 401: invalid api key" {
		t.Fatalf("expected error message, got %#v", got)
	}
}

func TestServerVoiceTranscriptRejectsEmptyText(t *testing.T) {
	t.Parallel()

	request := `{"jsonrpc":"2.0","id":1,"method":"voice.transcript","params":{"text":"   "}}`
	response := runSingleRequest(
		t,
		&voiceTranscriptServiceStub{},
		request,
	)

	errorObject, ok := response["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got: %#v", response)
	}

	if got := errorObject["code"]; got != float64(-32602) {
		t.Fatalf("expected invalid params error code -32602, got %#v", got)
	}
}
