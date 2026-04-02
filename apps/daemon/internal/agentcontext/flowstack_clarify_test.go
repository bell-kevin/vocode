package agentcontext

import "testing"

func TestClarifyPromptFromStack(t *testing.T) {
	t.Parallel()
	st := []FlowFrame{{Kind: FlowKindMain}, {
		Kind:                      FlowKindClarify,
		ClarifyTargetResolution:   "instruction",
		ClarifyQuestion:           " Which file? ",
		ClarifyOriginalTranscript: " fix it ",
	}}
	q, orig, tgt, ok := ClarifyPromptFromStack(st)
	if !ok || q != "Which file?" || orig != "fix it" || tgt != "instruction" {
		t.Fatalf("got q=%q orig=%q tgt=%q ok=%v", q, orig, tgt, ok)
	}
	if _, _, _, ok2 := ClarifyPromptFromStack(st[:1]); ok2 {
		t.Fatalf("expected no clarify on main-only stack")
	}
}
