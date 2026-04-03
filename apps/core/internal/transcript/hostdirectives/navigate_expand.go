package hostdirectives

import (
	"path/filepath"
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// HitNavigateDirectivesExpand opens the hit file and selects the smallest LSP document-symbol
// range that contains the rg anchor when the hit file matches params.ActiveFile and symbols
// are present. Otherwise falls back to HitNavigateDirectives (narrow match span).
func HitNavigateDirectivesExpand(
	params protocol.VoiceTranscriptParams,
	path string,
	line0, char0, length int,
) []protocol.VoiceTranscriptDirective {
	active := strings.TrimSpace(params.ActiveFile)
	if active == "" || !pathsEqualFold(path, active) || len(params.ActiveFileSymbols) == 0 {
		return HitNavigateDirectives(path, line0, char0, length)
	}
	if sl, sc, el, ec, ok := smallestSymbolContainingPoint(params.ActiveFileSymbols, line0, char0); ok {
		return hitNavigateDirectivesRange(path, sl, sc, el, ec)
	}
	return HitNavigateDirectives(path, line0, char0, length)
}

func pathsEqualFold(a, b string) bool {
	a = filepath.Clean(strings.TrimSpace(a))
	b = filepath.Clean(strings.TrimSpace(b))
	if a == "" || b == "" {
		return false
	}
	return strings.EqualFold(a, b)
}

func smallestSymbolContainingPoint(
	symbols []struct {
		Name           string `json:"name"`
		Kind           string `json:"kind"`
		Range          struct {
			StartLine int64 `json:"startLine"`
			StartChar int64 `json:"startChar"`
			EndLine   int64 `json:"endLine"`
			EndChar   int64 `json:"endChar"`
		} `json:"range"`
		SelectionRange struct {
			StartLine int64 `json:"startLine"`
			StartChar int64 `json:"startChar"`
			EndLine   int64 `json:"endLine"`
			EndChar   int64 `json:"endChar"`
		} `json:"selectionRange"`
	},
	line0, char0 int,
) (startLine, startChar, endLine, endChar int, ok bool) {
	line := int64(line0)
	char := int64(char0)
	bestIdx := -1
	bestSize := 0
	for i := range symbols {
		r := symbols[i].Range
		if line < r.StartLine || line > r.EndLine {
			continue
		}
		if line == r.StartLine && char < r.StartChar {
			continue
		}
		if line == r.EndLine && char > r.EndChar {
			continue
		}
		sz := int(r.EndLine-r.StartLine)*1_000_000 + int(r.EndChar-r.StartChar)
		if bestIdx == -1 || sz < bestSize {
			bestIdx = i
			bestSize = sz
		}
	}
	if bestIdx == -1 {
		return 0, 0, 0, 0, false
	}
	r := symbols[bestIdx].Range
	return int(r.StartLine), int(r.StartChar), int(r.EndLine), int(r.EndChar), true
}

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
