package transcript

import (
	"io"
	"log"
	"os"
	"testing"
	"time"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agent/stub"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/voicesession"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func TestAcceptTranscript_searchControl_exitClearsSession(t *testing.T) {
	t.Helper()
	ag := agent.New(stub.New())
	svc := NewService(ag, log.New(io.Discard, "", 0))
	svc.queue = nil
	t.Setenv("VOCODE_RG_BIN", `C:\Dev\vocode\tools\ripgrep\win32-x64\rg.exe`)

	key := "session-key-exit-1"

	// Put the daemon into a real search-active state by running a search that returns directives.
	root := t.TempDir()
	path := root + "/x.txt"
	if err := os.WriteFile(path, []byte("hello main\n"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	svc.SetHostApplyClient(noopHostApplyClient{})
	res1, ok1, reason1 := svc.AcceptTranscript(protocol.VoiceTranscriptParams{
		ContextSessionId: key,
		Text:             "find main",
		WorkspaceRoot:    root,
		ActiveFile:       path,
	})
	if !ok1 || !res1.Success || reason1 != "" {
		t.Fatalf("search got ok=%v success=%v reason=%q res=%+v", ok1, res1.Success, reason1, res1)
	}
	loaded1 := voicesession.Load(svc.sessions, key, time.Hour, nil)
	if len(loaded1.SearchResults) == 0 {
		t.Fatalf("expected search results persisted in session")
	}

	res, ok, reason := svc.AcceptTranscript(protocol.VoiceTranscriptParams{
		ContextSessionId: key,
		Text:             "close now",
	})
	if !ok || !res.Success || reason != "" {
		t.Fatalf("got ok=%v success=%v reason=%q res=%+v", ok, res.Success, reason, res)
	}
	if res.TranscriptOutcome != "search_control" {
		t.Fatalf("expected outcome=search_control, got %q", res.TranscriptOutcome)
	}
	if res.UiDisposition != "hidden" {
		t.Fatalf("expected uiDisposition=hidden, got %q", res.UiDisposition)
	}

	loaded := voicesession.Load(svc.sessions, key, time.Hour, nil)
	if len(loaded.SearchResults) != 0 {
		t.Fatalf("expected SearchResults cleared, got %+v", loaded.SearchResults)
	}
}

