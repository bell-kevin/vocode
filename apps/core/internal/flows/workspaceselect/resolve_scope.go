package workspaceselectflow

import (
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/transcript/hostdirectives"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// resolveEditRange picks an LSP-style range for a scoped edit from params and file text.
func resolveEditRange(params protocol.VoiceTranscriptParams, fileText string) (startLine, startChar, endLine, endChar int, ok bool) {
	lines := strings.Split(fileText, "\n")
	if len(lines) == 0 {
		return 0, 0, 0, 0, false
	}
	last := len(lines) - 1

	if sel := params.ActiveSelection; sel != nil {
		sl := int(sel.StartLine)
		sc := int(sel.StartChar)
		el := int(sel.EndLine)
		ec := int(sel.EndChar)
		if sl == el && sc == ec {
			// caret-only selection: fall through to symbol / file heuristics
		} else {
			if normalizeRange(lines, &sl, &sc, &el, &ec) {
				return sl, sc, el, ec, true
			}
		}
	}

	if cp := params.CursorPosition; cp != nil && len(params.ActiveFileSymbols) > 0 {
		line := int(cp.Line)
		char := int(cp.Character)
		syms := hostdirectives.DocumentSymbolsFromParams(params)
		if sl, sc, el, ec, hit := hostdirectives.SmallestSymbolContainingRange(syms, line, char); hit {
			return sl, sc, el, ec, true
		}
	}

	// Whole file
	endLine = last
	endChar = len(lines[last])
	return 0, 0, endLine, endChar, true
}

func normalizeRange(lines []string, sl, sc, el, ec *int) bool {
	if *sl < 0 || *el < 0 || *sl >= len(lines) || *el >= len(lines) {
		return false
	}
	if *sl > *el {
		*sl, *el = *el, *sl
		*sc, *ec = *ec, *sc
	}
	if *sc < 0 {
		*sc = 0
	}
	if *ec < 0 {
		*ec = 0
	}
	if *sc > len(lines[*sl]) {
		*sc = len(lines[*sl])
	}
	if *ec > len(lines[*el]) {
		*ec = len(lines[*el])
	}
	return true
}

func extractRangeText(fileText string, sl, sc, el, ec int) (string, bool) {
	lines := strings.Split(fileText, "\n")
	if sl < 0 || el < sl || el >= len(lines) {
		return "", false
	}
	if sc < 0 || ec < 0 {
		return "", false
	}
	if sc > len(lines[sl]) || ec > len(lines[el]) {
		return "", false
	}
	if sl == el {
		return lines[sl][sc:ec], true
	}
	var b strings.Builder
	b.WriteString(lines[sl][sc:])
	for i := sl + 1; i < el; i++ {
		b.WriteByte('\n')
		b.WriteString(lines[i])
	}
	b.WriteByte('\n')
	b.WriteString(lines[el][:ec])
	return b.String(), true
}
