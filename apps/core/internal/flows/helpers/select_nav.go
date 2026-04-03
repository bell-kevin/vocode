package helpers

import (
	"regexp"
	"strings"
)

var (
	navNextRe = regexp.MustCompile(`\b(next|forward)\b`)
	navBackRe = regexp.MustCompile(`\b(back|prev|previous)\b`)
	navGotoRe = regexp.MustCompile(`\b(goto|jump|show)\b`)
	navIntRe  = regexp.MustCompile(`\b\d+\b`)
)

// ParseNav parses list-navigation intent from a voice transcript.
// kind is one of: "next", "back", "pick", "exit". ord is 1-based for "pick".
func ParseNav(transcript string) (kind string, ord int, ok bool) {
	t := strings.TrimSpace(transcript)
	if t == "" {
		return "", 0, false
	}
	if IsExitPhrase(t) {
		return "exit", 0, true
	}
	lower := strings.ToLower(t)
	if navNextRe.MatchString(lower) {
		return "next", 0, true
	}
	if navBackRe.MatchString(lower) {
		return "back", 0, true
	}
	if n := parsePickOrdinal(t); n > 0 {
		return "pick", n, true
	}
	if navGotoRe.MatchString(lower) {
		if n := parseAnyIntToken(t); n > 0 {
			return "pick", n, true
		}
	}
	return "", 0, false
}

func parsePickOrdinal(s string) int {
	if n := parseAnyIntToken(s); n > 0 {
		if navIntRe.MatchString(strings.TrimSpace(s)) && len(strings.Fields(strings.TrimSpace(s))) == 1 {
			return n
		}
	}
	t := strings.TrimSpace(strings.ToLower(s))
	for _, w := range strings.Fields(t) {
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
