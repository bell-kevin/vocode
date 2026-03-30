# Transcript / gathered context architecture (implemented plan)

## Goals

- **Voice-first UX:** No user-facing “sessions” or “clear context”; internal keys and resets only.
- **Thinner wire:** Daemon keeps executable intents for the last directive batch; host sends `applyBatchId` + `{ok, message}[]` only.
- **Batching:** Multiple directives per `voice.transcript` remain allowed; extension stops on first failure and reports **one row per directive** (tail = “not attempted”).
- **Long mic on:** **Idle reset** clears accumulated `Gathered` for a context key; **rolling cap** trims excerpts/notes/symbols while **never dropping the current `activeFile` excerpt**.

## Phases

### A — Wire + pending apply batch (daemon authority)

- Result: `applyBatchId` when `success` and `directives.length > 0`; remove `directiveIntentSteps`.
- Params: `lastBatchApply` (`status`: `ok` | `failed` | `skipped`, optional `message`), `reportApplyBatchId` when reporting.
- `TranscriptService` (`transcript` package) holds `VoiceSessionStore`, `executeMu`, and `transcript/executor.Executor`; `Executor` returns a new `DirectiveApplyBatch` when the result includes directives. Session load/save + apply report stripping live in `transcript/voicesession`; env in `transcript/config`.
- `Executor` collects parallel `[]intents.Intent` for each emitted directive; returns pending payload; rename helper to **source-intent** wording internally.
- If a prior batch was pending and the next RPC has **no** report, **drop** pending (forward progress; extension should normally report).

### B — `Gathered` policy (`agentcontext`)

- **`ApplyGatheredRollingCap(g, activeFile, maxBytes, maxExcerpts)`:** never remove excerpt whose path equals cleaned `activeFile`; evict other excerpts by slice order (oldest first) until under caps.
- **Idle reset:** `VoiceSessionStore` tracks last activity per `contextSessionId`; before `Get`, if idle > env threshold, delete stored session for that key (gathered + pending apply).
- Env: `VOCODE_DAEMON_SESSION_IDLE_RESET_MS` (unset → 30m default, `0` → off), optional cap envs for excerpts/bytes.

### C — Docs + naming

- `AGENTS.md` / `agentcontext` package docs: internal context session, no user session UX; batch apply rules.
- Executor options: `MaxAgentTurns` (env `VOCODE_DAEMON_VOICE_MAX_AGENT_TURNS`).

## Extension
- Apply directives immediately only when the daemon requests them via `host.applyDirectives`; the daemon owns the retry/repair loop.
- Committed transcript handlers are serialized so no next user transcript can start until the current apply/repair sequence finishes.
- The daemon owns the apply + repair loop: it calls `host.applyDirectives`, waits for per-directive outcomes, then continues planning in the same `voice.transcript` RPC.
- Cap for the daemon-owned apply/repair loop: `VOCODE_DAEMON_VOICE_MAX_REPAIR_RPCS` (configured from `vocode.maxTranscriptRepairRpcs`).
- Failure messages come from the specific directive dispatcher (command stderr/exit reason, edit/workspace.applyEdit failures, navigation/undo errors) and are surfaced via `lastBatchApply[i].message`.
- `applyTranscriptResult` pads outcomes to **full directive count** after failure.
