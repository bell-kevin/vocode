# Agent Guidance (Architecture Contract)

This workspace is intentionally split into three execution hosts with strict ownership boundaries:

1. **Daemon (`apps/daemon`)**: reasoning + semantic safety policy + converting model output into validated, protocol-shaped results.
2. **VS Code extension (`apps/vscode-extension`)**: transport client calls + deterministic editor mechanical apply + executing allowed shell commands + user messaging.
3. **Voice sidecar (`apps/voice`)**: native microphone capture + STT orchestration + transcript event emission.

Agents must preserve these boundaries to keep behavior predictable and to avoid “side doors”.

## End Goal Mental Model (Contract-First)

One “turn” starts when the extension calls the daemon RPC:

`voice.transcript(text, activeFile?)` → `VoiceTranscriptCompletion`

### What the daemon guarantees

1. For `voice.transcript`, the daemon runs a **narrow-model** pipeline in `internal/transcript/executor` (classifier → scope intent → scoped edit / format / rename / search / …), not an iterative `intents[]` loop.
2. **Single-shot apply per utterance**: at most one `executor.Execute` and one `host.applyDirectives` batch for that RPC. On `stale_range` or other apply failures, the daemon returns failure; the user re-speaks (99-like)—no daemon-side repair retries.
3. The daemon returns a `VoiceTranscriptCompletion` containing:
   - `success`
   - optional `summary`
   - `transcriptOutcome` (e.g. `search`, `selection`, `selection_control`, `file_selection`, `file_selection_control`, `needs_workspace_folder`, `clarify`, `clarify_control`, `irrelevant`, `answer`, `completed`)
   - `uiDisposition` (`shown | skipped | hidden`) telling the host where to log the completion
   - optional `searchResults` + `activeSearchIndex` for search flows
   - optional `answerText` for Q/A  
   Per-`contextSessionId` state lives in `VoiceSessionStore`, subject to idle reset (`VOCODE_DAEMON_SESSION_IDLE_RESET_MS`: unset → 30m default, `0` → off) and a rolling byte/excerpt cap (active-file excerpt is retained).
4. Each directive is exactly one of:
   - `kind: "edit"` with an `editDirective` (a single explicit variant of `EditDirective`)
   - `kind: "command"` with `commandDirective` (daemon-validated command shape; extension executes)
   - `kind: "navigate"` with `navigationDirective` (extension applies deterministic UI navigation)

### What the extension guarantees

1. The extension implements the `host.applyDirectives` RPC: it applies directives sequentially and returns one outcome row per directive (`status: ok | failed | skipped`, optional `message`). Tail rows use `skipped` after the first failure.
2. For `edit` directives, it applies daemon-provided edit actions mechanically using `workspace.applyEdit`.
3. For `command` directives, it runs the command parameters using an allowlisted runner (no additional semantic policy).
4. If any directive fails, the extension stops processing remaining directives in that batch.
5. The extension treats `VoiceTranscriptCompletion.uiDisposition` as authoritative for UI logging (no host-side heuristics).

### Invariant: no mixed-state payloads

- `EditDirective` must be exactly one explicit variant: `success` (with `actions`) or `failure` (with `failure`) or `noop` (with `reason`).
- “Mixed-state” (multiple variants at once) is invalid and must be rejected/avoided. Keep schema, generated types, and runtime validators aligned.

### Invariant: no daemon “execution RPCs”

- The extension does not call `edit.dispatch` or `command.run`.
- The agent execution entrypoint is `voice.transcript`.

## Layering and Ownership (Where Logic Belongs)

### Daemon owns

- Meaning and safety decisions (semantic validation, deterministic failure/noop behavior)
- Action construction and orchestration
- Converting model output into protocol-shaped `directives` and completions
- Deterministic ordering of directives within a single batch

### Extension owns

- VS Code lifecycle and transport client calls
- Mechanical editor apply
- User-facing status/error/warning messaging
- Executing allowed commands and surfacing outputs
- Runtime shape checks using `@vocode/protocol` validators

### Voice sidecar owns

- Native microphone capture on device APIs
- Audio buffering/chunking before STT
- STT provider integration and request shaping
- Emitting transcript/error/state events over stdio

### Protocol owns

- JSON schema source of truth
- Generated TypeScript and Go types
- Runtime validators used at integration boundaries
- **Policy-free constructors and validators**: `packages/protocol` must only express wire shape and schema invariants. No daemon/extension policy (allowlists, safety semantics, env defaults, retries) in validators or `go/*_constructors.go` helpers — see `packages/protocol/README.md`.

## Quick Ownership Guide

- UI behavior, command flow, editor operations → extension.
- Meaning/safety decisions, agent/orchestration logic, action construction → daemon.
- Payload shape and validation contract → protocol.

One rule should have one owner. Duplicating ownership is a regression risk.

## What agents must not do

- Do not add daemon-side execution calls to run shell commands. The daemon should validate and return `commandDirective`; the extension executes.
- Do not add new RPC endpoints like `edit.dispatch` or `command.run`. Keep the main entrypoint as `voice.transcript`.
- Do not move semantic policy logic into the extension (keep it in daemon-owned layers).
- Do not mix editor/transport policy decisions with agent/orchestration concerns.
- Do not move native microphone or STT transport logic into the extension host; keep it in `apps/voice`.

## Daemon package boundaries (voice transcript)

- **`internal/agentcontext`:** session-shaped data (`VoiceSession`, `FlowStack`, `Gathered`, clarify-target rules). No RPC orchestration, no calls into `internal/transcript` or `internal/transcript/executor`.
- **`internal/transcript` (root package):** `TranscriptService` — RPC entry, queue/worker, mutex; delegates each utterance to **`internal/transcript/run`**.
- **`internal/transcript/run`:** one locked utterance (session load, control / clarify / selection / file-selection, executor, `host.applyDirectives`, persist). **Must not** import the parent `transcript` package (no cycles).
- **`internal/transcript/executor`:** narrow model pipeline for a single execute call → directives.
- Target dependency direction: `transcript` → `run` → `executor` → … → `agentcontext` / `agent`.

## Where to look (key files)

### Daemon

- Agent/runtime: `apps/daemon/internal/agent` — transcript/scoped-edit model calls (`ModelClient`: `ClassifyTranscript`, `ScopeIntent`, `ScopedEdit`, …). Prompts: `apps/daemon/internal/agent/prompt`. Providers: `openai` (`VOCODE_AGENT_PROVIDER=openai`, `OPENAI_API_KEY`, optional `VOCODE_OPENAI_MODEL` / `VOCODE_OPENAI_BASE_URL`); `anthropic` (`VOCODE_AGENT_PROVIDER=anthropic`, `ANTHROPIC_API_KEY`, optional `VOCODE_ANTHROPIC_MODEL` / `VOCODE_ANTHROPIC_BASE_URL`). Transcript debug log line: `VOCODE_DAEMON_VOICE_LOG_TRANSCRIPT=1` (needs daemon logger). Context types: `apps/daemon/internal/agentcontext` (`Gathered`, sessions, apply batches, model input structs). Tree-sitter tag parsing + cursor containment: `apps/daemon/internal/symbols/tags`
- Voice transcript (`apps/daemon/internal/transcript/`): root `service*.go` (RPC, queue); `transcript/run` (per-utterance orchestration); `transcript/executor` (narrow pipeline → protocol directives); `transcript/voicesession`; `transcript/config`; see `transcript/doc.go`

### Extension

- Send transcript: `apps/vscode-extension/src/commands/send-transcript.ts`
- Apply daemon directive batches + return per-directive outcomes to daemon: `apps/vscode-extension/src/voice-transcript/apply-directives.ts`, `apps/vscode-extension/src/daemon/rpc-transport.ts` (handler for `host.applyDirectives`)
- Voice main sidebar (webview UI state): `apps/vscode-extension/src/ui/panel/main-panel-store.ts` + `ui/panel/main-panel.ts`
- Directive host layer (`apps/vscode-extension/src/directives/`): `command`, `edits`, `navigation`, `undo` (each has `dispatch.ts` exporting `dispatchCommand` / `dispatchEdit` / `dispatchNavigation` / `dispatchUndo`), plus root `dispatch.ts` (`dispatchTranscript`)
- Daemon client: `apps/vscode-extension/src/daemon/client.ts`
- Voice sidecar spawn/client: `apps/vscode-extension/src/voice` (`client`, `spawn`, `paths`)
- Spawned daemon/voice env: `apps/vscode-extension/src/config/spawn-env.ts` — `package.json` configuration defaults + effective VS Code `vocode.*` settings + ElevenLabs key from SecretStorage (no workspace `.env`). **API keys** for cloud models are **not** in settings: set `OPENAI_API_KEY` and/or `ANTHROPIC_API_KEY` in your environment when using `vocode.daemonAgentProvider` `openai` or `anthropic`. For `go run` / shell workflows, export vars yourself; default numbers/strings match `package.json` where applicable.

### Voice sidecar

- Entrypoint: `apps/voice/cmd/vocode-voiced/main.go`
- Stdio app protocol loop: `apps/voice/internal/app/app.go`
- Native mic capture: `apps/voice/internal/mic`
- STT adapter(s): `apps/voice/internal/stt`

  Rule: downstream STT must eventually produce transcript text, which is sent to the daemon via `voice.transcript`.

## Extending the system (checklist)

### Add a new edit capability

1. Extend `internal/agent` prompts/parsing and `internal/transcript/executor` to emit the new edit shape.
2. Update the extension mechanical apply logic if you introduce a new protocol action kind.
3. Add tests in the owning layers:
   - daemon: executor/transcript tests for the new path
   - extension: mechanical apply behavior for the new action kind
   - protocol: validator acceptance/rejection if you updated schemas

### Add a new command capability

1. Ensure the narrow pipeline can produce a validated `commandDirective`.
2. Update daemon command validation / allowlist where it lives (executor or helper).
3. Update extension allowlist in `apps/vscode-extension/src/directives/command/execute-command.ts`.
4. Keep execution semantics in the extension; keep command-shape validation in the daemon.

### Change directive ordering semantics

- Ordering is contractually sequential within one batch: the extension processes directives in order.
- If failure semantics change, update both daemon batch construction and extension stop-on-failure behavior.

## Developer Playbooks (Short Summary)

### Add a new extension command

1. Add a new command file under `apps/vscode-extension/src/commands`.
2. Register it in the extension command registry.
3. If it talks to the daemon, call client methods only (no daemon policy logic in the command).
4. Add/extend extension tests for the new command behavior.

Rules:
- Keep command logic orchestration/UI-level.
- Any semantic safety logic belongs in daemon-owned layers.

### Add a new RPC method

1. Add protocol schema(s) for params/result in `packages/protocol/schema`.
2. Run `pnpm codegen`.
3. Update runtime validators if the schema requires it.
4. Add a daemon handler under `apps/daemon/internal/rpc`:
   - decode params
   - call one daemon entrypoint (thin handler)
   - return result or structured RPC error
5. Update the extension daemon client and command usage.
6. Add RPC-level tests for success/failure/noop and invalid-result rejection when invariants exist.

Rules:
- Handlers must stay thin.
- Transport must not run the transcript executor or interpret model output.
- If a request crosses multiple daemon domains, route through app-level orchestration.

### Add a new edit action type

1. Add action schema in `packages/protocol/schema`.
2. Wire the action union schema updates.
3. Regenerate TS/Go protocol types with `pnpm codegen`.
4. Implement daemon action building/validation in `apps/daemon/internal/transcript/executor`.
5. Implement extension mechanical apply logic for the new action kind.
6. Add tests:
   - daemon action-building + validation tests
   - extension action-application tests
   - protocol validator acceptance/rejection tests

Rules:
- Daemon decides whether an action is safe/valid to emit.
- Extension only applies actions deterministically (no semantic policy).

### Add a new voice-scoped edit capability

1. Extend agent prompt + structured parsing in `apps/daemon/internal/agent`.
2. Map parsed output to `EditDirective` / actions in `apps/daemon/internal/transcript/executor`.
3. Add executor/transcript tests for supported/unsupported instruction expectations and failure reasons.

Rules:
- Fail closed when the instruction is unclear.
- Keep natural language in prompts; keep directive emission deterministic given parsed output.

## Contributor Checklist (Boundary Safety)

Before merging:

- `internal/app` remains composition + orchestration owner.
- `internal/rpc` remains transport/routing only (thin handlers).
- `internal/transcript/executor` builds protocol `EditDirective` / other directives for `host.applyDirectives` (not a separate edit RPC).
- Extension contains only mechanical apply + UI-level orchestration; no semantic policy duplication.
- Protocol schema/types/validators/runtime behavior stay aligned.
- Tests cover variant invariants and boundary behavior.

## Protocol & Codegen Rules

- Treat `packages/protocol/schema` as source of truth.
- After schema changes, regenerate code/validators via `pnpm codegen`.
- Never “hand edit” generated protocol files; keep schema/types/validators aligned.

## Project Scripts (root `package.json`)

These are the scripts you should run for cleanliness and correctness:

- `pnpm build`: runs `turbo run build` across packages.
- `pnpm test`: runs `turbo run test`.
- `pnpm typecheck`: runs `turbo run typecheck`.
- `pnpm lint`: `biome check .` (read-only; use for verification).
- `pnpm lint:fix`: `biome check --write .` (auto-fix formatting/import/lint issues).
- `pnpm format`: `biome format .` (format without writing lint fixes).
- `pnpm format:write`: `biome format --write .` (write formatter output).
- `pnpm fix`: `pnpm lint:fix && go fmt ./...`.
- `pnpm codegen`: runs:
  - `pnpm codegen:ts`: `node scripts/codegen/generate-protocol-ts.mjs && pnpm --filter @vocode/protocol build`
  - `pnpm codegen:go`: `node scripts/codegen/generate-protocol-go.mjs`
- `pnpm dev`: `turbo run dev --parallel`.

## Anti-patterns (regression risks)

- Handler performing agent-side reasoning or target resolution.
- Daemon repair retries on `stale_range` inside a single `voice.transcript` RPC (single-shot only).
- Extension re-deciding semantic policy already owned by daemon.
- Ambiguous/overloaded result shapes (breaks runtime validators and tests).
- Adding “temporary” policy logic in the extension to unblock daemon work.

## Testing expectations for boundary safety

When touching architecture-sensitive code, add tests in the owning layer:

- RPC/tests: transport behavior + invalid-result rejection.
- Agent tests: supported parsing + unsupported/failure code expectations.
- Transcript/executor tests: directive building and safety validation behavior.
- Extension tests: mechanical apply behavior + runtime shape handling.

