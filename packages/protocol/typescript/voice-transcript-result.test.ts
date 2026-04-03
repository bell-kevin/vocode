import assert from "node:assert/strict";
import test from "node:test";

import { isVoiceTranscriptCompletion } from "./validators";

test("isVoiceTranscriptCompletion accepts success=true shape", () => {
  assert.equal(isVoiceTranscriptCompletion({ success: true }), true);
});

test("isVoiceTranscriptCompletion accepts summary when success", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      summary: "Renamed the handler and fixed imports.",
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion accepts uiDisposition when success", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      summary: "Not a coding request.",
      uiDisposition: "skipped",
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion accepts search with results + activeIndex", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      uiDisposition: "hidden",
      search: {
        results: [
          { path: "c:\\\\x.ts", line: 0, character: 1, preview: "hit" },
        ],
        activeIndex: 0,
      },
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion accepts search closed (control / cancel)", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      uiDisposition: "hidden",
      search: { closed: true },
      summary: "Search session closed",
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion accepts uiDisposition browse (search side panel)", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      uiDisposition: "browse",
      summary: "found 1 matches",
      search: {
        results: [{ path: "a.ts", line: 0, character: 0, preview: "x" }],
        activeIndex: 0,
      },
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion accepts clarify offer + summary", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      uiDisposition: "hidden",
      summary: "Clarification cancelled",
      clarify: { targetResolution: "instruction" },
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion accepts question group with answerText", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      question: {
        answerText: "About 10,957 or 10,958 depending on leap years.",
      },
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion accepts workspace needsFolder", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      summary: "Open a folder first.",
      uiDisposition: "shown",
      workspace: { needsFolder: true },
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion accepts clarify targetResolution", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      uiDisposition: "hidden",
      summary: "Which symbol?",
      clarify: { targetResolution: "instruction" },
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion rejects grouped fields when not success", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: false,
      search: { closed: true },
    }),
    false,
  );
});

test("isVoiceTranscriptCompletion allows summary when not success (error message)", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: false,
      summary: "oops",
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion rejects uiDisposition when not success", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: false,
      summary: "err",
      uiDisposition: "hidden",
    }),
    false,
  );
});

test("isVoiceTranscriptCompletion allows success=false minimal shape", () => {
  assert.equal(isVoiceTranscriptCompletion({ success: false }), true);
});

test("isVoiceTranscriptCompletion rejects extra keys", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      extra: 123,
    }),
    false,
  );
});

test("isVoiceTranscriptCompletion rejects extra keys (unexpected property)", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      unexpected: "bad",
    }),
    false,
  );
});

test("isVoiceTranscriptCompletion accepts fileSelection with results + activeIndex", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      uiDisposition: "hidden",
      fileSelection: {
        results: [{ path: "C:\\\\repo\\\\src\\\\main.ts", preview: "main.ts" }],
        activeIndex: 0,
      },
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion rejects fileSelection results without activeIndex", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      uiDisposition: "hidden",
      fileSelection: {
        results: [{ path: "C:\\\\x.ts" }],
      },
    }),
    false,
  );
});

test("isVoiceTranscriptCompletion accepts fileSelection empty object (enter session)", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      uiDisposition: "hidden",
      fileSelection: {},
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion rejects question without non-empty answerText", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      question: { answerText: "   " },
    }),
    false,
  );
});
