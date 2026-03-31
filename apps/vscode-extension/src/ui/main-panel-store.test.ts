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

test("directive apply checklist rows get stable ids and update state", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("edit file") as number;
  store.appendDirectiveApplyChecklist(id, [
    "1. Edit foo.go",
    "2. Command: go test",
  ]);
  const row = store.getSnapshot().pending[0];
  assert.equal(row?.applyChecklist?.length, 2);
  assert.ok(
    row?.applyChecklist?.every(
      (c) => typeof c.id === "string" && c.id.length > 0,
    ),
  );
  store.setDirectiveApplyItemState(id, 0, "done");
  assert.equal(
    store.getSnapshot().pending[0]?.applyChecklist?.[0]?.state,
    "done",
  );
});

test("directive apply checklist appends across repair batches", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("x") as number;
  store.appendDirectiveApplyChecklist(id, ["1. a"]);
  assert.equal(store.directiveApplyChecklistLength(id), 1);
  store.appendDirectiveApplyChecklist(id, ["2. b", "3. c"]);
  assert.equal(store.directiveApplyChecklistLength(id), 3);
  assert.equal(
    store.getSnapshot().pending[0]?.applyChecklist?.[0]?.label,
    "1. a",
  );
  assert.equal(
    store.getSnapshot().pending[0]?.applyChecklist?.[2]?.label,
    "3. c",
  );
});

test("markHandled preserves directive checklist in recent history", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("apply directives") as number;
  store.appendDirectiveApplyChecklist(id, ["1. Edit a.ts", "2. npm test"]);
  store.setDirectiveApplyItemState(id, 0, "done");
  store.setDirectiveApplyItemState(id, 1, "failed", "Command failed");

  store.markHandled(id, { summary: "Applied what it could." });
  const h = store.getSnapshot().recentHandled[0];
  assert.equal(h?.applyChecklist?.length, 2);
  assert.equal(h?.applyChecklist?.[0]?.state, "done");
  assert.equal(h?.applyChecklist?.[1]?.state, "failed");
  assert.equal(h?.applyChecklist?.[1]?.message, "Command failed");
});

test("markError preserves directive checklist in recent history", () => {
  const store = new MainPanelStore();
  const id = store.enqueueCommitted("apply directives") as number;
  store.appendDirectiveApplyChecklist(id, ["1. Edit a.ts", "2. npm test"]);
  store.setDirectiveApplyItemState(id, 0, "done");
  store.setDirectiveApplyItemState(id, 1, "skipped", "Stopped after failure");

  store.markError(id, "Failed while applying");
  const h = store.getSnapshot().recentHandled[0];
  assert.equal(h?.applyChecklist?.length, 2);
  assert.equal(h?.applyChecklist?.[0]?.state, "done");
  assert.equal(h?.applyChecklist?.[1]?.state, "skipped");
  assert.equal(h?.applyChecklist?.[1]?.message, "Stopped after failure");
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
  });
  const h = store.getSnapshot().recentHandled[0];
  assert.equal(h?.skipped, true);
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
