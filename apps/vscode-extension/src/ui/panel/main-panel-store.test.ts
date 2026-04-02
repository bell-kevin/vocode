import assert from "node:assert/strict";
import test from "node:test";

import { MainPanelStore } from "./main-panel-store";

test("clears partial buffer when a committed line arrives", () => {
  const store = new MainPanelStore();
  store.setVoiceListening(true);

  store.onPartial("partial one");
  store.onPartial("partial two");

  assert.equal(store.getSnapshot().latestPartial, "partial two");

  store.enqueueCommitted("final line");

  const snap = store.getSnapshot();
  assert.equal(snap.latestPartial, null);
  assert.equal(snap.pending.length, 1);
  assert.equal(snap.pending[0]?.text, "final line");
});

test("does not enqueue empty committed text", () => {
  const store = new MainPanelStore();

  assert.equal(store.enqueueCommitted("   "), null);
  assert.equal(store.getSnapshot().pending.length, 0);
});

test("tracks pending through processing to handled", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("run tests");

  assert.ok(id !== null);
  assert.equal(store.getSnapshot().pending[0]?.status, "queued");

  store.markProcessing(id);
  assert.equal(store.getSnapshot().pending[0]?.status, "processing");

  store.markHandled(id);
  assert.equal(store.getSnapshot().pending.length, 0);
  assert.equal(store.getSnapshot().recentHandled[0]?.text, "run tests");
});

test("voice transcript RPC queue exposes active pending id", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("line") as number;
  store.markProcessing(id);
  store.beginVoiceTranscriptRpc(id);
  assert.equal(store.activeVoiceTranscriptRpcPendingId(), id);
  store.endVoiceTranscriptRpc(id);
  assert.equal(store.activeVoiceTranscriptRpcPendingId(), undefined);
});

test("markHandled stores optional agent summary", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("fix the bug") as number;
  store.markHandled(id, { summary: "  Updated handler and tests.  " });
  const h = store.getSnapshot().recentHandled[0];
  assert.equal(h?.summary, "Updated handler and tests.");
});

test("markHandled sets skipped when transcript outcome is irrelevant", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("what is the weather") as number;
  store.markHandled(id, {
    summary: "Not a coding task.",
    transcriptOutcome: "irrelevant",
    uiDisposition: "skipped",
  });
  const h = store.getSnapshot().recentHandled[0];
  assert.equal(h?.skipped, true);
});

test("abortClarifyAsSkipped clears prompt and appends skipped row", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("fix thing") as number;
  store.markHandled(id, {
    summary: "Which file?",
    transcriptOutcome: "clarify",
  });
  assert.ok(store.getSnapshot().clarifyPrompt);
  store.abortClarifyAsSkipped();
  assert.equal(store.getSnapshot().clarifyPrompt, undefined);
  const h = store.getSnapshot().recentHandled[0];
  assert.equal(h?.text, "fix thing");
  assert.equal(h?.skipped, true);
  assert.ok(
    typeof h?.summary === "string" &&
      h.summary.includes("Clarification cancelled"),
  );
});

test("dismissSearchState clears search hit list", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("find foo") as number;
  store.markHandled(id, {
    transcriptOutcome: "search",
    uiDisposition: "hidden",
    searchResults: [
      {
        path: "a.ts",
        line: 0,
        character: 0,
        preview: "hit",
      },
    ],
    activeSearchIndex: 0,
  });
  assert.ok(store.getSnapshot().searchState);
  store.dismissSearchState();
  assert.equal(store.getSnapshot().searchState, undefined);
});

test("markHandled selection_control without searchResults clears search (voice cancel)", () => {
  const store = new MainPanelStore();
  const id1 = store.enqueueCommitted("find foo") as number;
  store.markHandled(id1, {
    transcriptOutcome: "search",
    uiDisposition: "hidden",
    searchResults: [{ path: "a.ts", line: 0, character: 0, preview: "hit" }],
    activeSearchIndex: 0,
  });
  assert.ok(store.getSnapshot().searchState);
  const id2 = store.enqueueCommitted("cancel") as number;
  store.markHandled(id2, {
    transcriptOutcome: "selection_control",
    uiDisposition: "hidden",
    summary: "Search session closed",
  });
  assert.equal(store.getSnapshot().searchState, undefined);
});

test("markHandled stores contextSessionId for daemon cancel RPCs", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("fix thing") as number;
  store.markHandled(id, {
    summary: "Which file?",
    transcriptOutcome: "clarify",
    contextSessionId: "ctx-clarify-1",
  });
  assert.equal(store.clarifyPromptContextSessionId(), "ctx-clarify-1");
  const snap = store.getSnapshot();
  assert.equal(snap.clarifyPrompt?.question, "Which file?");
  assert.deepEqual(Object.keys(snap.clarifyPrompt ?? {}), [
    "question",
    "originalTranscript",
  ]);

  const id2 = store.enqueueCommitted("find foo") as number;
  store.markHandled(id2, {
    transcriptOutcome: "search",
    uiDisposition: "hidden",
    contextSessionId: "ctx-search-1",
    searchResults: [{ path: "a.ts", line: 0, character: 0, preview: "hit" }],
    activeSearchIndex: 0,
  });
  assert.equal(store.searchContextSessionId(), "ctx-search-1");
});

test("markHandled preserves search contextSessionId when follow-up omits it", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("find foo") as number;
  store.markHandled(id, {
    transcriptOutcome: "search",
    uiDisposition: "hidden",
    contextSessionId: "ctx-keep",
    searchResults: [{ path: "a.ts", line: 0, character: 0, preview: "h" }],
    activeSearchIndex: 0,
  });
  const id2 = store.enqueueCommitted("next") as number;
  store.markHandled(id2, {
    transcriptOutcome: "selection_control",
    uiDisposition: "hidden",
    searchResults: [{ path: "b.ts", line: 1, character: 0, preview: "h2" }],
    activeSearchIndex: 0,
  });
  assert.equal(store.searchContextSessionId(), "ctx-keep");
});

test("markHandled sets clarifyPrompt when transcript outcome is clarify", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("fix thing") as number;
  store.markHandled(id, {
    summary: "Which function should I edit?",
    transcriptOutcome: "clarify",
  });
  const snap = store.getSnapshot();
  assert.equal(snap.clarifyPrompt?.question, "Which function should I edit?");
  assert.equal(snap.clarifyPrompt?.originalTranscript, "fix thing");
  // Clarify is in-progress; don't add it to Recent/History yet.
  assert.equal(snap.recentHandled.length, 0);
});

test("uiDisposition=hidden prevents adding items to Recent while clarify is active", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("fix thing") as number;
  store.markHandled(id, {
    summary: "Which file?",
    transcriptOutcome: "clarify",
    uiDisposition: "hidden",
  });
  assert.ok(store.getSnapshot().clarifyPrompt);

  const filler = store.enqueueCommitted("uh") as number;
  store.markHandled(filler, {
    transcriptOutcome: "irrelevant",
    uiDisposition: "hidden",
  });
  assert.equal(store.getSnapshot().recentHandled.length, 0);

  // Explicit cancel still appends a skipped row.
  store.abortClarifyAsSkipped();
  assert.equal(store.getSnapshot().recentHandled[0]?.skipped, true);
});

test("uiDisposition=hidden keeps search/selection_control out of Recent/History", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("find foo") as number;
  store.markHandled(id, {
    transcriptOutcome: "search",
    uiDisposition: "hidden",
    searchResults: [{ path: "a.ts", line: 0, character: 0, preview: "hit" }],
    activeSearchIndex: 0,
  });
  // Search flows are not edits; don't add them to history.
  assert.equal(store.getSnapshot().recentHandled.length, 0);

  const nav = store.enqueueCommitted("next") as number;
  store.markHandled(nav, {
    transcriptOutcome: "selection_control",
    uiDisposition: "hidden",
    searchResults: [{ path: "b.ts", line: 1, character: 0, preview: "hit2" }],
    activeSearchIndex: 0,
  });
  assert.equal(store.getSnapshot().recentHandled.length, 0);
});

test("uiDisposition=hidden prevents skipped spam while searchState is active", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("find foo") as number;
  store.markHandled(id, {
    transcriptOutcome: "search",
    uiDisposition: "hidden",
    searchResults: [{ path: "a.ts", line: 0, character: 0, preview: "hit" }],
    activeSearchIndex: 0,
  });
  assert.ok(store.getSnapshot().searchState);

  // Simulate daemon returning uiDisposition=hidden (e.g. during active search navigation flow).
  const nav = store.enqueueCommitted("next") as number;
  store.markHandled(nav, {
    transcriptOutcome: "irrelevant",
    uiDisposition: "hidden",
  });
  assert.equal(store.getSnapshot().recentHandled.length, 0);
});

test("markHandled sets answerState when transcript outcome is answer", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("what is a closure") as number;
  store.markHandled(id, {
    transcriptOutcome: "answer",
    answerText: "A closure is a function plus its lexical environment.",
  });
  const snap = store.getSnapshot();
  assert.equal(snap.answerState?.question, "what is a closure");
  assert.equal(
    snap.answerState?.answerText,
    "A closure is a function plus its lexical environment.",
  );
});

test("markHandled stores Q/A in qaHistory and does not add to recentHandled", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("what is a closure") as number;
  store.markHandled(id, {
    transcriptOutcome: "answer",
    answerText: "A closure is a function plus its lexical environment.",
  });
  const snap = store.getSnapshot();
  assert.equal(snap.qaHistory?.[0]?.question, "what is a closure");
  assert.equal(
    snap.qaHistory?.[0]?.answerText,
    "A closure is a function plus its lexical environment.",
  );
  assert.equal(snap.recentHandled.length, 0);
});

test("recordCompletedTranscript appends done entry without pending", () => {
  const store = new MainPanelStore();
  store.recordCompletedTranscript("manual line", { summary: "Done." });
  const h = store.getSnapshot().recentHandled[0];
  assert.equal(h?.text, "manual line");
  assert.equal(h?.summary, "Done.");
});

test("recordCompletedTranscript can record a failed manual line for the panel", () => {
  const store = new MainPanelStore();
  store.recordCompletedTranscript("manual line", {
    errorMessage: "Failed to process transcript.",
  });
  const h = store.getSnapshot().recentHandled[0];
  assert.equal(h?.text, "manual line");
  assert.equal(h?.errorMessage, "Failed to process transcript.");
  assert.equal(h?.summary, undefined);
});

test("markError moves the line to Done without blocking Applying", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("x") as number;

  store.markError(id);
  assert.equal(store.getSnapshot().pending.length, 0);
  const h = store.getSnapshot().recentHandled[0];
  assert.equal(h?.text, "x");
  assert.equal(h?.errorMessage, undefined);
});

test("markError stores a readable error message on the Done entry", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("x") as number;

  store.markError(id, "  failed to apply directive  ");
  assert.equal(store.getSnapshot().pending.length, 0);
  assert.equal(
    store.getSnapshot().recentHandled[0]?.errorMessage,
    "failed to apply directive",
  );
});

test("markProcessing is a no-op after markError removed the line", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("x") as number;

  store.markError(id, "failed");
  assert.equal(store.getSnapshot().pending.length, 0);

  store.markProcessing(id);
  assert.equal(store.getSnapshot().pending.length, 0);
  assert.equal(store.getSnapshot().recentHandled[0]?.errorMessage, "failed");
});

test("setVoiceListening false removes live hypothesis", () => {
  const store = new MainPanelStore();
  store.setVoiceListening(true);

  store.onPartial("hello");
  assert.ok(store.getSnapshot().latestPartial);

  store.setVoiceListening(false);
  assert.equal(store.getSnapshot().latestPartial, null);
  assert.equal(store.getSnapshot().voiceListening, false);
});

test("onPartial is ignored while not listening", () => {
  const store = new MainPanelStore();

  store.onPartial("hello");
  assert.equal(store.getSnapshot().latestPartial, null);
});

test("notifies listeners on change", () => {
  const store = new MainPanelStore();
  store.setVoiceListening(true);
  let notifications = 0;

  const unsubscribe = store.onDidChange(() => {
    notifications += 1;
  });

  store.onPartial("hello");
  unsubscribe();
  store.onPartial("world");

  assert.equal(notifications, 1);
});

test("a throwing listener does not block other listeners", () => {
  const store = new MainPanelStore();
  store.setVoiceListening(true);
  let ok = 0;

  store.onDidChange(() => {
    throw new Error("bad listener");
  });
  store.onDidChange(() => {
    ok += 1;
  });

  store.onPartial("hello");

  assert.equal(ok, 1);
});

test("empty partial debounces clearing latestPartial", async () => {
  const store = new MainPanelStore(30, 40);
  store.setVoiceListening(true);
  store.onPartial("stay");
  store.onPartial("   ");
  assert.equal(store.getSnapshot().latestPartial, "stay");
  await new Promise((r) => setTimeout(r, 70));
  assert.equal(store.getSnapshot().latestPartial, null);
});

test("non-empty partial after empty cancels debounced clear", async () => {
  const store = new MainPanelStore(30, 40);
  store.setVoiceListening(true);
  store.onPartial("a");
  store.onPartial("   ");
  await new Promise((r) => setTimeout(r, 20));
  store.onPartial("b");
  await new Promise((r) => setTimeout(r, 70));
  assert.equal(store.getSnapshot().latestPartial, "b");
});

test("caps recent handled history", () => {
  const store = new MainPanelStore(2);

  const a = store.enqueueCommitted("a") as number;
  const b = store.enqueueCommitted("b") as number;
  const c = store.enqueueCommitted("c") as number;

  store.markHandled(a);
  store.markHandled(b);
  store.markHandled(c);

  assert.equal(store.getSnapshot().recentHandled.length, 2);
  assert.equal(store.getSnapshot().recentHandled[0]?.text, "c");
  assert.equal(store.getSnapshot().recentHandled[1]?.text, "b");
});
