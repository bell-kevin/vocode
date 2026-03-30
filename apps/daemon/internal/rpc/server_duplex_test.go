package rpc

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"testing"
	"time"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func TestServerDuplexOutboundRequest(t *testing.T) {
	daemonStdinR, extensionStdinW := io.Pipe()
	extensionStdoutR, daemonStdoutW := io.Pipe()

	router := NewRouter(log.New(io.Discard, "", 0))
	var srv *Server

	router.Register("test", func(req protocol.JSONRPCRequest[json.RawMessage]) (any, *protocol.JSONRPCErrorObject) {
		_, err := srv.ApplyDirectives(protocol.HostApplyParams{
			ApplyBatchId: "batch-1",
			ActiveFile:  "test.js",
			Directives:  []protocol.VoiceTranscriptDirective{},
		})
		if err != nil {
			return nil, NewInternalError(err)
		}
		return map[string]any{"ok": true}, nil
	})

	srv = NewServer(ServerOptions{
		Logger: log.New(io.Discard, "", 0),
		Stdin:  daemonStdinR,
		Stdout: daemonStdoutW,
		Router: router,
	})

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- srv.Run()
	}()

	extensionErrCh := make(chan error, 1)
	go func() {
		// Send a single inbound request to the daemon.
		_, err := extensionStdinW.Write([]byte(`{"jsonrpc":"2.0","id":1,"method":"test","params":{}}` + "\n"))
		if err != nil {
			extensionErrCh <- err
			return
		}

		sc := bufio.NewScanner(extensionStdoutR)
		for sc.Scan() {
			line := sc.Bytes()
			if len(line) == 0 {
				continue
			}
			var msg map[string]any
			if err := json.Unmarshal(line, &msg); err != nil {
				extensionErrCh <- err
				return
			}

			// Daemon outbound JSON-RPC request (host.applyDirectives).
			if method, ok := msg["method"].(string); ok && method != "" {
				if method != "host.applyDirectives" {
					continue
				}
				idFloat, _ := msg["id"].(float64)
				id := int64(idFloat)
				resp := map[string]any{
					"jsonrpc": "2.0",
					"id":      id,
					"result": map[string]any{
						"items": []any{
							map[string]any{"status": "ok"},
						},
					},
				}
				b, _ := json.Marshal(resp)
				if _, err := extensionStdinW.Write(append(b, '\n')); err != nil {
					extensionErrCh <- err
					return
				}
				continue
			}

			// Final response to inbound "test".
			if _, ok := msg["result"]; ok {
				_ = extensionStdinW.Close()
				break
			}
		}

		if err := sc.Err(); err != nil {
			extensionErrCh <- err
			return
		}
		extensionErrCh <- nil
	}()

	select {
	case err := <-serverErrCh:
		if err != nil {
			t.Fatalf("server.Run returned error: %v", err)
		}
	case err := <-extensionErrCh:
		if err != nil {
			t.Fatalf("extension simulator returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for duplex JSON-RPC exchange")
	}
}

