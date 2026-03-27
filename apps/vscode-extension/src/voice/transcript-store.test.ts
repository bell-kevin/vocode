import assert from "node:assert/strict";
import test from "node:test";

import { TranscriptStore } from "./transcript-store";

test("stores transcript entries in reverse chronological order", () => {
  const store = new TranscriptStore();

  store.add("first partial", "partial");
  store.add("final transcript", "final");

  const entries = store.getEntries();

  assert.equal(entries.length, 2);
  assert.equal(entries[0]?.text, "final transcript");
  assert.equal(entries[0]?.kind, "final");
  assert.equal(entries[1]?.text, "first partial");
  assert.equal(entries[1]?.kind, "partial");
});

test("does not store empty transcript text", () => {
  const store = new TranscriptStore();

  store.add("   ", "partial");

  assert.equal(store.getEntries().length, 0);
});

test("notifies listeners when a transcript is added", () => {
  const store = new TranscriptStore();
  let notifications = 0;

  const unsubscribe = store.onDidChange(() => {
    notifications += 1;
  });

  store.add("hello", "partial");
  unsubscribe();
  store.add("world", "final");

  assert.equal(notifications, 1);
});

test("caps the number of retained transcript entries", () => {
  const store = new TranscriptStore(2);

  store.add("one", "partial");
  store.add("two", "partial");
  store.add("three", "final");

  const entries = store.getEntries();
  assert.deepEqual(
    entries.map((entry) => entry.text),
    ["three", "two"],
  );
});
