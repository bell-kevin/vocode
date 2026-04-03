package workspaceselectflow

import "testing"

func TestParseSymbolRenameNewName(t *testing.T) {
	t.Parallel()
	if n, ok := parseSymbolRenameNewName(`rename foo to bar`); !ok || n != "bar" {
		t.Fatalf("got %q %v", n, ok)
	}
	if _, ok := parseSymbolRenameNewName("rename only"); ok {
		t.Fatal("expected false without ' to '")
	}
}
