package searchapply

import (
	"path/filepath"
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// FileSearchStateFromPaths builds protocol fileSelection state for a numbered path list.
func FileSearchStateFromPaths(paths []string, activeIndex int) *protocol.VoiceTranscriptFileSearchState {
	return FileSearchStateFromPathsWithDir(paths, nil, activeIndex)
}

// FileSearchStateFromPathsWithDir sets VoiceTranscriptFileListHit.isDirectory when isDir[i] is true.
func FileSearchStateFromPathsWithDir(paths []string, isDir []bool, activeIndex int) *protocol.VoiceTranscriptFileSearchState {
	if len(paths) == 0 {
		return nil
	}
	if activeIndex < 0 || activeIndex >= len(paths) {
		activeIndex = 0
	}
	res := make([]protocol.VoiceTranscriptFileListHit, 0, len(paths))
	for i, p := range paths {
		p = strings.TrimSpace(p)
		h := protocol.VoiceTranscriptFileListHit{
			Path:    p,
			Preview: filepath.Base(p),
		}
		if i < len(isDir) && isDir[i] {
			h.IsDirectory = true
		}
		res = append(res, h)
	}
	idx := int64(activeIndex)
	return &protocol.VoiceTranscriptFileSearchState{
		Results:     res,
		ActiveIndex: &idx,
	}
}

// FileSearchStateFromSinglePath is one hit at index 0.
func FileSearchStateFromSinglePath(path string) *protocol.VoiceTranscriptFileSearchState {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	return FileSearchStateFromPaths([]string{path}, 0)
}
