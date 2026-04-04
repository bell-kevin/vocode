package fileselectflow

import "testing"

func TestParseCreateEntryNameHeuristic(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Make a file called what dot js", "what.js"},
		{"make a new file called what dot js", "what.js"},
		{"create file called foo dot ts", "foo.ts"},
		{"named bar dot go please", "bar.go"},
	}
	for _, tc := range cases {
		if g := parseCreateEntryNameHeuristic(tc.in); g != tc.want {
			t.Fatalf("%q: got %q want %q", tc.in, g, tc.want)
		}
	}
}

func TestSanitizeNewFileName_rejectsDotDot(t *testing.T) {
	if g := sanitizeNewFileName(`evil/../x`); g != "" {
		t.Fatalf("got %q", g)
	}
}

func TestSanitizeNewFileName_trimsSttTrailingDotOnExtension(t *testing.T) {
	if g := sanitizeNewFileName(`index.tsx.`); g != "index.tsx" {
		t.Fatalf("got %q want index.tsx", g)
	}
}
