import assert from "node:assert/strict";
import test from "node:test";

import { isVoiceTranscriptResult } from "./validators";

test("isVoiceTranscriptResult accepts success=true shape", () => {
  assert.equal(isVoiceTranscriptResult({ success: true }), true);
});

test("isVoiceTranscriptResult accepts summary when success", () => {
  assert.equal(
    isVoiceTranscriptResult({
      success: true,
      summary: "Renamed the handler and fixed imports.",
    }),
    true,
  );
});

test("isVoiceTranscriptResult rejects summary when not success", () => {
  assert.equal(
    isVoiceTranscriptResult({
      success: false,
      summary: "oops",
    }),
    false,
  );
});

test("isVoiceTranscriptResult allows success=false minimal shape", () => {
  assert.equal(isVoiceTranscriptResult({ success: false }), true);
});

test("isVoiceTranscriptResult rejects extra keys", () => {
  assert.equal(
    isVoiceTranscriptResult({
      success: true,
      extra: 123,
    }),
    false,
  );
});

test("isVoiceTranscriptResult accepts directives with edit directive success", () => {
  assert.equal(
    isVoiceTranscriptResult({
      success: true,
      directives: [
        {
          kind: "edit",
          editDirective: {
            kind: "success",
            actions: [
              {
                kind: "replace_between_anchors",
                path: "/tmp/x.ts",
                anchor: { before: "a", after: "b" },
                newText: "\n",
              },
            ],
          },
        },
      ],
      applyBatchId: "abc123",
    }),
    true,
  );
});

test("isVoiceTranscriptResult rejects directives without applyBatchId", () => {
  assert.equal(
    isVoiceTranscriptResult({
      success: true,
      directives: [
        {
          kind: "command",
          commandDirective: { command: "echo", args: ["stub"] },
        },
      ],
    }),
    false,
  );
});

test("isVoiceTranscriptResult rejects extra keys (unexpected property)", () => {
  assert.equal(
    isVoiceTranscriptResult({
      success: true,
      unexpected: "bad",
      directives: [
        {
          kind: "command",
          commandDirective: { command: "echo", args: ["stub"] },
        },
      ],
      applyBatchId: "x",
    }),
    false,
  );
});

test("isVoiceTranscriptResult accepts undo directive", () => {
  assert.equal(
    isVoiceTranscriptResult({
      success: true,
      directives: [
        {
          kind: "undo",
          undoDirective: { scope: "last_transcript" },
        },
      ],
      applyBatchId: "batch-1",
    }),
    true,
  );
});
