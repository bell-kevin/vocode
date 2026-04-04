package searchapply

import (
	"testing"

	"vocoding.net/vocode/v2/apps/core/internal/search"
)

func TestPrioritizeActiveFileSearchHits_ordersActiveFileFirstByLine(t *testing.T) {
	active := "/proj/foo.ts"
	hits := []search.Hit{
		{Path: "/proj/other.ts", Line0: 1, Char0: 0},
		{Path: "/proj/foo.ts", Line0: 10, Char0: 0},
		{Path: "/proj/foo.ts", Line0: 2, Char0: 0},
		{Path: "/proj/b.ts", Line0: 0, Char0: 0},
	}
	got := prioritizeActiveFileSearchHits(hits, active)
	if got[0].Path != "/proj/foo.ts" || got[0].Line0 != 2 {
		t.Fatalf("first want foo.ts line 2, got %+v", got[0])
	}
	if got[1].Path != "/proj/foo.ts" || got[1].Line0 != 10 {
		t.Fatalf("second want foo.ts line 10, got %+v", got[1])
	}
}

func TestSearchHitPathSame_fold(t *testing.T) {
	if !searchHitPathSame(`/PROJ/Foo.ts`, `/proj/foo.ts`) {
		t.Fatal("expected fold match")
	}
	if searchHitPathSame(`/proj/a.ts`, `/proj/b.ts`) {
		t.Fatal("expected no match")
	}
}

func TestPrioritizeActiveFileSearchHits_emptyActiveUnchanged(t *testing.T) {
	hits := []search.Hit{{Path: "b"}, {Path: "a"}}
	got := prioritizeActiveFileSearchHits(hits, "  ")
	if len(got) != 2 || got[0].Path != "b" || got[1].Path != "a" {
		t.Fatalf("%+v", got)
	}
}
