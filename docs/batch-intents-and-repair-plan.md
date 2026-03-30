# Batch intents + apply-repair loop — implementation checklist

This document is the **exact checklist** for implementing: (1) **multiple intents per model turn**, (2) **irrelevant / done / request_context / intents[]** at the model boundary, and (3) **honest apply feedback** so the model can repair after partial host failure.

It complements [`transcript-architecture-plan.md`](./transcript-architecture-plan.md) (wire, session, gathered).

**Implemented in tree:** Phase 1 (apply `status`), cumulative `IntentApplyHistory` + ephemeral full-session persistence, `TurnContext` fields, `ModelClient.NextTurn` / `TurnResult`, executor batch dispatch, stub batch of four intents, `VOCODE_DAEMON_VOICE_MAX_INTENTS_PER_BATCH` (unset → 16; **0 → no cap**). Remaining: real model JSON for `TurnResult`, prompt/schema work, optional protocol field for irrelevant summary vs `VoiceTranscriptResult.summary`.

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
- [ ] Define **boundary for “cumulative history”**: same `contextSessionId` + same logical user transcript vs. reset rules (align with idle reset / new utterance).
- [ ] Confirm **irrelevant** maps to: `success: true`, **zero directives**, optional user-visible reason via existing `summary` (or a future optional field).
- [ ] Document for prompts: **request_context** may precede **intents[]** in the same RPC via the existing inner loop pattern.

---

## Phase 1 — Three-state apply outcomes (protocol + extension + daemon)

**Problem today:** `ok: false` + message `"not attempted"` is indistinguishable from a **real** apply failure in `ConsumeHostApplyReport` — both become `FailedIntent`. Repair needs **succeeded / failed (attempted) / skipped (not attempted)**.

- [ ] **Schema** (`packages/protocol/schema/voice-transcript.directive-apply-item.schema.json`): add an explicit discriminator, e.g. `status: "ok" | "failed" | "skipped"` **or** `attempted: boolean` with clear semantics; avoid inferring skipped state only from free-text `message`.
- [ ] Run **`pnpm codegen`**; update TS validators if applicable.
- [ ] **Extension** (`apps/vscode-extension/src/transcript/apply-result.ts`): for indices after the first failure, set **`skipped`** (not `failed`).
- [ ] **Daemon** (`apps/daemon/internal/agentcontext/directive_apply_batch.go` — or successor): parse three outcomes; build:
  - [ ] `extSucc []Intent` — host applied successfully
  - [ ] `extFailed []FailedIntent` — **attempted** and failed (`PhaseExtension`, real message)
  - [ ] `extSkipped []Intent` — **not attempted** (not `FailedIntent`)
- [ ] **Tests:** consume report with mixed ok / failed / skipped; length mismatch / batch id mismatch still errors.

---

## Phase 2 — Cumulative batch history in `TurnContext`

**Goal:** The model always sees an **ordered, auditable list** of every intent from the first batch through the latest apply, with correct status — not only flat `SucceededIntents` / `FailedIntents` that lose batch structure.

- [ ] Add a structured type, e.g. `IntentRunRecord` / `ApplyHistoryEntry`: stable **index or id**, **intent JSON** (or canonical form), **status** (`applied_ok` | `apply_failed` | `skipped`), **optional message**, optional **batch ordinal** (1st host batch, 2nd host batch, …).
- [ ] **Session / executor:** append to this history when:
  - [ ] Daemon emits a batch (`SourceIntents` + `applyBatchId`).
  - [ ] Host reports `lastBatchApply` (merge statuses for that batch’s indices).
- [ ] **Persistence:** history lives in `VoiceSession` (or equivalent) for the same `contextSessionId` until idle reset or explicit clear policy.
- [ ] **`ComposeTurnContext`** (`apps/daemon/internal/agentcontext/build.go`): pass **full cumulative history** into `TurnContext` for the model (plus transcript text, editor snapshot, gathered, notes).
- [ ] **Docs:** `agentcontext` package doc comment describes the new field(s).

---

## Phase 3 — Model turn union (`ModelClient` boundary)

**Goal:** One parsed model response = **one** of irrelevant | done | request_context | intents[]; **no** extra host RPC for relevance alone.

- [ ] Introduce `TurnResult` (name TBD) in `apps/daemon/internal/agent` (or adjacent), e.g.:
  - `Irrelevant { Reason string }`
  - `Done { Summary string }`
  - `RequestContext *RequestContextIntent` (reuse intents type)
  - `Intents []Intent` (non-empty, max cap)
- [ ] Extend **`ModelClient`** (`model_client.go`): e.g. `NextTurn(ctx, TurnContext) (TurnResult, error)` **or** extend `NextIntent` to return `TurnResult` (migrate callers).
- [ ] **Stub** (`apps/daemon/internal/agent/stub/client.go`): emit batches, irrelevant, done, request_context for tests.
- [ ] **Real providers** (OpenAI / Anthropic): JSON schema / tool output matches union (when implemented).

---

## Phase 4 — Executor refactor (`apps/daemon/internal/transcript/executor`)

- [ ] Replace “one intent per model call” happy path with:
  - [ ] Call model → `TurnResult`.
  - [ ] **irrelevant** → finalize: success, zero directives, summary from reason.
  - [ ] **done** → finalize: same as today’s done path (summary).
  - [ ] **request_context** → fulfill, update gathered, append to completed/history as today, **continue** inner loop (counts toward context caps).
  - [ ] **intents[]** → **validate all** before emitting any directive; on validation failure use existing retry/gathered-note policy or fail closed.
  - [ ] For each valid intent, **dispatch in order**, append directive + parallel `SourceIntents` entry (reuse `applyHandleOutcome` logic in an inner loop for one model step).
- [ ] **Single** `VoiceTranscriptResult` per RPC still has **N directives** and **N source intents** for that batch (outstanding-only batch).
- [ ] Revisit **`maxAgentTurns`**: primarily `request_context` rounds + validation retries, not one turn per intent in the happy path; document env tuning.
- [ ] **Tests:** full batch success; irrelevant; done after batch; mixed validation failure; request_context then batch.

---

## Phase 5 — Prompts & model schema

- [ ] System/developer text: output **only outstanding intents** after partial apply; never repeat succeeded steps as new directives.
- [ ] System/developer text: input includes **full cumulative intent history** with per-intent status (Phase 2).
- [ ] Strict JSON / tool schema for the union; document **cap** on `intents.length`.

---

## Phase 6 — Extension & protocol integration checks

- [ ] E2E shape: result with 7 directives → host fails at 4 → next params carry `reportApplyBatchId` + `lastBatchApply` with **failed** vs **skipped** correct.
- [ ] Confirm **`VoiceTranscriptResult.Validate`** still holds for multi-directive batches.
- [ ] Optional UI: show irrelevant reason / done summary (already on result if mapped to `summary`).

---

## Phase 7 — Observability & limits

- [ ] Env caps documented next to `VOCODE_DAEMON_VOICE_MAX_AGENT_TURNS` in `spawn-env` / daemon config docs as needed.
- [ ] Structured log fields: batch size, repair RPC, counts of ok / failed / skipped.

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
