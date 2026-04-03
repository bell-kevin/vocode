package transcript

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"vocoding.net/vocode/v2/apps/core/internal/flows/router"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/clarify"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func TestCancelSelection_clearsSelectionAndDismissesClarify(t *testing.T) {
	s := NewService(router.NewFlowRouter(nil))
	*s.env.Ephemeral = session.VoiceSession{
		BasePhase:             session.BasePhaseSelection,
		SearchResults:         []session.SearchHit{{Path: "x.go", Line: 0, Character: 0, Preview: ""}},
		ActiveSearchIndex:     0,
		PendingDirectiveApply: &session.DirectiveApplyBatch{ID: "b", NumDirectives: 1},
		Clarify: &session.ClarifyOverlay{
			TargetResolution:   "edit",
			Question:           "Q",
			OriginalTranscript: "orig",
		},
	}

	res, ok, _ := s.AcceptTranscript(protocol.VoiceTranscriptParams{
		ControlRequest: "cancel_selection",
	})
	if !ok || !res.Success {
		t.Fatalf("expected ok=true success=true; got ok=%v res=%+v", ok, res)
	}
	if res.Search == nil || !res.Search.Closed {
		t.Fatalf("expected search.closed on cancel_selection, got %+v", res.Search)
	}

	if s.env.Ephemeral.BasePhase != session.BasePhaseMain {
		t.Fatalf("expected base phase main after cancel_selection, got %q", s.env.Ephemeral.BasePhase)
	}
	if s.env.Ephemeral.Clarify != nil {
		t.Fatalf("expected clarify overlay dismissed")
	}
	if s.env.Ephemeral.SearchResults != nil {
		t.Fatalf("expected SearchResults cleared")
	}
	if s.env.Ephemeral.ActiveSearchIndex != 0 {
		t.Fatalf("expected ActiveSearchIndex=0 after cancel_selection, got %d", s.env.Ephemeral.ActiveSearchIndex)
	}
}

func TestClarifyAnswer_editWhileSelection_closesSelection(t *testing.T) {
	s := NewService(router.NewFlowRouter(nil))
	*s.env.Ephemeral = session.VoiceSession{
		BasePhase:         session.BasePhaseSelection,
		SearchResults:     []session.SearchHit{{Path: "x.go", Line: 0, Character: 0, Preview: ""}},
		ActiveSearchIndex: 0,
		Clarify: &session.ClarifyOverlay{
			TargetResolution:   clarify.ClarifyTargetEdit,
			Question:           "Which file?",
			OriginalTranscript: "find foo",
		},
	}

	res, ok, _ := s.AcceptTranscript(protocol.VoiceTranscriptParams{
		Text: "my answer",
	})
	if !ok || !res.Success {
		t.Fatalf("expected ok=true success=true; got ok=%v res=%+v", ok, res)
	}
	if res.Clarify != nil || res.Search != nil || res.Question != nil {
		t.Fatalf("expected no grouped fields after clarify resolution, got %+v", res)
	}

	if s.env.Ephemeral.BasePhase != session.BasePhaseMain {
		t.Fatalf("expected base phase main after clarify(edit), got %q", s.env.Ephemeral.BasePhase)
	}
	if s.env.Ephemeral.Clarify != nil {
		t.Fatalf("expected clarify overlay dismissed")
	}
	if s.env.Ephemeral.SearchResults != nil {
		t.Fatalf("expected SearchResults cleared after edit while in selection")
	}
}

func TestFileSelectionNavigation_nextUpdatesFocus(t *testing.T) {
	root := t.TempDir()
	if err := writeFile(root, "a.go"); err != nil {
		t.Fatalf("write a.go: %v", err)
	}
	if err := writeFile(root, "b.go"); err != nil {
		t.Fatalf("write b.go: %v", err)
	}
	if err := writeFile(root, "c.go"); err != nil {
		t.Fatalf("write c.go: %v", err)
	}

	paths := []string{
		filepath.Join(root, "a.go"),
		filepath.Join(root, "b.go"),
		filepath.Join(root, "c.go"),
	}
	sort.Strings(paths)
	expected := paths[1]

	s := NewService(router.NewFlowRouter(nil))
	*s.env.Ephemeral = session.VoiceSession{
		BasePhase:          session.BasePhaseFileSelection,
		FileSelectionPaths: paths,
		FileSelectionIndex: 0,
		FileSelectionFocus: paths[0],
	}

	res, ok, _ := s.AcceptTranscript(protocol.VoiceTranscriptParams{
		WorkspaceRoot: root,
		Text:          "next",
	})
	if !ok || !res.Success {
		t.Fatalf("expected ok=true success=true; got ok=%v res=%+v", ok, res)
	}
	if res.FileSelection == nil || !res.FileSelection.NavigatingList || res.FileSelection.FocusPath != expected {
		t.Fatalf("expected fileSelection navigation focus %q, got %+v", expected, res.FileSelection)
	}
}

func TestFileSelectionExit_doneReturnsMain(t *testing.T) {
	root := t.TempDir()
	if err := writeFile(root, "a.go"); err != nil {
		t.Fatalf("write a.go: %v", err)
	}

	s := NewService(router.NewFlowRouter(nil))
	*s.env.Ephemeral = session.VoiceSession{
		BasePhase: session.BasePhaseFileSelection,
	}

	res, ok, _ := s.AcceptTranscript(protocol.VoiceTranscriptParams{
		WorkspaceRoot: root,
		Text:          "done",
	})
	if !ok || !res.Success {
		t.Fatalf("expected ok=true success=true; got ok=%v res=%+v", ok, res)
	}
	if res.Search != nil || res.FileSelection != nil {
		t.Fatalf("expected no search/fileSelection after file-selection exit, got %+v / %+v", res.Search, res.FileSelection)
	}
	if s.env.Ephemeral.BasePhase != session.BasePhaseMain {
		t.Fatalf("expected base phase main after exit, got %q", s.env.Ephemeral.BasePhase)
	}
	if s.env.Ephemeral.FileSelectionPaths != nil {
		t.Fatalf("expected FileSelectionPaths cleared")
	}
}

func writeFile(dir, name string) error {
	p := filepath.Join(dir, name)
	return os.WriteFile(p, []byte("x"), 0o644)
}
