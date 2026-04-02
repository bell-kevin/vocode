package service

import (
	"strings"

	global "vocoding.net/vocode/v2/apps/core/internal/flows/global"
	"vocoding.net/vocode/v2/apps/core/internal/flows/selection"
)

// selectListControlFromText maps the transcript to hit-list navigation (next/back/pick).
func selectListControlFromText(text string) (op string, pick1Based int, ok bool) {
	k, ord, ok := selection.ParseNav(text)
	if !ok {
		return "", 0, false
	}
	switch k {
	case "next":
		return "next", 0, true
	case "back":
		return "back", 0, true
	case "pick":
		return "pick", ord, true
	default:
		return "", 0, false
	}
}

// handleGlobalControlRoute implements the shared "control" route (exit today).
func handleGlobalControlRoute(text string) (exit bool) {
	return global.IsExitPhrase(text)
}

func heuristicSearchQuery(text string) string {
	t := strings.TrimSpace(strings.ToLower(text))
	q := t
	for _, p := range []string{"find ", "search ", "where is ", "locate "} {
		if strings.HasPrefix(q, p) {
			q = strings.TrimSpace(q[len(p):])
			break
		}
	}
	if q == "" {
		return strings.TrimSpace(text)
	}
	return q
}

func stubQuestionAnswer() string {
	return "Stub answer."
}
