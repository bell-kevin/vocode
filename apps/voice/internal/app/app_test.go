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
	if !strings.Contains(got, `"transcript_committed_field":true`) {
		t.Fatalf("expected ready event to advertise transcript_committed_field, got output: %s", got)
	}
	if !strings.Contains(got, `"state":"shutdown"`) {
		t.Fatalf("expected shutdown state, got output: %s", got)
	}
}

