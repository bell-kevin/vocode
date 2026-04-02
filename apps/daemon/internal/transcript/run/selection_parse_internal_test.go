package run

import "testing"

// parseSelectionNav treats exit as any utterance containing a whole-word exit cue
// (cancel, done, close, stop, etc.), not a single fixed phrase.
func TestParseSelectionNav_exitDetectsKeywordInUtterance(t *testing.T) {
	t.Parallel()
	exitPhrases := []string{
		"cancel",
		"cancel that",
		"please cancel",
		"I'm done",
		"done with this",
		"close",
		"close the results",
		"stop",
		"stop searching",
		"quit",
		"leave",
		"end",
		"abort",
		"exit search",
	}
	for _, text := range exitPhrases {
		t.Run(text, func(t *testing.T) {
			t.Parallel()
			kind, _, ok := parseSelectionNav(text)
			if !ok || kind != "exit" {
				t.Fatalf("parseSelectionNav(%q) = kind=%q ok=%v; want exit, true", text, kind, ok)
			}
		})
	}
}

func TestParseSelectionNav_nonExitUtterances(t *testing.T) {
	t.Parallel()
	cases := []struct {
		text string
		kind string // expected kind when ok; "" means ok should be false
	}{
		{"", ""},
		{"next", "next"},
		{"go forward", "next"},
		{"back", "back"},
		{"previous", "back"},
		{"result 2", "pick"},
		{"find main", ""},
		{"rename foo to bar", ""},
	}
	for _, tc := range cases {
		t.Run(tc.text, func(t *testing.T) {
			t.Parallel()
			kind, _, ok := parseSelectionNav(tc.text)
			if tc.kind == "" {
				if ok {
					t.Fatalf("parseSelectionNav(%q) = kind=%q ok=true; want ok=false", tc.text, kind)
				}
				return
			}
			if !ok || kind != tc.kind {
				t.Fatalf("parseSelectionNav(%q) = kind=%q ok=%v; want kind=%q ok=true", tc.text, kind, ok, tc.kind)
			}
		})
	}
}
