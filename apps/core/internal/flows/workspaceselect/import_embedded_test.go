package workspaceselectflow

import (
	"strings"
	"testing"
)

func TestMergePeelLeadingImportLines_peelsWhenNotImportTarget(t *testing.T) {
	t.Parallel()
	rep := "import { useState } from 'react';\n\nexport default function App() {\n  return null;\n}"
	target := "export default function App() {\n  return null;\n}"
	rest, peeled := mergePeelLeadingImportLines(rep, target)
	if len(peeled) != 1 || peeled[0] != "import { useState } from 'react';" {
		t.Fatalf("peeled: %#v", peeled)
	}
	if !strings.Contains(rest, "export default function App()") {
		t.Fatalf("rest missing body: %q", rest)
	}
}

func TestMergePeelLeadingImportLines_skipsWhenTargetIsImportSection(t *testing.T) {
	t.Parallel()
	rep := "import { x } from \"a\";\nimport { y } from \"b\";\n"
	target := "import { x } from \"a\";\n"
	rest, peeled := mergePeelLeadingImportLines(rep, target)
	if len(peeled) != 0 || rest != rep {
		t.Fatalf("want no peel, got peeled=%#v rest=%q", peeled, rest)
	}
}

func TestTargetTextLooksLikeImportOnlySection(t *testing.T) {
	t.Parallel()
	if !targetTextLooksLikeImportOnlySection("import a from 'a'\n") {
		t.Fatal("expected import-only")
	}
	if targetTextLooksLikeImportOnlySection("import a from 'a'\nconst x = 1\n") {
		t.Fatal("expected not import-only")
	}
}
