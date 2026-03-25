import assert from "node:assert/strict";
import test from "node:test";
import { isVoiceTranscriptResult } from "@vocode/protocol";

test("isVoiceTranscriptResult accepts accepted=true shape", () => {
  assert.equal(isVoiceTranscriptResult({ accepted: true }), true);
});

test("isVoiceTranscriptResult rejects accepted=false shape", () => {
  assert.equal(isVoiceTranscriptResult({ accepted: false }), false);
});

test("isVoiceTranscriptResult rejects extra keys", () => {
  assert.equal(isVoiceTranscriptResult({ accepted: true, extra: 123 }), false);
});
