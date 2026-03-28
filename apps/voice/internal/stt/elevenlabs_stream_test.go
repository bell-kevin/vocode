package stt

import "testing"

func TestExtractStreamingEvent(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want StreamingEvent
	}{
		{name: "partial transcript", in: `{"message_type":"partial_transcript","text":"hello world"}`, want: StreamingEvent{Text: "hello world", IsFinal: false}},
		{name: "partial uses transcript field", in: `{"message_type":"partial_transcript","transcript":"hello"}`, want: StreamingEvent{Text: "hello", IsFinal: false}},
		{name: "message_type in type field", in: `{"type":"partial_transcript","text":"x"}`, want: StreamingEvent{Text: "x", IsFinal: false}},
		{name: "committed transcript", in: `{"message_type":"committed_transcript","text":"hello there"}`, want: StreamingEvent{Text: "hello there", IsFinal: true}},
		{name: "committed with timestamps", in: `{"message_type":"committed_transcript_with_timestamps","text":"done"}`, want: StreamingEvent{Text: "done", IsFinal: true}},
		{name: "session started", in: `{"message_type":"session_started","session_id":"s1"}`, want: StreamingEvent{SessionStarted: true}},
		{name: "error payload", in: `{"message_type":"input_error","error":"bad audio"}`, want: StreamingEvent{Error: assertErr("elevenlabs stream input_error: bad audio")}},
		{name: "fallback text only", in: `{"text":"legacy text"}`, want: StreamingEvent{Text: "legacy text"}},
		{name: "empty", in: `{"message_type":"metadata"}`, want: StreamingEvent{}},
		{name: "invalid json", in: `{`, want: StreamingEvent{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStreamingEventPayload([]byte(tt.in))
			if tt.want.Error != nil {
				if got.Error == nil || got.Error.Error() != tt.want.Error.Error() {
					t.Fatalf("expected error %q, got %#v", tt.want.Error.Error(), got.Error)
				}
				return
			}
			if got != tt.want {
				t.Fatalf("expected %#v, got %#v", tt.want, got)
			}
		})
	}
}

func assertErr(msg string) error {
	return errString(msg)
}

type errString string

func (e errString) Error() string { return string(e) }
