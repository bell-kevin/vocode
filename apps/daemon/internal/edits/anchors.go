package edits

import (
	"fmt"
	"regexp"
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type lineBlock struct {
	beforeLine  string
	afterAnchor string
	between     string
	indent      string
}

var functionStartPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\s*func\b`),
	regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\b`),
	regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s+\w+\s*=\s*(?:async\s*)?\([^)]*\)\s*=>\s*\{`),
	regexp.MustCompile(`^\s*(?:public\s+|private\s+|protected\s+|static\s+|async\s+)*\w+\([^)]*\)\s*\{`),
}

func findUniqueAnchoredRange(fileText string, before string, after string) (int, int, *protocol.EditFailure) {
	_, beforeEnd, beforeFailure := findUniqueOccurrence(fileText, before, 0, "before")
	if beforeFailure != nil {
		return 0, 0, beforeFailure
	}

	afterStart, _, afterFailure := findUniqueOccurrence(fileText, after, beforeEnd, "after")
	if afterFailure != nil {
		return 0, 0, afterFailure
	}

	if afterStart < beforeEnd {
		return 0, 0, editFailure("validation_failed", "Anchor order was invalid.")
	}

	return beforeEnd, afterStart, nil
}

func findUniqueOccurrence(fileText string, needle string, start int, label string) (int, int, *protocol.EditFailure) {
	if needle == "" {
		return 0, 0, editFailure("missing_anchor", fmt.Sprintf("The %s anchor was empty.", label))
	}

	index := strings.Index(fileText[start:], needle)
	if index == -1 {
		return 0, 0, editFailure("missing_anchor", fmt.Sprintf("Could not find %s anchor %q.", label, needle))
	}

	absoluteStart := start + index
	nextIndex := strings.Index(fileText[absoluteStart+1:], needle)
	if nextIndex != -1 {
		return 0, 0, editFailure("ambiguous_target", fmt.Sprintf("The %s anchor %q matched multiple locations.", label, needle))
	}

	return absoluteStart, absoluteStart + len(needle), nil
}

func findSingleFunctionBlock(fileText string) (*lineBlock, *protocol.EditFailure) {
	lines := strings.Split(fileText, "\n")
	candidates := make([]lineBlock, 0, 1)

	for lineIndex, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !isFunctionStart(trimmed) || !strings.Contains(line, "{") {
			continue
		}

		closeIndex, ok := findMatchingBraceLine(lines, lineIndex)
		if !ok || closeIndex <= lineIndex {
			continue
		}

		between := "\n"
		if closeIndex > lineIndex+1 {
			between += strings.Join(lines[lineIndex+1:closeIndex], "\n")
			between += "\n"
		}

		candidates = append(candidates, lineBlock{
			beforeLine:  line,
			afterAnchor: strings.Join(lines[closeIndex:], "\n"),
			between:     between,
			indent:      indentForBlock(line),
		})
	}

	switch len(candidates) {
	case 0:
		return nil, editFailure("missing_anchor", "Could not find a supported current function in the active file.")
	case 1:
		return &candidates[0], nil
	default:
		return nil, editFailure("ambiguous_target", "The active file contains multiple candidate functions; current function was ambiguous.")
	}
}

func isFunctionStart(trimmedLine string) bool {
	for _, pattern := range functionStartPatterns {
		if pattern.MatchString(trimmedLine) {
			return true
		}
	}
	return false
}

func findMatchingBraceLine(lines []string, startLine int) (int, bool) {
	depth := 0
	started := false

	for lineIndex := startLine; lineIndex < len(lines); lineIndex++ {
		for _, r := range lines[lineIndex] {
			switch r {
			case '{':
				depth++
				started = true
			case '}':
				if started {
					depth--
					if depth == 0 {
						return lineIndex, true
					}
				}
			}
		}
	}

	return 0, false
}

func indentForBlock(signatureLine string) string {
	indentWidth := len(signatureLine) - len(strings.TrimLeft(signatureLine, " \t"))
	baseIndent := signatureLine[:indentWidth]
	if strings.HasPrefix(baseIndent, "\t") {
		return baseIndent + "\t"
	}
	return baseIndent + "  "
}
