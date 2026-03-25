import assert from "node:assert/strict";
import test from "node:test";

import { isEditApplyResult } from "./validators";

test("isEditApplyResult accepts explicit success shape", () => {
  const value = {
    kind: "success",
    actions: [
      {
        kind: "replace_between_anchors",
        path: "/tmp/file.ts",
        anchor: { before: "a", after: "b" },
        newText: "x",
      },
    ],
  };

  assert.equal(isEditApplyResult(value), true);
});

test("isEditApplyResult rejects mixed success/failure shape", () => {
  const value = {
    kind: "success",
    actions: [],
    failure: {
      code: "validation_failed",
      message: "bad",
    },
  };

  assert.equal(isEditApplyResult(value), false);
});

test("isEditApplyResult accepts explicit noop shape", () => {
  const value = {
    kind: "noop",
    reason: "No change required.",
  };

  assert.equal(isEditApplyResult(value), true);
});

test("isEditApplyResult rejects extra keys on success", () => {
  const value = {
    kind: "success",
    actions: [],
    reason: "unexpected",
  };

  assert.equal(isEditApplyResult(value), false);
});

test("isEditApplyResult rejects invalid failure code", () => {
  const value = {
    kind: "failure",
    failure: {
      code: "random_error_code",
      message: "bad",
    },
  };

  assert.equal(isEditApplyResult(value), false);
});

