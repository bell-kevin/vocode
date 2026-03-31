package transcript_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agent/stub"
	"vocoding.net/vocode/v2/apps/daemon/internal/gather"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch/edit"
	"vocoding.net/vocode/v2/apps/daemon/internal/symbols"
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
				if action.Kind != "replace_between_anchors" || action.Anchor == nil {
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

				next := text[:searchStart] + action.NewText + text[afterAbs:]
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

func TestVoiceTranscript_DuplexApply_RepairsAndEditsBubbleSort(t *testing.T) {
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

	sym := symbols.NewTreeSitterResolver()
	a := agent.New(stub.New())
	h := dispatch.NewHandler(edit.NewEngine())
	g := gather.NewProvider(sym)
	svc := transcript.NewService(a, h, g, sym, nil)
	svc.SetHostApplyClient(&applyHost{t: t})

	params := protocol.VoiceTranscriptParams{
		Text:          "Fix the bug in the bubble sort function",
		ActiveFile:    active,
		WorkspaceRoot: dir,
		DaemonConfig: &struct {
			MaxPlannerTurns                *int64 `json:"maxPlannerTurns,omitempty"`
			MaxIntentsPerBatch             *int64 `json:"maxIntentsPerBatch,omitempty"`
			MaxIntentDispatchRetries       *int64 `json:"maxIntentDispatchRetries,omitempty"`
			MaxContextRounds               *int64 `json:"maxContextRounds,omitempty"`
			MaxContextBytes                *int64 `json:"maxContextBytes,omitempty"`
			MaxConsecutiveContextRequests  *int64 `json:"maxConsecutiveContextRequests,omitempty"`
			MaxTranscriptRepairRpcs        *int64 `json:"maxTranscriptRepairRpcs,omitempty"`
			SessionIdleResetMs             *int64 `json:"sessionIdleResetMs,omitempty"`
			MaxGatheredBytes               *int64 `json:"maxGatheredBytes,omitempty"`
			MaxGatheredExcerpts            *int64 `json:"maxGatheredExcerpts,omitempty"`
		}{
			MaxTranscriptRepairRpcs: ptrInt64(4),
		},
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

func ptrInt64(v int64) *int64 {
	return &v
}

