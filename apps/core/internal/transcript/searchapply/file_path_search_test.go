package searchapply

import (
	"os"
	"path/filepath"
	"testing"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type fakeHostApply struct {
	lastBatch string
	nDirectives int
}

func (f *fakeHostApply) ApplyDirectives(p protocol.HostApplyParams) (protocol.HostApplyResult, error) {
	f.lastBatch = p.ApplyBatchId
	f.nDirectives = len(p.Directives)
	items := make([]protocol.VoiceTranscriptDirectiveApplyItem, len(p.Directives))
	for i := range items {
		items[i] = protocol.VoiceTranscriptDirectiveApplyItem{Status: "ok"}
	}
	return protocol.HostApplyResult{Items: items}, nil
}

func TestFileSearchFromQuery_findsByPathNotBody(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "pkg", "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(nested, "test.js")
	if err := os.WriteFile(path, []byte("// no filename substring in body\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	host := &fakeHostApply{}
	e := &TranscriptSearch{
		HostApply:  host,
		NewBatchID: func() string { return "batch-test" },
	}
	params := protocol.VoiceTranscriptParams{
		WorkspaceRoot: root,
		ActiveFile:    "",
	}
	comp, handled, msg := e.FileSearchFromQuery(params, "test.js", nil)
	if !handled || msg != "" {
		t.Fatalf("unexpected: handled=%v msg=%q", handled, msg)
	}
	if !comp.Success {
		t.Fatalf("success=false")
	}
	if comp.FileSelection == nil || comp.FileSelection.NoHits {
		t.Fatalf("expected hits, got %#v", comp.FileSelection)
	}
	if len(comp.FileSelection.Results) == 0 {
		t.Fatal("no results")
	}
	if filepath.Clean(comp.FileSelection.Results[0].Path) != filepath.Clean(path) {
		t.Fatalf("first path %q want %q", comp.FileSelection.Results[0].Path, path)
	}
	if host.nDirectives != 1 {
		t.Fatalf("host directives: %d", host.nDirectives)
	}
}
