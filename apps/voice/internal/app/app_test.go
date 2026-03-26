package app

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRun_EmitsReadyAndShutdownState(t *testing.T) {
	in := strings.NewReader("{\"type\":\"shutdown\"}\n")
	var out bytes.Buffer

	a := New(in, &out)
	if err := a.Run(); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, `"type":"ready"`) {
		t.Fatalf("expected ready event, got output: %s", got)
	}
	if !strings.Contains(got, `"state":"shutdown"`) {
		t.Fatalf("expected shutdown state, got output: %s", got)
	}
}

func TestSttMode_DefaultsToBatch(t *testing.T) {
	t.Setenv("VOCODE_VOICE_STT_MODE", "")
	if got := sttMode(); got != "batch" {
		t.Fatalf("expected batch, got %q", got)
	}
}

func TestSttMode_ParsesStreamAliases(t *testing.T) {
	for _, v := range []string{"stream", "streaming", "websocket", "ws"} {
		t.Setenv("VOCODE_VOICE_STT_MODE", v)
		if got := sttMode(); got != "stream" {
			t.Fatalf("mode %q: expected stream, got %q", v, got)
		}
	}
}

func TestSttMode_InvalidFallsBackToBatch(t *testing.T) {
	t.Setenv("VOCODE_VOICE_STT_MODE", "unknown")
	if got := sttMode(); got != "batch" {
		t.Fatalf("expected batch fallback, got %q", got)
	}
}

func TestVADEnabled_DefaultTrue(t *testing.T) {
	t.Setenv("VOCODE_VOICE_VAD_ENABLED", "")
	if !vadEnabled() {
		t.Fatalf("expected VAD enabled by default")
	}
}

func TestVADEnabled_ParsesFalse(t *testing.T) {
	t.Setenv("VOCODE_VOICE_VAD_ENABLED", "false")
	if vadEnabled() {
		t.Fatalf("expected VAD disabled")
	}
}

func TestAppendRollingContext(t *testing.T) {
	got := appendRollingContext("", "hello world", 500)
	if got != "hello world" {
		t.Fatalf("unexpected initial context: %q", got)
	}

	got = appendRollingContext(got, "second phrase", 20)
	if got != "world second phrase" {
		t.Fatalf("expected tail-trimmed context, got %q", got)
	}
}

func TestMain(m *testing.M) {
	// Ensure env from prior tests doesn't leak into package tests.
	_ = os.Unsetenv("VOCODE_VOICE_STT_MODE")
	os.Exit(m.Run())
}

