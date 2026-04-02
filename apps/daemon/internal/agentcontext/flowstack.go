package agentcontext

import "strings"

// Flow kinds for voice transcript dispatch (top of stack handles each utterance).
const (
	FlowKindMain          = "main"
	FlowKindSelection     = "selection"
	FlowKindFileSelection = "file_selection"
	FlowKindClarify       = "clarify"
)

const maxFlowStackDepth = 4

// FlowFrame is one level on the voice flow stack (see product plan: Main / Selection / FileSelection / Clarify).
type FlowFrame struct {
	Kind                      string
	ClarifyTargetResolution   string
	ClarifyQuestion           string
	ClarifyOriginalTranscript string
}

// ClarifyPromptFromStack returns question, original utterance, and target key when the top frame is clarify.
func ClarifyPromptFromStack(stack []FlowFrame) (question, originalTranscript, target string, ok bool) {
	if FlowTopKind(stack) != FlowKindClarify {
		return "", "", "", false
	}
	top := stack[len(stack)-1]
	return strings.TrimSpace(top.ClarifyQuestion),
		strings.TrimSpace(top.ClarifyOriginalTranscript),
		strings.TrimSpace(top.ClarifyTargetResolution),
		true
}

// FlowTopKind returns the active flow; empty stack means main.
func FlowTopKind(stack []FlowFrame) string {
	if len(stack) == 0 {
		return FlowKindMain
	}
	return stack[len(stack)-1].Kind
}

// FlowPush appends a frame if under max depth and returns true.
func FlowPush(stack []FlowFrame, frame FlowFrame) ([]FlowFrame, bool) {
	if len(stack) >= maxFlowStackDepth {
		return stack, false
	}
	return append(stack, frame), true
}

// FlowPop removes the top frame and returns it, ok false if empty.
func FlowPop(stack []FlowFrame) ([]FlowFrame, FlowFrame, bool) {
	if len(stack) == 0 {
		return stack, FlowFrame{}, false
	}
	top := stack[len(stack)-1]
	return stack[:len(stack)-1], top, true
}

// FlowPopIfTop pops one frame when the top kind matches want.
func FlowPopIfTop(stack []FlowFrame, want string) ([]FlowFrame, bool) {
	if FlowTopKind(stack) != want {
		return stack, false
	}
	ns, _, ok := FlowPop(stack)
	if !ok {
		return stack, false
	}
	return ns, true
}

// PopWhileTopKind pops while the top frame kind is one of kinds (order: repeated pop until top ∉ kinds).
func PopWhileTopKind(stack []FlowFrame, kinds ...string) []FlowFrame {
	want := make(map[string]bool, len(kinds))
	for _, k := range kinds {
		want[k] = true
	}
	for {
		top := FlowTopKind(stack)
		if !want[top] {
			return stack
		}
		ns, _, ok := FlowPop(stack)
		if !ok {
			return stack
		}
		stack = ns
	}
}
