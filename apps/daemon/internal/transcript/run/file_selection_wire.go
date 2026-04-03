package run

import (
	"path/filepath"
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func fileSearchStateFromPaths(paths []string, activeIndex int) *protocol.VoiceTranscriptFileSearchState {
	if len(paths) == 0 {
		return nil
	}
	if activeIndex < 0 || activeIndex >= len(paths) {
		activeIndex = 0
	}
	res := make([]protocol.VoiceTranscriptFileListHit, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		res = append(res, protocol.VoiceTranscriptFileListHit{
			Path:    p,
			Preview: filepath.Base(p),
		})
	}
	idx := int64(activeIndex)
	return &protocol.VoiceTranscriptFileSearchState{
		Results:     res,
		ActiveIndex: &idx,
	}
}

func fileSearchStateFromSinglePath(path string) *protocol.VoiceTranscriptFileSearchState {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	return fileSearchStateFromPaths([]string{path}, 0)
}
