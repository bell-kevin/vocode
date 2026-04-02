package selectfileflow

import (
	"regexp"
	"strings"
)

var (
	selectionNavNextRe = regexp.MustCompile(`\b(next|forward)\b`)
	selectionNavBackRe = regexp.MustCompile(`\b(back|prev|previous)\b`)
	selectionNavGotoRe = regexp.MustCompile(`\b(goto|jump|show)\b`)
	navIntRe           = regexp.MustCompile(`\b\d+\b`)
)

// Heuristic-based determination of control intent.
func parseControl(transcript string) (kind string, ok bool) {
	t := strings.TrimSpace(strings.ToLower(transcript))
	if selectionNavNextRe.MatchString(t) {
		return "next", true
	}
	if selectionNavBackRe.MatchString(t) {
		return "back", true
	}
	if selectionNavGotoRe.MatchString(t) {
		return "goto", true
	}
	return "", false
}

// HandleControl determines a control intent and handles it.
func HandleControl(transcript string) {
	intent, ok := parseControl(transcript)
	if !ok {
		return
	}

	switch intent {
	case "next":
		// Handle next
	case "back":
		// Handle back
	case "goto":
		// Handle goto by number (extract ordinal / cardinal from text)
	default:
		return
	}
}

// Helpers
func parseAnyOrdinal(s string) int {
	if n := parseAnyIntToken(s); n > 0 {
		return n
	}
	for _, w := range strings.Fields(s) {
		switch strings.Trim(w, ".,;:!?") {
		case "one", "1st", "first":
			return 1
		case "two", "2nd", "second":
			return 2
		case "three", "3rd", "third":
			return 3
		case "four", "4th", "fourth":
			return 4
		case "five", "5th", "fifth":
			return 5
		case "six", "6th", "sixth":
			return 6
		case "seven", "7th", "seventh":
			return 7
		case "eight", "8th", "eighth":
			return 8
		case "nine", "9th", "ninth":
			return 9
		case "ten", "10th", "tenth":
			return 10
		}
	}
	return 0
}

func isBareOrdinalUtterance(t string) bool {
	t = strings.TrimSpace(strings.ToLower(t))
	if t == "" {
		return false
	}
	if navIntRe.MatchString(t) && len(strings.Fields(t)) == 1 {
		return true
	}
	switch strings.Trim(t, ".,;:!?") {
	case "one", "first", "1st",
		"two", "second", "2nd",
		"three", "third", "3rd",
		"four", "fourth", "4th",
		"five", "fifth", "5th",
		"six", "sixth", "6th",
		"seven", "seventh", "7th",
		"eight", "eighth", "8th",
		"nine", "ninth", "9th",
		"ten", "tenth", "10th":
		return true
	}
	return false
}

func parseAnyIntToken(s string) int {
	n := 0
	inDigits := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			inDigits = true
			n = n*10 + int(c-'0')
			continue
		}
		if inDigits {
			return n
		}
	}
	if inDigits {
		return n
	}
	return 0
}
