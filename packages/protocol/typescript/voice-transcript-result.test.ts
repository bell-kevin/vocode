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

test("isVoiceTranscriptCompletion accepts transcriptOutcome when success", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      summary: "Not a coding request.",
      transcriptOutcome: "irrelevant",
      uiDisposition: "skipped",
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion accepts selection_control outcome when success", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      transcriptOutcome: "selection_control",
      uiDisposition: "hidden",
      searchResults: [
        { path: "c:\\\\x.ts", line: 0, character: 1, preview: "hit" },
      ],
      activeSearchIndex: 0,
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion accepts clarify_control outcome when success", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      transcriptOutcome: "clarify_control",
      uiDisposition: "hidden",
      summary: "Clarification cancelled",
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion accepts answer outcome with answerText", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      transcriptOutcome: "answer",
      answerText: "About 10,957 or 10,958 depending on leap years.",
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion accepts selection outcome with searchResults", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      transcriptOutcome: "selection",
      searchResults: [
        {
          path: "c:\\\\x.ts",
          line: 0,
          character: 1,
          preview: "function test() {}",
        },
      ],
      activeSearchIndex: 0,
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion accepts needs_workspace_folder outcome", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      transcriptOutcome: "needs_workspace_folder",
      summary: "Open a folder first.",
      uiDisposition: "shown",
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion accepts clarifyTargetResolution", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      transcriptOutcome: "clarify",
      summary: "Which symbol?",
      clarifyTargetResolution: "instruction",
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion accepts search outcome with searchResults", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      transcriptOutcome: "search",
      searchResults: [
        {
          path: "c:\\\\x.ts",
          line: 0,
          character: 1,
          preview: "function test() {}",
        },
      ],
      activeSearchIndex: 0,
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion rejects transcriptOutcome when not success", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: false,
      transcriptOutcome: "irrelevant",
    }),
    false,
  );
});

test("isVoiceTranscriptCompletion rejects summary when not success", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: false,
      summary: "oops",
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

test("isVoiceTranscriptCompletion accepts file_selection_control with fileSelectionFocusPath", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      transcriptOutcome: "file_selection_control",
      uiDisposition: "hidden",
      fileSelectionFocusPath: "C:\\\\repo\\\\src\\\\main.ts",
    }),
    true,
  );
});

test("isVoiceTranscriptCompletion rejects file_selection_control without focus path", () => {
  assert.equal(
    isVoiceTranscriptCompletion({
      success: true,
      transcriptOutcome: "file_selection_control",
      uiDisposition: "hidden",
    }),
    false,
  );
});
