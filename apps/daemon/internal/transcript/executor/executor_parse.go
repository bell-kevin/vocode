package executor

import (
	"path/filepath"
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type parsedCodeAction struct {
	path string
	rng  *struct {
		StartLine int64 `json:"startLine"`
		StartChar int64 `json:"startChar"`
		EndLine   int64 `json:"endLine"`
		EndChar   int64 `json:"endChar"`
	}
	kind    string
	pref    string
	summary string
}

func parseCodeAction(text string, params protocol.VoiceTranscriptParams) (parsedCodeAction, bool) {
	l := strings.ToLower(strings.TrimSpace(text))
	active := filepath.Clean(strings.TrimSpace(params.ActiveFile))
	if active == "" {
		return parsedCodeAction{}, false
	}
	sel := params.ActiveSelection
	var rng *struct {
		StartLine int64 `json:"startLine"`
		StartChar int64 `json:"startChar"`
		EndLine   int64 `json:"endLine"`
		EndChar   int64 `json:"endChar"`
	}
	if sel != nil {
		rng = &struct {
			StartLine int64 `json:"startLine"`
			StartChar int64 `json:"startChar"`
			EndLine   int64 `json:"endLine"`
			EndChar   int64 `json:"endChar"`
		}{StartLine: sel.StartLine, StartChar: sel.StartChar, EndLine: sel.EndLine, EndChar: sel.EndChar}
	}

	switch {
	case strings.Contains(l, "extract function"):
		if rng == nil || (rng.StartLine == rng.EndLine && rng.StartChar == rng.EndChar) {
			return parsedCodeAction{summary: "extract function requires a selection"}, false
		}
		return parsedCodeAction{path: active, rng: rng, kind: "refactor.extract.function", summary: "extract function"}, true
	case strings.Contains(l, "extract variable"):
		if rng == nil || (rng.StartLine == rng.EndLine && rng.StartChar == rng.EndChar) {
			return parsedCodeAction{summary: "extract variable requires a selection"}, false
		}
		return parsedCodeAction{path: active, rng: rng, kind: "refactor.extract.variable", summary: "extract variable"}, true
	case strings.Contains(l, "extract constant"):
		if rng == nil || (rng.StartLine == rng.EndLine && rng.StartChar == rng.EndChar) {
			return parsedCodeAction{summary: "extract constant requires a selection"}, false
		}
		return parsedCodeAction{path: active, rng: rng, kind: "refactor.extract.constant", summary: "extract constant"}, true
	case strings.Contains(l, "inline"):
		return parsedCodeAction{path: active, rng: rng, kind: "refactor.inline", summary: "inline"}, true
	case strings.Contains(l, "organize imports"):
		return parsedCodeAction{path: active, kind: "source.organizeImports", summary: "organize imports"}, true
	case strings.Contains(l, "fix all"):
		return parsedCodeAction{path: active, kind: "source.fixAll", summary: "fix all"}, true
	case strings.Contains(l, "quick fix") || strings.Contains(l, "quickfix"):
		return parsedCodeAction{path: active, rng: rng, kind: "quickfix", summary: "quick fix"}, true
	default:
		return parsedCodeAction{}, false
	}
}

type parsedFormat struct {
	path  string
	scope string
	rng   *struct {
		StartLine int64 `json:"startLine"`
		StartChar int64 `json:"startChar"`
		EndLine   int64 `json:"endLine"`
		EndChar   int64 `json:"endChar"`
	}
	summary string
}

func parseFormat(text string, params protocol.VoiceTranscriptParams) (parsedFormat, bool) {
	l := strings.ToLower(strings.TrimSpace(text))
	if !strings.Contains(l, "format") {
		return parsedFormat{}, false
	}
	active := filepath.Clean(strings.TrimSpace(params.ActiveFile))
	if active == "" {
		return parsedFormat{}, false
	}
	if strings.Contains(l, "selection") || strings.Contains(l, "selected") {
		sel := params.ActiveSelection
		if sel == nil {
			return parsedFormat{}, false
		}
		return parsedFormat{
			path:  active,
			scope: "selection",
			rng: &struct {
				StartLine int64 `json:"startLine"`
				StartChar int64 `json:"startChar"`
				EndLine   int64 `json:"endLine"`
				EndChar   int64 `json:"endChar"`
			}{StartLine: sel.StartLine, StartChar: sel.StartChar, EndLine: sel.EndLine, EndChar: sel.EndChar},
			summary: "format selection",
		}, true
	}
	return parsedFormat{path: active, scope: "document", summary: "format document"}, true
}

func parseRenameNewName(text string) (string, bool) {
	t := strings.ToLower(text)
	if !strings.Contains(t, "rename") {
		return "", false
	}
	idx := strings.LastIndex(t, " to ")
	if idx < 0 {
		return "", false
	}
	newName := strings.TrimSpace(text[idx+4:])
	newName = strings.Trim(newName, "\"'`")
	if newName == "" {
		return "", false
	}
	if strings.ContainsAny(newName, " \t\r\n") {
		return "", false
	}
	return newName, true
}
