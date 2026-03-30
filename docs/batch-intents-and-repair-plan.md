# Batch intents + apply-repair loop — implementation checklist

This document is the **exact checklist** for implementing: (1) **multiple intents per model turn**, (2) **irrelevant / done / request_context / intents[]** at the model boundary, and (3) **honest apply feedback** so the model can repair after partial host failure.

It complements [`transcript-architecture-plan.md`](./transcript-architecture-plan.md) (wire, session, gathered).

**Implemented in tree:** Phase 1–5 core, executor **batch advance** fix (`advanceBatchIntentDone` so multi-item `TurnIntents` runs all items), executor **tests**, **Anthropic** Messages client, **OpenAI** client (strict **`json_schema`** turn envelope), `VOCODE_DAEMON_VOICE_LOG_TRANSCRIPT` + daemon logger line, VS Code `vocode.daemonVoiceLogTranscript` / Anthropic model+base URL settings, plus a daemon-owned **apply/repair loop** that applies directive batches via `host.applyDirectives` until directives are empty (cap: `vocode.maxTranscriptRepairRpcs`). Caps: `VOCODE_DAEMON_VOICE_MAX_INTENTS_PER_BATCH` (unset → 16; **0 → no cap**). **Remaining:** optional `VoiceTranscriptResult` field for irrelevant vs summary, full extension UI E2E, prompt tuning from real transcripts.

---

## Core contracts (non-negotiable)

### Output contract (model → daemon)

- After any partial apply, the model returns **only outstanding intents** — directives for work **not** already applied successfully on the host. **Never** re-emit intents that already succeeded in a prior batch for this transcript chain.
- Each model completion is **one** of: **irrelevant** (optional reason), **done** (optional summary), **request_context** (same semantics as today), or **non-empty `intents[]`** (capped).

### Input contract (daemon → model on repair RPCs)

- When sending the model back after a failure (or after any apply report), include **full cumulative context** of everything **attempted across all batches since the user’s utterance** (or since a defined “transcript turn” boundary): every intent that was planned, with **per-intent status** — succeeded on host, failed (attempted), or not executed (skipped because a prior directive in that batch failed).
- The model uses that **full history** to decide how to fix the plan; it still **outputs only the outstanding tail** for the next `VoiceTranscriptResult`.

### Host contract (unchanged)

- Extension applies **directives in order**; **stops after first failure**; reports **one row per directive** in `lastBatchApply` (tail rows are “not attempted” today — see Phase 1).

---

## Phase 0 — Decisions & invariants

- [x] **Max intents per batch:** `VOCODE_DAEMON_VOICE_MAX_INTENTS_PER_BATCH`; unset → 16; **0 → no cap** (VS Code `vocode.daemonVoiceMaxIntentsPerBatch`).
- [x] **Cumulative history boundary:** same `contextSessionId` (or ephemeral session) until idle reset; history appends on each host `lastBatchApply` consume.
- [x] **Irrelevant →** `success: true`, zero directives, reason → `VoiceTranscriptResult.summary` (executor).
- [x] **request_context** before **intents[]**: same RPC via executor loop (documented in `prompt.System()`).

---

## Phase 1 — Three-state apply outcomes (protocol + extension + daemon)

**Problem today:** `ok: false` + message `"not attempted"` is indistinguishable from a **real** apply failure in `ConsumeHostApplyReport` — both become `FailedIntent`. Repair needs **succeeded / failed (attempted) / skipped (not attempted)**.

- [x] **Schema:** `status: "ok" | "failed" | "skipped"` on directive apply items; `pnpm codegen`.
- [x] **Extension:** tail outcomes use **`skipped`**.
- [x] **Daemon:** `ConsumeHostApplyReport` → `extSucc` / `extFail` / `extSkipped`; tests in `directive_apply_batch_test.go`.

---

## Phase 2 — Cumulative batch history in `TurnContext`

**Goal:** The model always sees an **ordered, auditable list** of every intent from the first batch through the latest apply, with correct status — not only flat `SucceededIntents` / `FailedIntents` that lose batch structure.

- [x] **`IntentApplyRecord`** + **`IntentApplyHistory`** on `VoiceSession`; append on host report consume (`AppendIntentApplyHistory`).
- [x] **Persistence:** keyed store + full ephemeral `VoiceSession` clone between RPCs.
- [x] **`ComposeTurnContext`:** passes `IntentApplyHistory` and `SkippedIntents` into `TurnContext`.
- [x] **`agentcontext` doc** updated for `TurnContext`.

---

## Phase 3 — Model turn union (`ModelClient` boundary)

**Goal:** One parsed model response = **one** of irrelevant | done | request_context | intents[]; **no** extra host RPC for relevance alone.

- [x] **`TurnResult`** + **`ModelClient.NextTurn`** in `apps/daemon/internal/agent`.
- [x] **Stub** batch + post-apply `done`.
- [x] **OpenAI:** Chat Completions + JSON object + `turnjson.ParseTurn`.
- [x] **Anthropic:** Messages API + assistant text → `turnjson.ParseTurn`.

---

## Phase 4 — Executor refactor (`apps/daemon/internal/transcript/executor`)

- [x] Executor: `NextTurn` → switch `TurnResult`; **irrelevant** / **done** / **request_context** / batched **intents[]** with ordered dispatch; loop-cap vs `advanceBreakLoop` fix in finalize.
- [x] One result batch ↔ N directives + `SourceIntents`; inner batch loop uses `advanceBatchIntentDone` vs model-retry `advanceContinue`.
- [x] **Executor tests:** irrelevant, done, request_context→command, dispatch retry→command, multi-intent batch.

---

## Phase 5 — Prompts & model schema

- [x] **`prompt.System()`** documents union JSON, outstanding-only rule, and `attemptHistory`.
- [x] **`prompt.UserJSON`** includes transcript, editor, unified `attemptHistory` (dispatch failures + host apply outcomes), gathered (truncated excerpts).
- [x] OpenAI `json_schema` **strict** turn envelope (daemon-side; always enabled).
- [ ] Optional: prompt tuning from real transcripts.

---

## Phase 6 — Extension & protocol integration checks

- [x] Duplex apply shape: extension handles `host.applyDirectives` and returns per-directive `{status, message}` items (used for daemon repair).
- [x] **`VoiceTranscriptResult` validator:** protocol test for **seven-directive** batch + shared `applyBatchId`.
- [x] Directive dispatch failures propagate richer `lastBatchApply[i].message` (edit/command/navigation/undo) instead of a generic string.
- [x] Daemon-owned automatic repair loop runs internal apply/repair iterations until directives are empty (cap: `VOCODE_DAEMON_VOICE_MAX_REPAIR_RPCS`).
- [ ] Optional UI: **irrelevant** — *current thinking (not final):* grayed out in a **collapsed** section, using `VoiceTranscriptResult.summary` for the reason; **later:** optional **force-apply** / re-run path if something was misclassified. **Done:** show summary (same `summary` field); layout TBD.

---

## Phase 7 — Observability & limits

- [x] **`VOCODE_DAEMON_VOICE_LOG_TRANSCRIPT`:** one log line per RPC (apply ok/fail/skip counts, directive count, success, session flag, history length); VS Code `vocode.daemonVoiceLogTranscript`; requires non-nil daemon logger from `app.New`.
- [x] Caps documented in `AGENTS.md` / `spawn-env` / `package.json` for agent + voice env vars.

---

## Suggested implementation order

1. Phase 1 (three-state apply) — **unblocks honest repair context**.  
2. Phase 2 (cumulative history in session + `TurnContext`) — **unblocks correct prompts**.  
3. Phase 3 (model union + stub).  
4. Phase 4 (executor batch path).  
5. Phases 5–7 (prompts, integration polish, ops).

---

## References (code)

- Pending batch + apply consume: `apps/daemon/internal/agentcontext/directive_apply_batch.go`, `apps/daemon/internal/transcript/voicesession/voicesession.go`
- Executor loop: `apps/daemon/internal/transcript/executor/execute_iteration.go`, `apply_outcome.go`, `executor.go`
- Extension apply: `apps/vscode-extension/src/transcript/apply-result.ts`
- Params schema: `packages/protocol/schema/voice-transcript.params.schema.json`, `voice-transcript.directive-apply-item.schema.json`
