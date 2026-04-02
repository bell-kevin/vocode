# Transcript / gathered context architecture (implemented plan)

## Goals

- **Voice-first UX:** No user-facing sessions or clear-context UX; internal keys and resets only.
- **Thinner wire:** daemon sends directive batches via `host.applyDirectives`; host returns per-directive `{status, message}[]` only.
- **Batching:** Multiple directives per `voice.transcript` remain allowed; extension stops on first failure and reports **one row per directive** (tail = skipped / not attempted).
- **Long mic on:** **Idle reset** clears stored voice session for a context key; **rolling cap** trims gathered excerpts while **never dropping the current `activeFile` excerpt**.

## Phases

### A — Wire + pending apply batch (daemon authority)

- Daemon→host: `host.applyDirectives` with `applyBatchId`, `activeFile`, `directives[]`.
- Host→daemon: `HostApplyResult.items[]` (`status`: `ok` | `failed` | `skipped`, optional `message`).
- `voice.transcript` returns `VoiceTranscriptCompletion` (classification + UI disposition + optional search/Q&A payloads). It does not return directives on the completion object; directives are applied via `host.applyDirectives` in the same RPC when needed.
- `TranscriptService` (`internal/transcript`) holds `VoiceSessionStore`, `executeMu`, and `transcript/executor.Executor`; each utterance runs under the mutex via `transcript/run.Execute`. **Single-shot:** one executor pass and at most one host apply batch per utterance (no daemon-side repair loop).
- Session load/save + apply report consumption live in `transcript/voicesession`; tuning defaults in `transcript/config`; per-RPC overrides via `daemonConfig` on `voice.transcript` (`sessionIdleResetMs`, `maxGatheredBytes`, `maxGatheredExcerpts`).
- Flow stack + clarify targets: `internal/agentcontext` (`FlowStack`, `FlowFrame` carries clarify question + original utterance + target key, clarify registry). Utterance orchestration is `internal/transcript/run`; root `transcript` is RPC + queue. **`run` must not import `transcript`** (acyclic graph).

### B — `Gathered` policy (`agentcontext`)

- **`ApplyGatheredRollingCap(g, activeFile, maxBytes, maxExcerpts)`:** never remove excerpt whose path equals cleaned `activeFile`; evict other excerpts by slice order (oldest first) until under caps.
- **Idle reset:** `VoiceSessionStore` tracks last activity per `contextSessionId`; before `Get`, if idle elapsed, delete stored session for that key.

### C — Control RPC

- `controlRequest`: `cancel_clarify` | `cancel_selection` (close code match list / selection session without spoken text).

## Extension

- Code layout: `apps/vscode-extension/src/voice-transcript/` (RPC helpers, `apply-directives`, workspace root), `src/ui/panel/` (main webview provider + store).
- Apply directives when the daemon requests them via `host.applyDirectives` during `voice.transcript`; return per-directive outcomes in one shot.
- Committed transcript handlers are serialized so a new user transcript does not start until the current RPC finishes.
- Failure messages come from directive dispatchers and are surfaced in `HostApplyResult.items[i].message`.
