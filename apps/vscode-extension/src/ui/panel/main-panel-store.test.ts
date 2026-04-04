import assert from "node:assert/strict";
import test from "node:test";

import {
  MainPanelStore,
  panelHadActiveSearchInterrupt,
} from "./main-panel-store";

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

test("markHandled sets skipped when transcript uiDisposition is skipped", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("what is the weather") as number;
  store.markHandled(id, {
    summary: "Not a coding task.",
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
    clarify: { targetResolution: "instruction" },
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

test("panelHadActiveSearchInterrupt is true only when sidebar shows search/file list", () => {
  const store = new MainPanelStore();
  assert.equal(panelHadActiveSearchInterrupt(store.getSnapshot()), false);
  const id = store.enqueueCommitted("find foo") as number;
  store.markHandled(id, {
    uiDisposition: "browse",
    search: {
      results: [
        {
          path: "/x.ts",
          line: 0,
          character: 0,
          preview: "hit",
        },
      ],
      activeIndex: 0,
    },
  });
  assert.equal(panelHadActiveSearchInterrupt(store.getSnapshot()), true);
  store.dismissSearchState();
  assert.equal(panelHadActiveSearchInterrupt(store.getSnapshot()), false);
});

test("dismissSearchState clears search hit list", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("find foo") as number;
  store.markHandled(id, {
    uiDisposition: "browse",
    search: {
      results: [
        {
          path: "a.ts",
          line: 0,
          character: 0,
          preview: "hit",
        },
      ],
      activeIndex: 0,
    },
  });
  assert.ok(store.getSnapshot().searchState);
  store.dismissSearchState();
  assert.equal(store.getSnapshot().searchState, undefined);
});

test("markHandled search.closed clears search (voice cancel)", () => {
  const store = new MainPanelStore();
  const id1 = store.enqueueCommitted("find foo") as number;
  store.markHandled(id1, {
    uiDisposition: "browse",
    search: {
      results: [{ path: "a.ts", line: 0, character: 0, preview: "hit" }],
      activeIndex: 0,
    },
  });
  assert.ok(store.getSnapshot().searchState);
  const id2 = store.enqueueCommitted("cancel") as number;
  store.markHandled(id2, {
    uiDisposition: "browse",
    summary: "Search session closed",
    search: { closed: true },
  });
  assert.equal(store.getSnapshot().searchState, undefined);
});

test("markHandled fileSelection.results maps to sidebar searchState (file list)", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("open config") as number;
  store.markHandled(id, {
    uiDisposition: "browse",
    contextSessionId: "ctx-files-1",
    fileSelection: {
      results: [
        { path: "/w/src/foo.ts", preview: "foo.ts" },
        { path: "/w/src/bar.ts" },
      ],
      activeIndex: 1,
    },
  });
  const ss = store.getSnapshot().searchState;
  assert.ok(ss);
  assert.equal(ss?.listKind, "file");
  assert.equal(ss?.activeIndex, 1);
  assert.equal(ss?.results.length, 2);
  assert.equal(ss?.results[1]?.path, "/w/src/bar.ts");
  assert.equal(ss?.results[1]?.preview, "bar.ts");
  assert.equal(store.searchContextSessionId(), "ctx-files-1");
});

test("markHandled fileSelection.closed clears sidebar list", () => {
  const store = new MainPanelStore();
  const id1 = store.enqueueCommitted("pick file") as number;
  store.markHandled(id1, {
    uiDisposition: "browse",
    fileSelection: {
      results: [{ path: "/w/a.ts" }],
      activeIndex: 0,
    },
  });
  assert.ok(store.getSnapshot().searchState);
  const id2 = store.enqueueCommitted("cancel") as number;
  store.markHandled(id2, {
    uiDisposition: "browse",
    summary: "Closed",
    fileSelection: { closed: true },
  });
  assert.equal(store.getSnapshot().searchState, undefined);
});

test("markHandled stores contextSessionId for daemon cancel RPCs", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("fix thing") as number;
  store.markHandled(id, {
    summary: "Which file?",
    clarify: { targetResolution: "instruction" },
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
    uiDisposition: "browse",
    contextSessionId: "ctx-search-1",
    search: {
      results: [{ path: "a.ts", line: 0, character: 0, preview: "hit" }],
      activeIndex: 0,
    },
  });
  assert.equal(store.searchContextSessionId(), "ctx-search-1");
});

test("markHandled preserves search contextSessionId when follow-up omits it", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("find foo") as number;
  store.markHandled(id, {
    uiDisposition: "browse",
    contextSessionId: "ctx-keep",
    search: {
      results: [{ path: "a.ts", line: 0, character: 0, preview: "h" }],
      activeIndex: 0,
    },
  });
  const id2 = store.enqueueCommitted("next") as number;
  store.markHandled(id2, {
    uiDisposition: "browse",
    search: {
      results: [{ path: "b.ts", line: 1, character: 0, preview: "h2" }],
      activeIndex: 0,
    },
  });
  assert.equal(store.searchContextSessionId(), "ctx-keep");
});

test("markHandled sets clarifyPrompt when transcript outcome is clarify", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("fix thing") as number;
  store.markHandled(id, {
    summary: "Which function should I edit?",
    clarify: { targetResolution: "instruction" },
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
    clarify: { targetResolution: "instruction" },
    uiDisposition: "hidden",
  });
  assert.ok(store.getSnapshot().clarifyPrompt);

  const filler = store.enqueueCommitted("uh") as number;
  store.markHandled(filler, {
    uiDisposition: "hidden",
  });
  assert.equal(store.getSnapshot().recentHandled.length, 0);

  // Explicit cancel still appends a skipped row.
  store.abortClarifyAsSkipped();
  assert.equal(store.getSnapshot().recentHandled[0]?.skipped, true);
});

test("uiDisposition=browse keeps search updates out of Recent/History when no summary", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("find foo") as number;
  store.markHandled(id, {
    uiDisposition: "browse",
    search: {
      results: [{ path: "a.ts", line: 0, character: 0, preview: "hit" }],
      activeIndex: 0,
    },
  });
  assert.equal(store.getSnapshot().recentHandled.length, 0);

  const nav = store.enqueueCommitted("next") as number;
  store.markHandled(nav, {
    uiDisposition: "browse",
    search: {
      results: [{ path: "b.ts", line: 1, character: 0, preview: "hit2" }],
      activeIndex: 0,
    },
  });
  assert.equal(store.getSnapshot().recentHandled.length, 0);
});

test("markHandled does not log workspace search to History even when core sends a summary", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("find foo") as number;
  store.markHandled(id, {
    uiDisposition: "browse",
    summary: 'found 2 matches for "foo"',
    search: {
      results: [{ path: "a.ts", line: 0, character: 0, preview: "hit" }],
      activeIndex: 0,
    },
  });
  assert.equal(store.getSnapshot().recentHandled.length, 0);
});

test("markHandled logs hidden mutation summaries (e.g. applied edit) to History", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("pass delta into render") as number;
  store.markHandled(id, {
    uiDisposition: "hidden",
    summary: "applied edit",
  });
  assert.equal(store.getSnapshot().recentHandled.length, 1);
  assert.equal(store.getSnapshot().recentHandled[0]?.summary, "applied edit");
});

test("markHandled omits History when uiDisposition is browse even with a summary", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("anything") as number;
  store.markHandled(id, {
    uiDisposition: "browse",
    summary: "would be noisy",
  });
  assert.equal(store.getSnapshot().recentHandled.length, 0);
});

test("markHandled workspace noHits opens empty search state for sidebar", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("find zzz") as number;
  store.markHandled(id, {
    uiDisposition: "browse",
    summary: 'no matches for "zzz"',
    search: { noHits: true },
  });
  const ss = store.getSnapshot().searchState;
  assert.ok(ss);
  assert.equal(ss?.noHits, true);
  assert.equal(ss?.results.length, 0);
  assert.ok(
    (ss?.noHitsSummary ?? "").includes("zzz") ||
      ss?.noHitsSummary === 'no matches for "zzz"',
  );
  assert.equal(store.getSnapshot().recentHandled.length, 0);
});

test("uiDisposition=hidden prevents noise in Recent while searchState is active", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("find foo") as number;
  store.markHandled(id, {
    uiDisposition: "browse",
    search: {
      results: [{ path: "a.ts", line: 0, character: 0, preview: "hit" }],
      activeIndex: 0,
    },
  });
  assert.ok(store.getSnapshot().searchState);

  // During active search, daemon may force hidden (not skipped) for off-topic utterances.
  const nav = store.enqueueCommitted("next") as number;
  store.markHandled(nav, {
    uiDisposition: "hidden",
  });
  assert.equal(store.getSnapshot().recentHandled.length, 0);
});

test("markHandled sets answerState when transcript includes question answer", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("what is a closure") as number;
  store.markHandled(id, {
    question: {
      answerText: "A closure is a function plus its lexical environment.",
    },
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
    question: {
      answerText: "A closure is a function plus its lexical environment.",
    },
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

test("recordCompletedTranscript logs errors to History even when payload looks like a search", () => {
  const store = new MainPanelStore();
  store.recordCompletedTranscript("find foo", {
    errorMessage: "Search RPC failed.",
    uiDisposition: "browse",
    search: {
      results: [{ path: "a.ts", line: 0, character: 0, preview: "hit" }],
      activeIndex: 0,
    },
  });
  const h = store.getSnapshot().recentHandled[0];
  assert.equal(h?.text, "find foo");
  assert.equal(h?.errorMessage, "Search RPC failed.");
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
