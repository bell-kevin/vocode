package hostdirectives

import (
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func hitNavigateDirectivesRange(path string, startLine, startChar, endLine, endChar int) []protocol.VoiceTranscriptDirective {
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
						StartLine: int64(startLine),
						StartChar: int64(startChar),
						EndLine:   int64(endLine),
						EndChar:   int64(endChar),
					},
				},
			},
		},
	}
	return []protocol.VoiceTranscriptDirective{open, sel}
}
