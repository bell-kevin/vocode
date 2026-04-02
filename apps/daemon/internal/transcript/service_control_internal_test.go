package transcript

import (
	"io"
	"log"
	"testing"
	"time"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agent/stub"
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/voicesession"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func TestAcceptTranscript_controlCancelSearch_clearsKeyedSession(t *testing.T) {
	t.Helper()
	ag := agent.New(stub.New())
	svc := NewService(ag, log.New(io.Discard, "", 0))
	svc.queue = nil

	key := "session-key-1"
	vs := agentcontext.VoiceSession{
		SearchResults: []agentcontext.SearchHit{
			{Path: "/x.go", Line: 2, Character: 0, Preview: "z"},
		},
		ActiveSearchIndex: 0,
	}
	voicesession.SaveKeyed(svc.sessions, key, vs)

	res, ok, reason := svc.AcceptTranscript(protocol.VoiceTranscriptParams{
		ContextSessionId: key,
		ControlRequest:   "cancel_selection",
	})
	if !ok || !res.Success || reason != "" {
		t.Fatalf("got ok=%v success=%v reason=%q res=%+v", ok, res.Success, reason, res)
	}
	if res.UiDisposition != "hidden" {
		t.Fatalf("expected uiDisposition=hidden, got %q", res.UiDisposition)
	}

	loaded := voicesession.Load(svc.sessions, key, time.Hour, nil)
	if len(loaded.SearchResults) != 0 {
		t.Fatalf("expected SearchResults cleared, got %+v", loaded.SearchResults)
	}
}

func TestAcceptTranscript_controlCancelSearch_clearsEphemeralSession(t *testing.T) {
	t.Helper()
	ag := agent.New(stub.New())
	svc := NewService(ag, log.New(io.Discard, "", 0))
	svc.queue = nil

	ephemeral := agentcontext.VoiceSession{
		SearchResults: []agentcontext.SearchHit{
			{Path: "/y.go", Line: 1, Character: 0, Preview: "q"},
		},
		ActiveSearchIndex: 0,
	}
	voicesession.StoreEphemeralVoiceSession(&svc.ephemeralVoiceSession, ephemeral)

	res, ok, reason := svc.AcceptTranscript(protocol.VoiceTranscriptParams{
		ControlRequest: "cancel_selection",
	})
	if !ok || !res.Success {
		t.Fatalf("got ok=%v success=%v reason=%q", ok, res.Success, reason)
	}
	if res.UiDisposition != "hidden" {
		t.Fatalf("expected uiDisposition=hidden, got %q", res.UiDisposition)
	}

	loaded := voicesession.Load(svc.sessions, "", time.Hour, &svc.ephemeralVoiceSession)
	if len(loaded.SearchResults) != 0 {
		t.Fatalf("expected ephemeral search cleared")
	}
}

func TestAcceptTranscript_controlCancelClarify_ok(t *testing.T) {
	t.Helper()
	ag := agent.New(stub.New())
	svc := NewService(ag, log.New(io.Discard, "", 0))
	svc.queue = nil

	res, ok, reason := svc.AcceptTranscript(protocol.VoiceTranscriptParams{
		ContextSessionId: "any",
		ControlRequest:   "cancel_clarify",
	})
	if !ok || !res.Success || reason != "" {
		t.Fatalf("got ok=%v success=%v reason=%q", ok, res.Success, reason)
	}
	if res.UiDisposition != "hidden" {
		t.Fatalf("expected uiDisposition=hidden, got %q", res.UiDisposition)
	}
	_ = res
}

func TestAcceptTranscript_unknownControl_fails(t *testing.T) {
	t.Helper()
	ag := agent.New(stub.New())
	svc := NewService(ag, log.New(io.Discard, "", 0))
	svc.queue = nil

	_, ok, reason := svc.AcceptTranscript(protocol.VoiceTranscriptParams{
		ContextSessionId: "k",
		ControlRequest:   "bogus",
	})
	if ok || reason != "unknown controlRequest" {
		t.Fatalf("expected unknown control failure, ok=%v reason=%q", ok, reason)
	}
}
