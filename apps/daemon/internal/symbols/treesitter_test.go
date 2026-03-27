package symbols

import "testing"

func TestParseTagLine(t *testing.T) {
	t.Parallel()

	ref, ok := parseTagLine("test\t/tmp/test.js\t/^function test() {$/;\"\tfunction\tline:12")
	if !ok {
		t.Fatal("expected parse success")
	}
	if ref.Name != "test" {
		t.Fatalf("unexpected name: %q", ref.Name)
	}
	if ref.Path != "/tmp/test.js" {
		t.Fatalf("unexpected path: %q", ref.Path)
	}
	if ref.Kind != "function" {
		t.Fatalf("unexpected kind: %q", ref.Kind)
	}
	if ref.Line != 12 {
		t.Fatalf("unexpected line: %d", ref.Line)
	}
}

func TestParseTagLine_KindOverrideField(t *testing.T) {
	t.Parallel()

	ref, ok := parseTagLine("Foo\t/tmp/x.py\t/^class Foo:$/;\"\tidentifier\tline:7\tkind:class")
	if !ok {
		t.Fatal("expected parse success")
	}
	if ref.Kind != "class" {
		t.Fatalf("unexpected kind: %q", ref.Kind)
	}
}

func TestNormalizeKind(t *testing.T) {
	t.Parallel()
	if got := normalizeKind("f"); got != "function" {
		t.Fatalf("expected function, got %q", got)
	}
	if got := normalizeKind("Class"); got != "class" {
		t.Fatalf("expected class, got %q", got)
	}
	if got := normalizeKind("kind:method"); got != "method" {
		t.Fatalf("expected method, got %q", got)
	}
	if got := normalizeKind("trait"); got != "interface" {
		t.Fatalf("expected interface, got %q", got)
	}
}
