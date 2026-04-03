package run

import (
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// applyFileSelectionToVoiceSession updates daemon session from completion.fileSelection (mirrors core outcome.Apply).
func applyFileSelectionToVoiceSession(vs *agentcontext.VoiceSession, fs *protocol.VoiceTranscriptFileSearchState) {
	if vs == nil || fs == nil {
		return
	}
	if fs.Closed || fs.NoHits {
		vs.FileSelectionPaths = nil
		vs.FileSelectionIndex = 0
		vs.FileSelectionFocus = ""
		popFileSelectionFlow(vs)
		return
	}
	if len(fs.Results) > 0 {
		paths := make([]string, 0, len(fs.Results))
		for _, h := range fs.Results {
			paths = append(paths, strings.TrimSpace(h.Path))
		}
		vs.FileSelectionPaths = paths
		i := 0
		if fs.ActiveIndex != nil {
			i = int(*fs.ActiveIndex)
			if i < 0 || i >= len(paths) {
				i = 0
			}
		}
		vs.FileSelectionIndex = i
		vs.FileSelectionFocus = paths[i]
		syncFileSelectionStackForPaths(vs)
		return
	}
	// Wire {} — enter file-selection mode with no hit list yet.
	vs.FileSelectionPaths = nil
	vs.FileSelectionIndex = 0
	vs.FileSelectionFocus = ""
}

func popFileSelectionFlow(vs *agentcontext.VoiceSession) {
	for agentcontext.FlowTopKind(vs.FlowStack) == agentcontext.FlowKindFileSelection {
		ns, ok := agentcontext.FlowPopIfTop(vs.FlowStack, agentcontext.FlowKindFileSelection)
		if !ok {
			break
		}
		vs.FlowStack = ns
	}
}

func stackHasFileSelectionFrame(stack []agentcontext.FlowFrame) bool {
	for i := range stack {
		if stack[i].Kind == agentcontext.FlowKindFileSelection {
			return true
		}
	}
	return false
}

func syncFileSelectionStackForPaths(vs *agentcontext.VoiceSession) {
	if vs == nil || len(vs.FileSelectionPaths) == 0 {
		return
	}
	if stackHasFileSelectionFrame(vs.FlowStack) {
		return
	}
	if agentcontext.FlowTopKind(vs.FlowStack) == agentcontext.FlowKindMain {
		if ns, ok := agentcontext.FlowPush(vs.FlowStack, agentcontext.FlowFrame{Kind: agentcontext.FlowKindFileSelection}); ok {
			vs.FlowStack = ns
		}
	}
}
