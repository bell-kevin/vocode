import assert from "node:assert/strict";
import test from "node:test";

import { isVoiceTranscriptResult } from "./validators";

test("isVoiceTranscriptResult accepts accepted=true shape", () => {
  assert.equal(isVoiceTranscriptResult({ accepted: true }), true);
});

test("isVoiceTranscriptResult rejects accepted=false shape", () => {
  assert.equal(isVoiceTranscriptResult({ accepted: false }), false);
});

test("isVoiceTranscriptResult rejects extra keys", () => {
  assert.equal(isVoiceTranscriptResult({ accepted: true, extra: 123 }), false);
});

test("isVoiceTranscriptResult accepts steps with edit success", () => {
  assert.equal(
    isVoiceTranscriptResult({
      accepted: true,
      steps: [
        {
          kind: "edit",
          editResult: {
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
    }),
    true,
  );
});

test("isVoiceTranscriptResult rejects planError together with steps", () => {
  assert.equal(
    isVoiceTranscriptResult({
      accepted: true,
      planError: "bad",
      steps: [
        {
          kind: "run_command",
          commandParams: { command: "echo", args: ["stub"] },
        },
      ],
    }),
    false,
  );
});
