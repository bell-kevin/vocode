package transcript_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agent/stub"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type applyHost struct {
	t *testing.T
}

func (h *applyHost) ApplyDirectives(params protocol.HostApplyParams) (protocol.HostApplyResult, error) {
	h.t.Helper()

	out := protocol.HostApplyResult{Items: make([]protocol.VoiceTranscriptDirectiveApplyItem, 0, len(params.Directives))}
	for i := range params.Directives {
		d := params.Directives[i]
		item := protocol.VoiceTranscriptDirectiveApplyItem{Status: "ok"}

		switch d.Kind {
		case "edit":
			if d.EditDirective == nil || d.EditDirective.Kind != "success" {
				item.Status = "failed"
				item.Message = "host apply: edit directive missing or not success"
				out.Items = append(out.Items, item)
				for j := i + 1; j < len(params.Directives); j++ {
					out.Items = append(out.Items, protocol.VoiceTranscriptDirectiveApplyItem{Status: "skipped", Message: "not attempted"})
				}
				return out, nil
			}
			for _, action := range d.EditDirective.Actions {
				if action.Kind != "replace_between_anchors" && action.Kind != "replace_range" {
					item.Status = "failed"
					item.Message = "host apply: unsupported edit action kind"
					out.Items = append(out.Items, item)
					for j := i + 1; j < len(params.Directives); j++ {
						out.Items = append(out.Items, protocol.VoiceTranscriptDirectiveApplyItem{Status: "skipped", Message: "not attempted"})
					}
					return out, nil
				}

				b, err := os.ReadFile(action.Path)
				if err != nil {
					item.Status = "failed"
					item.Message = "host apply: read target file failed: " + err.Error()
					out.Items = append(out.Items, item)
					for j := i + 1; j < len(params.Directives); j++ {
						out.Items = append(out.Items, protocol.VoiceTranscriptDirectiveApplyItem{Status: "skipped", Message: "not attempted"})
					}
					return out, nil
				}
				text := string(b)

				next := text
				switch action.Kind {
				case "replace_between_anchors":
					if action.Anchor == nil {
						item.Status = "failed"
						item.Message = "host apply: missing anchor"
						out.Items = append(out.Items, item)
						for j := i + 1; j < len(params.Directives); j++ {
							out.Items = append(out.Items, protocol.VoiceTranscriptDirectiveApplyItem{Status: "skipped", Message: "not attempted"})
						}
						return out, nil
					}
					before := action.Anchor.Before
					after := action.Anchor.After
					beforeIdx := strings.Index(text, before)
					if beforeIdx < 0 {
						item.Status = "failed"
						item.Message = "host apply: missing before anchor"
						out.Items = append(out.Items, item)
						for j := i + 1; j < len(params.Directives); j++ {
							out.Items = append(out.Items, protocol.VoiceTranscriptDirectiveApplyItem{Status: "skipped", Message: "not attempted"})
						}
						return out, nil
					}
					searchStart := beforeIdx + len(before)
					afterIdx := strings.Index(text[searchStart:], after)
					if afterIdx < 0 {
						item.Status = "failed"
						item.Message = "host apply: missing after anchor"
						out.Items = append(out.Items, item)
						for j := i + 1; j < len(params.Directives); j++ {
							out.Items = append(out.Items, protocol.VoiceTranscriptDirectiveApplyItem{Status: "skipped", Message: "not attempted"})
						}
						return out, nil
					}
					afterAbs := searchStart + afterIdx
					next = text[:searchStart] + action.NewText + text[afterAbs:]
				case "replace_range":
					if action.Range == nil {
						item.Status = "failed"
						item.Message = "host apply: missing range"
						out.Items = append(out.Items, item)
						for j := i + 1; j < len(params.Directives); j++ {
							out.Items = append(out.Items, protocol.VoiceTranscriptDirectiveApplyItem{Status: "skipped", Message: "not attempted"})
						}
						return out, nil
					}
					lines := strings.Split(text, "\n")
					sl := int(action.Range.StartLine)
					el := int(action.Range.EndLine)
					if sl < 0 || el < sl || sl >= len(lines) {
						item.Status = "failed"
						item.Message = "host apply: invalid range"
						out.Items = append(out.Items, item)
						for j := i + 1; j < len(params.Directives); j++ {
							out.Items = append(out.Items, protocol.VoiceTranscriptDirectiveApplyItem{Status: "skipped", Message: "not attempted"})
						}
						return out, nil
					}
					if el >= len(lines) {
						el = len(lines) - 1
					}
					replLines := strings.Split(action.NewText, "\n")
					merged := make([]string, 0, len(lines)-(el-sl+1)+len(replLines))
					merged = append(merged, lines[:sl]...)
					merged = append(merged, replLines...)
					merged = append(merged, lines[el+1:]...)
					next = strings.Join(merged, "\n")
				}

				if err := os.WriteFile(action.Path, []byte(next), 0o644); err != nil {
					item.Status = "failed"
					item.Message = "host apply: write target file failed: " + err.Error()
					out.Items = append(out.Items, item)
					for j := i + 1; j < len(params.Directives); j++ {
						out.Items = append(out.Items, protocol.VoiceTranscriptDirectiveApplyItem{Status: "skipped", Message: "not attempted"})
					}
					return out, nil
				}
			}

		default:
			item.Status = "failed"
			item.Message = "host apply: unsupported directive kind"
			out.Items = append(out.Items, item)
			for j := i + 1; j < len(params.Directives); j++ {
				out.Items = append(out.Items, protocol.VoiceTranscriptDirectiveApplyItem{Status: "skipped", Message: "not attempted"})
			}
			return out, nil
		}

		out.Items = append(out.Items, item)
	}

	return out, nil
}

type flakyApplyHost struct {
	t     *testing.T
	calls int
}

func (h *flakyApplyHost) ApplyDirectives(params protocol.HostApplyParams) (protocol.HostApplyResult, error) {
	h.t.Helper()
	h.calls++
	if h.calls == 1 {
		return protocol.HostApplyResult{
			Items: []protocol.VoiceTranscriptDirectiveApplyItem{
				{Status: "failed", Message: "stale_range: expectedSha256 mismatch"},
			},
		}, nil
	}
	return protocol.HostApplyResult{
		Items: []protocol.VoiceTranscriptDirectiveApplyItem{
			{Status: "ok"},
		},
	}, nil
}

func TestVoiceTranscript_SingleShotScopedEditBubbleSort(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	active := filepath.Join(dir, "bubble_sort.ts")
	src := `export function bubbleSort(arr: number[]): number[] {
  const n = arr.length;
  for (let i = 0; i < n; i++) {
    for (let j = 0; j < n - i - 1; j++) {
      // ANCHOR:bubble_sort_if_before
      if (arr[j] < arr[j+1]) {
        const tmp = arr[j];
        arr[j] = arr[j+1];
        arr[j+1] = tmp;
      }
      // ANCHOR:bubble_sort_if_after
    }
  }
  return arr;
}
`
	if err := os.WriteFile(active, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	a := agent.New(stub.New())
	svc := transcript.NewService(a, nil)
	svc.SetHostApplyClient(&applyHost{t: t})

	params := protocol.VoiceTranscriptParams{
		Text:          "Fix the bug in the bubble sort function",
		ActiveFile:    active,
		WorkspaceRoot: dir,
		DaemonConfig:  nil,
		// A non-empty session id exercises the normal session store path.
		ContextSessionId: "e2e-bubble-sort",
		// Cursor optional for this stub path.
		CursorPosition: &struct {
			Line      int64 `json:"line"`
			Character int64 `json:"character"`
		}{Line: 0, Character: 0},
	}

	res, ok, reason := svc.AcceptTranscript(params)
	if !ok || !res.Success {
		t.Fatalf("expected success, got ok=%v success=%v reason=%q summary=%q", ok, res.Success, reason, res.Summary)
	}

	b, err := os.ReadFile(active)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.Contains(got, "if (arr[j] > arr[j+1])") {
		t.Fatalf("expected comparator to be fixed; got:\n%s", got)
	}
	if strings.Contains(got, "if (arr[j] < arr[j+1])") {
		t.Fatalf("expected buggy comparator to be removed; got:\n%s", got)
	}
}

func TestVoiceTranscript_StaleRangeFailsSingleShot(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	active := filepath.Join(dir, "x.ts")
	if err := os.WriteFile(active, []byte("export function f(){\n  return 1;\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := agent.New(stub.New())
	svc := transcript.NewService(a, nil)
	host := &flakyApplyHost{t: t}
	svc.SetHostApplyClient(host)

	params := protocol.VoiceTranscriptParams{
		Text:             "Fix f",
		ActiveFile:       active,
		WorkspaceRoot:    dir,
		ContextSessionId: "stale-range-single-shot",
		CursorPosition: &struct {
			Line      int64 `json:"line"`
			Character int64 `json:"character"`
		}{Line: 0, Character: 0},
	}

	res, ok, reason := svc.AcceptTranscript(params)
	if res.Success {
		t.Fatalf("expected transcript failure after stale_range, got success=true ok=%v reason=%q summary=%q", ok, reason, res.Summary)
	}
	if !strings.Contains(reason, "stale_range") && !strings.Contains(reason, "host apply failed") {
		t.Fatalf("expected host apply failure reason, got %q", reason)
	}
	if host.calls != 1 {
		t.Fatalf("expected exactly 1 host apply call (no retries), got %d", host.calls)
	}
}
