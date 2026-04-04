package globalflow

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"vocoding.net/vocode/v2/apps/core/internal/agent"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type stubCommandModel struct {
	out string
	err error
}

func (s *stubCommandModel) Call(ctx context.Context, req agent.CompletionRequest) (string, error) {
	return s.out, s.err
}

type mapReadHost struct {
	files map[string]string
}

func (m *mapReadHost) ReadHostFile(path string) (string, error) {
	if m == nil || m.files == nil {
		return "", errors.New("missing")
	}
	b, ok := m.files[path]
	if !ok {
		return "", errors.New("not found")
	}
	return b, nil
}

type recordingApply struct {
	last *protocol.HostApplyParams
}

func (r *recordingApply) ApplyDirectives(p protocol.HostApplyParams) (protocol.HostApplyResult, error) {
	if r != nil {
		r.last = &p
	}
	n := len(p.Directives)
	items := make([]protocol.VoiceTranscriptDirectiveApplyItem, n)
	for i := range items {
		items[i] = protocol.VoiceTranscriptDirectiveApplyItem{Status: session.ApplyItemStatusOK}
	}
	return protocol.HostApplyResult{Items: items}, nil
}

func TestHandleCommand_insufficientContext(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	deps := &CommandDeps{
		HostApply:     &recordingApply{},
		ExtensionHost: &mapReadHost{files: map[string]string{}},
		EditModel: &stubCommandModel{
			out: `{"status":"insufficient_context","message":"Open a folder with package.json."}`,
		},
		NewBatchID: func() string { return "batch-1" },
	}
	params := protocol.VoiceTranscriptParams{
		WorkspaceRoot: dir,
		HostPlatform:  "linux",
	}
	res, fail := HandleCommand(deps, params, &session.VoiceSession{}, "install dependencies")
	if !strings.Contains(fail, "package.json") {
		t.Fatalf("fail string: %q", fail)
	}
	if res.Success {
		t.Fatal("expected Success false")
	}
	if !strings.Contains(res.Summary, "package.json") {
		t.Fatalf("summary: %q", res.Summary)
	}
}

func TestHandleCommand_runsApply(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "package.json")
	read := &mapReadHost{files: map[string]string{
		pkgPath: `{"name":"x"}`,
	}}
	rec := &recordingApply{}
	planBytes, err := json.Marshal(map[string]any{
		"status":             "ok",
		"command":            "bash",
		"args":               []string{"-lc", "echo ok"},
		"workingDirectory":   dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	deps := &CommandDeps{
		HostApply:     rec,
		ExtensionHost: read,
		EditModel: &stubCommandModel{
			out: string(planBytes),
		},
		NewBatchID: func() string { return "batch-2" },
	}
	params := protocol.VoiceTranscriptParams{
		WorkspaceRoot: dir,
		HostPlatform:  "linux",
	}
	res, fail := HandleCommand(deps, params, &session.VoiceSession{}, "run tests")
	if fail != "" {
		t.Fatalf("unexpected fail: %q", fail)
	}
	if !res.Success {
		t.Fatalf("expected success: summary=%q", res.Summary)
	}
	if rec.last == nil || len(rec.last.Directives) != 1 {
		t.Fatalf("directives: %+v", rec.last)
	}
	d := rec.last.Directives[0]
	if d.Kind != "command" || d.CommandDirective == nil {
		t.Fatalf("directive: %+v", d)
	}
	if d.CommandDirective.Command != "bash" {
		t.Fatalf("command: %q", d.CommandDirective.Command)
	}
	if d.CommandDirective.WorkingDirectory != dir {
		t.Fatalf("cwd: %q", d.CommandDirective.WorkingDirectory)
	}
}

func TestGatherWorkspaceCommandContext_skipsMissing(t *testing.T) {
	t.Parallel()
	bundle, names := gatherWorkspaceCommandContext(&mapReadHost{files: map[string]string{}}, t.TempDir())
	if bundle != "" || len(names) != 0 {
		t.Fatalf("bundle=%q names=%v", bundle, names)
	}
}

func TestGatherWorkspaceCommandContext_includesFoundFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pkg := filepath.Join(dir, "package.json")
	read := &mapReadHost{files: map[string]string{pkg: `{"name":"x"}`}}
	bundle, names := gatherWorkspaceCommandContext(read, dir)
	if len(names) != 1 || names[0] != "package.json" {
		t.Fatalf("names=%v", names)
	}
	if !strings.Contains(bundle, "package.json") {
		t.Fatalf("bundle: %q", bundle)
	}
}
