package app

import (
	"bytes"
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

func TestSttModelID_Default(t *testing.T) {
	t.Setenv("ELEVENLABS_STT_MODEL_ID", "")
	t.Setenv("STT_MODEL_ID", "")
	if got := sttModelID(); got != "scribe_v2" {
		t.Fatalf("expected default model scribe_v2, got %q", got)
	}
}

func TestSttModelID_NewNamePreferred(t *testing.T) {
	t.Setenv("STT_MODEL_ID", "scribe_v1")
	t.Setenv("ELEVENLABS_STT_MODEL_ID", "scribe_v2")
	if got := sttModelID(); got != "scribe_v2" {
		t.Fatalf("expected ELEVENLABS_STT_MODEL_ID to win, got %q", got)
	}
}

func TestStreamChunkConfig_Defaults(t *testing.T) {
	t.Setenv("VOCODE_VOICE_STREAM_MIN_CHUNK_MS", "")
	t.Setenv("VOCODE_VOICE_STREAM_MAX_CHUNK_MS", "")
	t.Setenv("VOCODE_VOICE_STREAM_MAX_UTTERANCE_MS", "")
	if got := streamMinChunkMS(); got != 200 {
		t.Fatalf("expected default min chunk 200ms, got %d", got)
	}
	if got := streamMaxChunkMS(); got != 500 {
		t.Fatalf("expected default max chunk 500ms, got %d", got)
	}
	if got := streamMaxUtteranceMS(); got != 4000 {
		t.Fatalf("expected default max utterance 4000ms, got %d", got)
	}
}

