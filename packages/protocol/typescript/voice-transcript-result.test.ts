import assert from "node:assert/strict";
import test from "node:test";

import { isVoiceTranscriptResult } from "./validators";

test("isVoiceTranscriptResult accepts accepted=true shape", () => {
  assert.equal(isVoiceTranscriptResult({ accepted: true }), true);
});

test("isVoiceTranscriptResult rejects accepted=false shape", () => {
  assert.equal(isVoiceTranscriptResult({ accepted: false }), true);
});

test("isVoiceTranscriptResult rejects extra keys", () => {
  assert.equal(isVoiceTranscriptResult({ accepted: true, extra: 123 }), false);
});

test("isVoiceTranscriptResult accepts directives with edit directive success", () => {
  assert.equal(
    isVoiceTranscriptResult({
      accepted: true,
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
    }),
    true,
  );
});

test("isVoiceTranscriptResult rejects extra keys (unexpected property)", () => {
  assert.equal(
    isVoiceTranscriptResult({
      accepted: true,
      unexpected: "bad",
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

test("isVoiceTranscriptResult accepts undo directive", () => {
  assert.equal(
    isVoiceTranscriptResult({
      accepted: true,
      directives: [
        {
          kind: "undo",
          undoDirective: { scope: "last_transcript" },
        },
      ],
    }),
    true,
  );
});
