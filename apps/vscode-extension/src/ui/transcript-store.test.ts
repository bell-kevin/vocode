import assert from "node:assert/strict";
import test from "node:test";

import { TranscriptStore } from "./transcript-store";

test("clears partial buffer when a committed line arrives", () => {
  const store = new TranscriptStore();
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
  const store = new TranscriptStore();

  assert.equal(store.enqueueCommitted("   "), null);
  assert.equal(store.getSnapshot().pending.length, 0);
});

test("tracks pending through processing to handled", () => {
  const store = new TranscriptStore();
  const id = store.enqueueCommitted("run tests");

  assert.ok(id !== null);
  assert.equal(store.getSnapshot().pending[0]?.status, "queued");

  store.markProcessing(id);
  assert.equal(store.getSnapshot().pending[0]?.status, "processing");

  store.markHandled(id);
  assert.equal(store.getSnapshot().pending.length, 0);
  assert.equal(store.getSnapshot().recentHandled[0]?.text, "run tests");
});

test("markHandled stores optional planner summary", () => {
  const store = new TranscriptStore();
  const id = store.enqueueCommitted("fix the bug") as number;
  store.markHandled(id, { summary: "  Updated handler and tests.  " });
  const h = store.getSnapshot().recentHandled[0];
  assert.equal(h?.summary, "Updated handler and tests.");
});

test("recordCompletedTranscript appends done entry without pending", () => {
  const store = new TranscriptStore();
  store.recordCompletedTranscript("manual line", { summary: "Done." });
  const h = store.getSnapshot().recentHandled[0];
  assert.equal(h?.text, "manual line");
  assert.equal(h?.summary, "Done.");
});

test("markError leaves the line visible as error", () => {
  const store = new TranscriptStore();
  const id = store.enqueueCommitted("x") as number;

  store.markError(id);
  assert.equal(store.getSnapshot().pending[0]?.status, "error");
  assert.equal(store.getSnapshot().pending[0]?.errorMessage, undefined);
});

test("markError keeps a readable error message for the panel", () => {
  const store = new TranscriptStore();
  const id = store.enqueueCommitted("x") as number;

  store.markError(id, "  failed to apply directive  ");
  assert.equal(store.getSnapshot().pending[0]?.status, "error");
  assert.equal(
    store.getSnapshot().pending[0]?.errorMessage,
    "failed to apply directive",
  );
});

test("markProcessing clears stale error message", () => {
  const store = new TranscriptStore();
  const id = store.enqueueCommitted("x") as number;

  store.markError(id, "failed");
  assert.equal(store.getSnapshot().pending[0]?.errorMessage, "failed");

  store.markProcessing(id);
  assert.equal(store.getSnapshot().pending[0]?.status, "processing");
  assert.equal(store.getSnapshot().pending[0]?.errorMessage, undefined);
});

test("setVoiceListening false removes live hypothesis", () => {
  const store = new TranscriptStore();
  store.setVoiceListening(true);

  store.onPartial("hello");
  assert.ok(store.getSnapshot().latestPartial);

  store.setVoiceListening(false);
  assert.equal(store.getSnapshot().latestPartial, null);
  assert.equal(store.getSnapshot().voiceListening, false);
});

test("onPartial is ignored while not listening", () => {
  const store = new TranscriptStore();

  store.onPartial("hello");
  assert.equal(store.getSnapshot().latestPartial, null);
});

test("notifies listeners on change", () => {
  const store = new TranscriptStore();
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
  const store = new TranscriptStore();
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
  const store = new TranscriptStore(30, 40);
  store.setVoiceListening(true);
  store.onPartial("stay");
  store.onPartial("   ");
  assert.equal(store.getSnapshot().latestPartial, "stay");
  await new Promise((r) => setTimeout(r, 70));
  assert.equal(store.getSnapshot().latestPartial, null);
});

test("non-empty partial after empty cancels debounced clear", async () => {
  const store = new TranscriptStore(30, 40);
  store.setVoiceListening(true);
  store.onPartial("a");
  store.onPartial("   ");
  await new Promise((r) => setTimeout(r, 20));
  store.onPartial("b");
  await new Promise((r) => setTimeout(r, 70));
  assert.equal(store.getSnapshot().latestPartial, "b");
});

test("caps recent handled history", () => {
  const store = new TranscriptStore(2);

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
