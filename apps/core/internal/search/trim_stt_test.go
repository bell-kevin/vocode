package search

import "testing"

func TestTrimSttTrailingSentenceDot(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"index.tsx.", "index.tsx"},
		{"index.tsx..", "index.tsx"},
		{"foo.bar.go.", "foo.bar.go"},
		{"  index.tsx.  ", "index.tsx"},
		{"README.", "README."},
		{"x.", "x."},
		{"", ""},
		{"noext", "noext"},
		{".env.", ".env"},
	}
	for _, tt := range tests {
		if got := TrimSttTrailingSentenceDot(tt.in); got != tt.want {
			t.Errorf("TrimSttTrailingSentenceDot(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
