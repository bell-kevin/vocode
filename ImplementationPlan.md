# Implementation Plan

Tracks migration from batch action plans + fuzzy symbol lookup to iterative, deterministic planning.

## Goals

- Remove fuzzy symbol edit targeting (`symbol`) and use `symbol_id` only.
- Shift planner contract from one-shot `steps[]` to iterative single-step actions.
- Add explicit context request turns so model can ask for more code before editing.

## Phase 0: Completed Foundations

- [x] Enforce tree-sitter resolver in daemon edit path.
- [x] Provision tree-sitter binary (`pnpm provision:tree-sitter`) + extension auto-wiring.
- [x] Add `symbol_id` target support and symbol ID encode/decode utilities.
- [x] Keep runtime strict (no PATH fallback), with explicit override support.

## Phase 1: Remove Fuzzy Symbol Target (In Progress)

- [x] Remove `symbol` target from protocol schema (`edit-intent.schema.json`).
- [x] Remove `EditTargetKindSymbol` + `SymbolTarget` from daemon actionplan types.
- [x] Remove symbol-name resolution branch from `edits/action_builder.go`.
- [x] Update tests/fixtures to use `symbol_id` or non-symbol targets only.
- [x] Regenerate protocol types (`pnpm codegen`) and verify daemon + extension typechecks.

## Phase 2: Iterative Planner Contract

- [ ] Add single-step response contract (`NextAction`) with kinds:
  - `edit`
  - `navigate`
  - `run_command`
  - `request_context`
  - `done`
- [ ] Keep temporary compatibility adapter from old `ActionPlan.steps[]`.
- [ ] Add validation/guards for one-action-per-turn contract.

## Phase 3: Context Request / Fulfillment Loop

- [ ] Add `request_context` payloads (initial):
  - `request_symbols`
  - `request_file_excerpt`
  - `request_usages`
- [ ] Implement bounded context provider service in daemon.
- [ ] Add transcript/planning orchestrator loop:
  1. ask model for next action
  2. execute or fulfill context request
  3. feed results back
  4. repeat until `done`/limits
- [ ] Add caps/guardrails:
  - max turns
  - max context rounds
  - byte/token budget
  - repeated-failure bailout

## Phase 4: Client Prompting + UX

- [ ] Update model prompt contract to prefer `symbol_id` exclusively for symbol edits.
- [ ] Add concise execution trace messages for each turn.
- [ ] Return explicit “needs disambiguation/more context” outcomes when caps are hit.

## Phase 5: Cleanup

- [ ] Remove compatibility path for old `steps[]` plan after migration.
- [ ] Delete dead code/tests related to fuzzy symbol targeting.
- [ ] Document final planner protocol and troubleshooting flow in `README.md`.
