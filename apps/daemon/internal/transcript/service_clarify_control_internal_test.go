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

func TestAcceptTranscript_clarifyControl_cancelClearsState(t *testing.T) {
	t.Helper()
	ag := agent.New(stub.New())
	svc := NewService(ag, log.New(io.Discard, "", 0))
	svc.queue = nil

	key := "session-key-clarify-1"
	voicesession.SaveKeyed(svc.sessions, key, agentcontext.VoiceSession{
		FlowStack: []agentcontext.FlowFrame{{
			Kind:                      agentcontext.FlowKindClarify,
			ClarifyTargetResolution:   "instruction",
			ClarifyQuestion:           "Which file?",
			ClarifyOriginalTranscript: "fix thing",
		}},
	})

	res, ok, reason := svc.AcceptTranscript(protocol.VoiceTranscriptParams{
		ContextSessionId: key,
		Text:             "quit please",
	})
	if !ok || !res.Success || reason != "" {
		t.Fatalf("got ok=%v success=%v reason=%q res=%+v", ok, res.Success, reason, res)
	}
	if res.Clarify != nil {
		t.Fatalf("expected no clarify group on cancel, got %+v", res.Clarify)
	}
	if res.UiDisposition != "hidden" {
		t.Fatalf("expected uiDisposition=hidden, got %q", res.UiDisposition)
	}

	loaded := voicesession.Load(svc.sessions, key, time.Hour, nil)
	if agentcontext.FlowTopKind(loaded.FlowStack) == agentcontext.FlowKindClarify {
		t.Fatalf("expected clarify frame popped, stack=%+v", loaded.FlowStack)
	}
}
