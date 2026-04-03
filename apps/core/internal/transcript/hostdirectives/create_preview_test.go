package hostdirectives

import "testing"

func TestWorkspaceSearchStylePreviewFromSymbol_function(t *testing.T) {
	t.Parallel()
	got := WorkspaceSearchStylePreviewFromSymbol(DocumentSymbol{
		Name: "pong",
		Kind: "Function",
	})
	if got != "function pong()" {
		t.Fatalf("got %q", got)
	}
}

func TestCreateFlowHitPreview_insideSymbol(t *testing.T) {
	t.Parallel()
	syms := []DocumentSymbol{
		{
			Name: "pong",
			Kind: "Function",
			Range: DocumentRange{
				StartLine: 45, StartChar: 0,
				EndLine: 47, EndChar: 1,
			},
		},
	}
	fallback := "function pong() {"
	got := CreateFlowHitPreview(syms, 45, 0, fallback)
	if got != "function pong()" {
		t.Fatalf("got %q", got)
	}
	got = CreateFlowHitPreview(syms, 46, 4, fallback)
	if got != "function pong()" {
		t.Fatalf("inside body: got %q", got)
	}
}

func TestCreateFlowHitPreview_fallback(t *testing.T) {
	t.Parallel()
	fallback := "plain text"
	if got := CreateFlowHitPreview(nil, 0, 0, fallback); got != fallback {
		t.Fatalf("got %q", got)
	}
	if got := CreateFlowHitPreview([]DocumentSymbol{}, 0, 0, fallback); got != fallback {
		t.Fatalf("got %q", got)
	}
}
