import assert from "node:assert/strict";
import test from "node:test";
import type { ReplaceBetweenAnchorsAction } from "@vocode/protocol";

import { applyReplaceBetweenAnchors } from "./apply-edit-helpers";

test("applyReplaceBetweenAnchors replaces the range between unique anchors", () => {
  const action: ReplaceBetweenAnchorsAction = {
    kind: "replace_between_anchors",
    path: "/tmp/example.ts",
    anchor: {
      before: "function firstBraceAnchors() {",
      after: "}",
    },
    newText: '\n  console.log("updated safely");\n',
  };

  const input = [
    "function firstBraceAnchors() {",
    '  console.log("hi from vocode");',
    "}",
  ].join("\n");

  const output = applyReplaceBetweenAnchors(input, action);

  assert.match(output, /updated safely/);
  assert.doesNotMatch(output, /hi from vocode/);
});

test("applyReplaceBetweenAnchors throws when after anchor is missing", () => {
  const action: ReplaceBetweenAnchorsAction = {
    kind: "replace_between_anchors",
    path: "/tmp/example.ts",
    anchor: {
      before: "function firstBraceAnchors() {",
      after: "END_ANCHOR",
    },
    newText: '\n  console.log("updated safely");\n',
  };

  const input = ["function firstBraceAnchors() {", "  return 1;", "}"].join(
    "\n",
  );

  assert.throws(() => applyReplaceBetweenAnchors(input, action), {
    message: /after anchor/,
  });
});
