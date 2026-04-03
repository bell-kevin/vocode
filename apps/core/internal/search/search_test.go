package search

import "testing"

func TestSearchLikeQueryFromText_extractsPrefixes(t *testing.T) {
	cases := []struct {
		in     string
		want   string
		wantOk bool
	}{
		{in: "find foo", want: "foo", wantOk: true},
		{in: "search for bar", want: "bar", wantOk: true},
		{in: " where is baz ", want: "baz", wantOk: true},
		{in: "locate", want: "", wantOk: false},
		{in: "random text", want: "", wantOk: false},
	}

	for _, tc := range cases {
		got, ok := SearchLikeQueryFromText(tc.in)
		if ok != tc.wantOk {
			t.Fatalf("SearchLikeQueryFromText(%q): ok=%v, want %v", tc.in, ok, tc.wantOk)
		}
		if tc.wantOk && got != tc.want {
			t.Fatalf("SearchLikeQueryFromText(%q): got %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFileSearchLikeQueryFromText_extractsPrefixes(t *testing.T) {
	cases := []struct {
		in     string
		want   string
		wantOk bool
	}{
		{in: "find file foo.go", want: "foo.go", wantOk: true},
		{in: "OPEN FILE bar.ts", want: "bar.ts", wantOk: true},
		{in: "file named baz", want: "baz", wantOk: true},
		{in: "find file ", want: "", wantOk: false},
		{in: "find foo", want: "", wantOk: false},
	}

	for _, tc := range cases {
		got, ok := FileSearchLikeQueryFromText(tc.in)
		if ok != tc.wantOk {
			t.Fatalf("FileSearchLikeQueryFromText(%q): ok=%v, want %v", tc.in, ok, tc.wantOk)
		}
		if tc.wantOk && got != tc.want {
			t.Fatalf("FileSearchLikeQueryFromText(%q): got %q, want %q", tc.in, got, tc.want)
		}
	}
}
