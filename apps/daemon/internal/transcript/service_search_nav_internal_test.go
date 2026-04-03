package transcript

import (
	"errors"
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

type noopHostApplyClient struct{}

func (noopHostApplyClient) ApplyDirectives(protocol.HostApplyParams) (protocol.HostApplyResult, error) {
	return protocol.HostApplyResult{Items: nil}, nil
}

type errHostApplyClient struct{}

func (errHostApplyClient) ApplyDirectives(protocol.HostApplyParams) (protocol.HostApplyResult, error) {
	return protocol.HostApplyResult{}, errors.New("apply failed")
}

func TestAcceptTranscript_searchControl_returnsSearchControlHiddenAndUpdatesIndex(t *testing.T) {
	t.Helper()
	ag := agent.New(stub.New())
	svc := NewService(ag, log.New(io.Discard, "", 0))
	svc.queue = nil
	svc.SetHostApplyClient(noopHostApplyClient{})

	key := "session-key-nav-1"
	voicesession.SaveKeyed(svc.sessions, key, agentcontext.VoiceSession{
		SearchResults: []agentcontext.SearchHit{
			{Path: "/a.go", Line: 2, Character: 0, Preview: "one"},
			{Path: "/b.go", Line: 4, Character: 3, Preview: "two"},
		},
		ActiveSearchIndex: 0,
	})

	res, ok, reason := svc.AcceptTranscript(protocol.VoiceTranscriptParams{
		ContextSessionId: key,
		Text:             "next",
		ActiveFile:       "/a.go",
	})
	if !ok || !res.Success || reason != "" {
		t.Fatalf("got ok=%v success=%v reason=%q res=%+v", ok, res.Success, reason, res)
	}
	if res.Search == nil || len(res.Search.Results) == 0 {
		t.Fatalf("expected search results in completion, got %+v", res.Search)
	}
	if res.UiDisposition != "hidden" {
		t.Fatalf("expected uiDisposition=hidden, got %q", res.UiDisposition)
	}
	if res.Search.ActiveIndex == nil || *res.Search.ActiveIndex != 1 {
		t.Fatalf("expected search.activeIndex=1, got %+v", res.Search.ActiveIndex)
	}

	loaded := voicesession.Load(svc.sessions, key, time.Hour, nil)
	if loaded.ActiveSearchIndex != 1 {
		t.Fatalf("expected stored activeSearchIndex=1, got %d", loaded.ActiveSearchIndex)
	}
}

func TestAcceptTranscript_searchNav_hostApplyError_revertsIndex(t *testing.T) {
	t.Parallel()
	ag := agent.New(stub.New())
	svc := NewService(ag, log.New(io.Discard, "", 0))
	svc.queue = nil
	svc.SetHostApplyClient(errHostApplyClient{})

	key := "session-key-nav-apply-err"
	voicesession.SaveKeyed(svc.sessions, key, agentcontext.VoiceSession{
		SearchResults: []agentcontext.SearchHit{
			{Path: "/a.go", Line: 2, Character: 0, Preview: "one"},
			{Path: "/b.go", Line: 4, Character: 3, Preview: "two"},
		},
		ActiveSearchIndex: 0,
	})

	res, ok, reason := svc.AcceptTranscript(protocol.VoiceTranscriptParams{
		ContextSessionId: key,
		Text:             "next",
		ActiveFile:       "/a.go",
	})
	if !ok || res.Success || reason == "" {
		t.Fatalf("expected failure, ok=%v success=%v reason=%q", ok, res.Success, reason)
	}
	if reason != "host.applyDirectives failed: apply failed" {
		t.Fatalf("unexpected reason: %q", reason)
	}

	loaded := voicesession.Load(svc.sessions, key, time.Hour, nil)
	if loaded.ActiveSearchIndex != 0 {
		t.Fatalf("expected index reverted to 0, got %d", loaded.ActiveSearchIndex)
	}
}
