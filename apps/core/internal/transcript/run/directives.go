package run

import (
	"fmt"
	"sync/atomic"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

var applyBatchSeq uint64

func newApplyBatchID() string {
	seq := atomic.AddUint64(&applyBatchSeq, 1)
	return fmt.Sprintf("core-%d", seq)
}

func hitNavigateDirectives(path string, line0, char0, length int) []protocol.VoiceTranscriptDirective {
	open := protocol.VoiceTranscriptDirective{
		Kind: "navigate",
		NavigationDirective: &protocol.NavigationDirective{
			Kind: "success",
			Action: &protocol.NavigationAction{
				Kind: "open_file",
				OpenFile: &struct {
					Path string `json:"path"`
				}{Path: path},
			},
		},
	}
	sel := protocol.VoiceTranscriptDirective{
		Kind: "navigate",
		NavigationDirective: &protocol.NavigationDirective{
			Kind: "success",
			Action: &protocol.NavigationAction{
				Kind: "select_range",
				SelectRange: &struct {
					Target struct {
						Path      string `json:"path,omitempty"`
						StartLine int64  `json:"startLine"`
						StartChar int64  `json:"startChar"`
						EndLine   int64  `json:"endLine"`
						EndChar   int64  `json:"endChar"`
					} `json:"target"`
				}{
					Target: struct {
						Path      string `json:"path,omitempty"`
						StartLine int64  `json:"startLine"`
						StartChar int64  `json:"startChar"`
						EndLine   int64  `json:"endLine"`
						EndChar   int64  `json:"endChar"`
					}{
						Path:      path,
						StartLine: int64(line0),
						StartChar: int64(char0),
						EndLine:   int64(line0),
						EndChar:   int64(char0 + length),
					},
				},
			},
		},
	}
	return []protocol.VoiceTranscriptDirective{open, sel}
}
