package helpers

import (
	"regexp"
	"strings"
)

var exitPhraseRe = regexp.MustCompile(`\b(cancel|exit|close|stop|done|quit|leave|end|abort)\b`)

// IsExitPhrase is true when the utterance clearly requests leaving the flow.
func IsExitPhrase(transcript string) bool {
	t := strings.TrimSpace(strings.ToLower(transcript))
	return exitPhraseRe.MatchString(t)
}
