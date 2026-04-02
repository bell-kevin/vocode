# Vocode Architecture

Vocode is a voice-driven code editing system with strict ownership boundaries:

- **VS Code extension**: daemon/voice-sidecar process lifecycle, transport client calls, mechanical editor apply, user messaging.
- **Go daemon**: intent understanding, semantic safety policy, action building, orchestration.
- **Go voice sidecar**: native microphone capture, STT orchestration, transcript event emission.
- **Protocol package**: schemas, generated types, runtime validators, and shared result contracts.

Core principle: **magical UX, deterministic core**.

## Layering and ownership

Expected daemon flow:

`cmd/vocoded/main.go`  
→ `internal/app` (composition root)  
→ `internal/rpc` (transport/routing only)  
→ `internal/transcript` — `voice.transcript` RPC service, queue, session store (orchestration shell)
→ `internal/transcript/run` — one locked utterance: flows (clarify / selection / file list), **single-shot** executor + optional `host.applyDirectives`
→ `internal/transcript/executor` — narrow model pipeline: classifier → scope intent → scoped edit / format / rename / search / etc., producing protocol `directives`
→ `internal/agent` — model adapter for those prompts (`ClassifyTranscript`, `ScopeIntent`, `ScopedEdit`, …)
→ `internal/agentcontext` — gathered IDE state (`Gathered`, sessions, apply batches, model-facing context structs)

### Extension (`apps/vscode-extension`)

Owns:

- Spawning and managing daemon process lifecycle
- Spawning and managing voice sidecar lifecycle
- Sending RPC requests and receiving typed responses
- Runtime shape checks (`@vocode/protocol` validators)
- Mechanical editor application of daemon actions
- User-facing status/error/warning messages and command UX

Does not own:

- Semantic edit safety policy
- Instruction interpretation or ambiguity-resolution policy owned by the daemon agent
- Daemon business rules duplicated in UI
- Native microphone APIs and STT provider integration

### Daemon (`apps/daemon`)

Owns:

- Request interpretation and orchestration
- Semantic safety policy and deterministic failure/noop behavior
- Building validated edit actions
- Returning explicit result variants

Does not own:

- VS Code UX concerns
- UI messaging policy
- Extension/editor behavior details

### Voice sidecar (`apps/voice`)

Owns:

- Native microphone device I/O
- Audio chunking and buffering for STT
- Calling STT provider(s) and extracting transcript text
- Emitting transcript/state/error events to the extension over stdio

Does not own:

- Edit/command intent semantics
- Extension UI behavior
- Protocol schema ownership

### Protocol (`packages/protocol`)

Owns:

- JSON schema source of truth
- Generated TypeScript + Go types
- Runtime validators used by clients and services

Does not own:

- Agent-loop/orchestration/business logic
- Editor or transport implementations

## Edit directive shape (`EditDirective`)

There is **no** `edit.dispatch` or **`command.run`** JSON-RPC method; the extension does not call them. The daemon sends execution `directives` to the extension via `host.applyDirectives` during `voice.transcript`, and `voice.transcript` returns a `VoiceTranscriptCompletion` (classification + UI disposition + optional search/Q&A payloads).

**`EditDirective`** must be one explicit variant:

- `success` with `actions`
- `failure` with `failure`
- `noop` with `reason`

Mixed-state payloads are invalid. Schema, generated types, validators, and runtime behavior must remain aligned.

## Quick ownership guide

If you are about to write logic, pick the owner first:

- **UI behavior, command flow, editor operations** → extension.
- **Meaning/safety decisions, agent/orchestration logic, action construction** → daemon.
- **Payload shape and validation contract** → protocol.

One rule should have one owner. Duplicate ownership is a regression risk.

## Developer playbooks

### How to add a new extension command

1. Add a new command file in `apps/vscode-extension/src/commands`.
2. Register it in the extension command registry.
3. Add a command contribution in `apps/vscode-extension/package.json`.
4. If it talks to daemon, call client methods only (no daemon policy in command code).
5. Add/extend tests under `apps/vscode-extension/src/commands`.

Rules:

- Keep command logic orchestration/UI-level.
- Any semantic safety logic belongs in daemon.

### How to add a new RPC method

1. Add protocol schema(s) for params/result in `packages/protocol/schema`.
2. Regenerate protocol types/code.
3. Update protocol runtime validators if needed.
4. Add daemon handler in `apps/daemon/internal/rpc`:
   - decode params
   - call one service/orchestrator entrypoint
   - return result or structured RPC error
5. Register handler in `BuildHandlers`.
6. Add RPC-level tests for:
   - success path
   - expected failure/noop paths (if applicable)
   - invalid-result rejection path if result has invariants
7. Add/extend extension client call and command/UI usage.

Rules:

- Handlers must stay thin.
- Transport layer must not run the agent loop or interpret intents.
- If a method crosses multiple daemon domains, route through app-level orchestration.

### How to add a new edit action type

1. Add action schema in `packages/protocol/schema`.
2. Wire action union schema updates.
3. Regenerate TS/Go protocol types and keep validators aligned.
4. Implement daemon emission in `internal/transcript/executor` (or a small helper it calls), producing the new `EditAction` shape inside an `EditDirective`.
5. Add daemon validation for action safety/uniqueness where applicable.
6. Implement extension mechanical apply logic for the new action kind.
7. Add tests:
   - daemon executor / transcript tests
   - extension action-application tests
   - protocol validator acceptance/rejection tests
8. Ensure extension apply logic remains mechanical (no semantic policy added).

Rules:

- Daemon decides whether action is safe/valid to emit.
- Extension only performs deterministic mechanical apply + sanity checks.

### How to add a new voice edit capability

1. Extend `internal/agent` (prompt + structured result parsing) as needed.
2. Map the model output to protocol `directives` inside `internal/transcript/executor`; failures stay in the daemon.
3. Keep natural-language interpretation in the agent prompts, not in the extension.
4. Add tests under `internal/transcript/executor` and/or `internal/transcript` for the new path.

Rules:

- The model path should fail closed when the instruction is unclear.
- Executor should remain deterministic given parsed model output.
- Keep failure reasons intentional and test them.

## Testing expectations for boundary safety

When touching architecture-sensitive code, include tests in the owning layer:

- **RPC tests**: handler/server transport behavior and invalid-result rejection.
- **Agent tests**: supported parsing + unsupported/failure code expectations.
- **Transcript/executor tests**: directive construction and safety validation behavior.
- **Extension tests**: mechanical apply behavior and runtime shape handling.

## Anti-patterns

- Handler performing agent-side reasoning or target resolution
- Extension or RPC layer re-running repair loops that belong in product policy (voice is single-shot per utterance)
- Extension re-deciding daemon semantic policy
- Ambiguous/overloaded result shapes
- Placeholder layers/files with no active usage
- “Temporary” policy logic added in extension to unblock daemon work

## Contributor checklist

Before merging:

- `main.go` remains bootstrap-only
- `internal/app` remains composition + orchestration owner
- `internal/rpc` remains transport/routing only
- `internal/transcript/executor` produces protocol `EditDirective` / other directives for `host.applyDirectives` (not a separate edit RPC)
- Extension contains only mechanical apply + UI policy
- Protocol schema/types/validators/runtime behavior stay aligned
- Tests cover variant invariants and boundary behavior

## Summary

Vocode stays reliable when ownership is explicit and enforced:

- daemon decides meaning and safety,
- extension applies actions and presents UX,
- protocol defines and validates the contract.

For contributors: if you are unsure where logic belongs, choose the layer that already owns that rule, and add tests there.