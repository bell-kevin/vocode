import assert from "node:assert/strict";
import test from "node:test";
import type { ReplaceBetweenAnchorsAction } from "@vocode/protocol";

import { resolveReplaceBetweenAnchors } from "./apply-edit-helpers";

test("resolveReplaceBetweenAnchors replaces the range between unique anchors", () => {
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

  const output = resolveReplaceBetweenAnchors(input, action);

  assert.match(output.nextText, /updated safely/);
  assert.doesNotMatch(output.nextText, /hi from vocode/);
});

test("resolveReplaceBetweenAnchors throws when after anchor is missing", () => {
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

  assert.throws(() => resolveReplaceBetweenAnchors(input, action), {
    message: /after anchor/,
  });
});

test("resolveReplaceBetweenAnchors throws when before anchor is ambiguous", () => {
  const action: ReplaceBetweenAnchorsAction = {
    kind: "replace_between_anchors",
    path: "/tmp/example.ts",
    anchor: {
      before: "TARGET_START",
      after: "TARGET_END",
    },
    newText: "updated",
  };

  const input = ["TARGET_START", "x", "TARGET_END", "TARGET_START"].join("\n");

  assert.throws(() => resolveReplaceBetweenAnchors(input, action), {
    message: /matched multiple locations/,
  });
});

test("resolveReplaceBetweenAnchors throws when after anchor is ambiguous", () => {
  const action: ReplaceBetweenAnchorsAction = {
    kind: "replace_between_anchors",
    path: "/tmp/example.ts",
    anchor: {
      before: "TARGET_START",
      after: "TARGET_END",
    },
    newText: "updated",
  };

  const input = ["TARGET_START", "x", "TARGET_END", "TARGET_END"].join("\n");

  assert.throws(() => resolveReplaceBetweenAnchors(input, action), {
    message: /matched multiple locations/,
  });
});

test("resolveReplaceBetweenAnchors returns exact offsets for editor range edits", () => {
  const action: ReplaceBetweenAnchorsAction = {
    kind: "replace_between_anchors",
    path: "/tmp/example.ts",
    anchor: {
      before: "const value = 1;",
      after: "return value;",
    },
    newText: "\nconst value = 2;\n",
  };

  const input = [
    "function run() {",
    "const value = 1;",
    "return value;",
    "}",
  ].join("\n");

  const replacement = resolveReplaceBetweenAnchors(input, action);

  assert.equal(
    input.slice(replacement.startOffset, replacement.endOffset),
    "\n",
  );
  assert.match(replacement.nextText, /const value = 2/);
  assert.equal(replacement.replacementText, action.newText);
});

test("resolveReplaceBetweenAnchors supports sequential actions against updated text", () => {
  const firstAction: ReplaceBetweenAnchorsAction = {
    kind: "replace_between_anchors",
    path: "/tmp/example.ts",
    anchor: {
      before: "function run() {",
      after: "const second = 2;",
    },
    newText: "\n  const first = 100;\n",
  };

  const secondAction: ReplaceBetweenAnchorsAction = {
    kind: "replace_between_anchors",
    path: "/tmp/example.ts",
    anchor: {
      before: "const second = 2;",
      after: "}",
    },
    newText: "\n  return first + second;\n",
  };

  const input = [
    "function run() {",
    "  const first = 1;",
    "const second = 2;",
    "  return first;",
    "}",
  ].join("\n");

  const first = resolveReplaceBetweenAnchors(input, firstAction);
  const second = resolveReplaceBetweenAnchors(first.nextText, secondAction);

  assert.match(second.nextText, /const first = 100/);
  assert.match(second.nextText, /return first \+ second/);
});
