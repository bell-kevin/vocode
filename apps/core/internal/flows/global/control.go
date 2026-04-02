package globalflow

import (
	"regexp"
	"strings"
)

var exitRe = regexp.MustCompile(`\b(cancel|exit|close|stop|done|quit|leave|end|abort)\b`)

// Heuristic-based determination of control intent.
func parseControl(transcript string) (kind string, ok bool) {
	t := strings.TrimSpace(strings.ToLower(transcript))
	if exitRe.MatchString(t) {
		return "exit", true
	}
	return "", false
}

// IsExitPhrase is true when the utterance clearly requests leaving the flow.
func IsExitPhrase(transcript string) bool {
	t := strings.TrimSpace(strings.ToLower(transcript))
	return exitRe.MatchString(t)
}

// HandleControl determines a control intent and handles it.
func HandleControl(transcript string) {
	intent, ok := parseControl(transcript)
	if !ok {
		return
	}

	switch intent {
	case "exit":
		// Handle flow exit
	default:
		return
	}
}
