package hostdirectives

import (
	"testing"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func TestHitNavigateDirectivesExpand_prefersInnerSymbol(t *testing.T) {
	path := "/tmp/x.go"
	p := protocol.VoiceTranscriptParams{
		ActiveFile: path,
		ActiveFileSymbols: []struct {
			Name           string `json:"name"`
			Kind           string `json:"kind"`
			Range          struct {
				StartLine int64 `json:"startLine"`
				StartChar int64 `json:"startChar"`
				EndLine   int64 `json:"endLine"`
				EndChar   int64 `json:"endChar"`
			} `json:"range"`
			SelectionRange struct {
				StartLine int64 `json:"startLine"`
				StartChar int64 `json:"startChar"`
				EndLine   int64 `json:"endLine"`
				EndChar   int64 `json:"endChar"`
			} `json:"selectionRange"`
		}{
			{
				Name: "outer",
				Range: struct {
					StartLine int64 `json:"startLine"`
					StartChar int64 `json:"startChar"`
					EndLine   int64 `json:"endLine"`
					EndChar   int64 `json:"endChar"`
				}{StartLine: 0, StartChar: 0, EndLine: 20, EndChar: 0},
			},
			{
				Name: "inner",
				Range: struct {
					StartLine int64 `json:"startLine"`
					StartChar int64 `json:"startChar"`
					EndLine   int64 `json:"endLine"`
					EndChar   int64 `json:"endChar"`
				}{StartLine: 5, StartChar: 0, EndLine: 7, EndChar: 10},
			},
		},
	}
	d := HitNavigateDirectivesExpand(p, path, 6, 2, 3)
	if len(d) != 2 {
		t.Fatalf("directives: %d", len(d))
	}
	sel := d[1].NavigationDirective.Action.SelectRange
	if sel == nil {
		t.Fatal("no select_range")
	}
	if sel.Target.StartLine != 5 || sel.Target.EndLine != 7 {
		t.Fatalf("range %d..%d want 5..7", sel.Target.StartLine, sel.Target.EndLine)
	}
}

func TestSmallestSymbolContainingPointSyms(t *testing.T) {
	syms := []DocumentSymbol{
		{Name: "outer", Range: DocumentRange{StartLine: 0, StartChar: 0, EndLine: 10, EndChar: 1}},
		{Name: "inner", Range: DocumentRange{StartLine: 2, StartChar: 0, EndLine: 4, EndChar: 10}},
	}
	sl, sc, el, ec, ok := smallestSymbolContainingPointSyms(syms, 3, 2)
	if !ok {
		t.Fatal("expected hit")
	}
	if sl != 2 || el != 4 {
		t.Fatalf("got range %d,%d..%d,%d want inner", sl, sc, el, ec)
	}
}

func TestHitNavigateDirectivesExpand_fallsBackWhenNoSymbols(t *testing.T) {
	p := protocol.VoiceTranscriptParams{
		ActiveFile: "/x/a.go",
	}
	d := HitNavigateDirectivesExpand(p, "/x/a.go", 1, 2, 5)
	if len(d) != 2 {
		t.Fatalf("want 2 directives got %d", len(d))
	}
}
