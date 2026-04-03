package workspaceselectflow

import (
	"strings"
	"testing"
)

func TestStubMatchesWorkspaceCreate(t *testing.T) {
	t.Parallel()
	if !StubMatchesWorkspaceCreate("insert foo on line 12", "insert foo on line 12") {
		t.Fatal("expected on line + insert")
	}
	if !StubMatchesWorkspaceCreate("add a new function at the beginning of the file", "raw") {
		t.Fatal("expected beginning")
	}
	if !StubMatchesWorkspaceCreate("put exports at the end of the file", "raw") {
		t.Fatal("expected end of file")
	}
	if StubMatchesWorkspaceCreate("make it compile on line 9", "make it compile on line 9") {
		t.Fatal("did not expect create without add/insert/new/put")
	}
}

func TestNumberedFileSnippet_shortFile(t *testing.T) {
	t.Parallel()
	s := numberedFileSnippet([]string{"a", "b", "c"})
	if s != "1|a\n2|b\n3|c\n" {
		t.Fatalf("got %q", s)
	}
}

func TestRangeAfterLine(t *testing.T) {
	t.Parallel()
	lines := []string{"L1", "L2", "L3"}
	sl, sc, el, ec := rangeAfterLine(lines, 1)
	if sl != 1 || sc != 0 || el != 1 || ec != 0 {
		t.Fatalf("after line 1: got %d,%d-%d,%d", sl, sc, el, ec)
	}
	sl, sc, el, ec = rangeAfterLine(lines, 2)
	if sl != 2 || sc != 0 || el != 2 || ec != 0 {
		t.Fatalf("after line 2: got %d,%d-%d,%d", sl, sc, el, ec)
	}
	sl, sc, el, ec = rangeAfterLine(lines, 3)
	if sl != 2 || sc != 2 || el != 2 || ec != 2 { // len("L3") == 2
		t.Fatalf("after last line: got %d,%d-%d,%d", sl, sc, el, ec)
	}
}

func TestInsertAffixForZeroWidth(t *testing.T) {
	t.Parallel()
	defaultCore := "let x;"
	cases := []struct {
		name    string
		lines   []string
		sl, sc  int
		wantPfx string
		wantSuf string
		core    string
	}{
		{name: "bof_before_nonempty", lines: []string{"a"}, sl: 0, sc: 0, wantPfx: "", wantSuf: "\n"},
		{name: "bof_before_empty_row", lines: []string{"", "y"}, sl: 0, sc: 0, wantPfx: "", wantSuf: ""},
		{name: "before_nonempty_line", lines: []string{"a", "b"}, sl: 1, sc: 0, wantPfx: "", wantSuf: "\n"},
		{name: "before_empty_line", lines: []string{"x", "", "y"}, sl: 1, sc: 0, wantPfx: "", wantSuf: ""},
		{name: "core_has_trailing_nl", lines: []string{"a", "b"}, sl: 1, sc: 0, wantPfx: "", wantSuf: "", core: "let x;\n"},
		{name: "eof_after_nonempty", lines: []string{"// c"}, sl: 0, sc: len("// c"), wantPfx: "\n", wantSuf: "\n"},
		{name: "eof_core_leading_nl", lines: []string{"// c"}, sl: 0, sc: len("// c"), wantPfx: "", wantSuf: "\n", core: "\n" + defaultCore},
		{name: "eol_not_eof", lines: []string{"a", "b", "c"}, sl: 1, sc: len("b"), wantPfx: "\n", wantSuf: ""},
		{name: "eol_is_eof", lines: []string{"a", "b"}, sl: 1, sc: len("b"), wantPfx: "\n", wantSuf: "\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := tc.core
			if c == "" {
				c = defaultCore
			}
			pfx, suf := insertAffixForZeroWidth(tc.lines, tc.sl, tc.sc, c)
			if pfx != tc.wantPfx || suf != tc.wantSuf {
				t.Fatalf("got pfx=%q suf=%q want pfx=%q suf=%q", pfx, suf, tc.wantPfx, tc.wantSuf)
			}
		})
	}
}

func TestCreateReplaceCoreByteOffset(t *testing.T) {
	t.Parallel()
	pre := "a\n\nb"
	lines := strings.Split(pre, "\n")
	core := "thing"
	pfx, suf := insertAffixForZeroWidth(lines, 1, 0, core)
	full := pfx + core + suf
	post := pre[:byteOffsetAtLineChar(pre, 1, 0)] + full + pre[byteOffsetAtLineChar(pre, 1, 0):]
	off, ok := createReplaceCoreByteOffset(pre, post, 1, 0, pfx, full, core)
	if !ok {
		t.Fatal("expected ok")
	}
	line, ch := lineCharUTF16FromByteIndex(normalizeEOL(post), off)
	if line != 1 || ch != 0 {
		t.Fatalf("core start want line=1 char=0, got %d,%d", line, ch)
	}
}

func TestAlignCoreAnchorByteOffset(t *testing.T) {
	t.Parallel()
	post := "a\n\nfunction x() {}\n"
	off := strings.Index(post, "function")
	if off < 0 {
		t.Fatal("setup")
	}
	// Simulate a stale hint pointing at the blank line before "function"
	hint := strings.Index(post, "\n\n") + 1
	got := alignCoreAnchorByteOffset(post, hint, "function x() {}")
	if got != off {
		t.Fatalf("want offset %d, got %d", off, got)
	}
}

func TestAppendCreateToAppend_spacing(t *testing.T) {
	t.Parallel()
	got, core, pl := appendCreateToAppend("foo", "bar")
	if core != "bar" || pl != 1 || got != "\nbar\n" {
		t.Fatalf("mid-line body: got %q core=%q pl=%d", got, core, pl)
	}
	got, core, pl = appendCreateToAppend("foo\n", "bar")
	if core != "bar" || pl != 0 || got != "bar\n" {
		t.Fatalf("single trailing nl: got %q", got)
	}
	got, core, pl = appendCreateToAppend("foo\n\n", "bar")
	if core != "bar" || pl != 0 || got != "bar\n" {
		t.Fatalf("already ends with nl: got %q", got)
	}
	got, core, pl = appendCreateToAppend("foo\n\n", "\n\nbar")
	if core != "\n\nbar" || pl != 0 || got != "\n\nbar\n" {
		t.Fatalf("model supplies leading nls: got %q core=%q", got, core)
	}
	got, core, pl = appendCreateToAppend("foo", "\n\nbar")
	if core != "\n\nbar" || pl != 0 || got != "\n\nbar\n" {
		t.Fatalf("mid-line + model leading nls: got %q core=%q", got, core)
	}
}

func TestAppendInsertionByteOffset(t *testing.T) {
	t.Parallel()
	body := "hello\n\n"
	to, core, pl := appendCreateToAppend(body, "block")
	full := body + to
	off, ok := appendInsertionByteOffset(body, full, pl, core)
	if !ok || off != len(body)+pl {
		t.Fatalf("offset ok=%v off=%d want %d", ok, off, len(body)+pl)
	}
}

func TestLineCharUTF16FromByteIndex_emoji(t *testing.T) {
	t.Parallel()
	s := "😀x"
	off := len("😀")
	line, char := lineCharUTF16FromByteIndex(s, off)
	if line != 0 || char != 2 {
		t.Fatalf("want line=0 char=2 (UTF-16), got %d,%d", line, char)
	}
}
