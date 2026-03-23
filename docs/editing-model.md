# Editing Model

Vocode uses a structured editing model so AI-driven code changes stay reliable, inspectable, and evolvable.

## Core Principle

Magical UX, deterministic core:
users describe changes naturally; the system turns them into structured actions that can be validated, diffed, and applied safely.

## Current Implemented Slice

Today's daemon ships a deliberately small, rule-based planning layer. It is not the final editing system, but it is a safe vertical slice that proves the daemon-first architecture.

The currently supported intents are:

- `insert statement "..." inside current function`
  - Only succeeds when the active file contains exactly one supported function-shaped block.
  - The inserted statement is appended just before that function's closing brace.
- `replace block after "..." before "..." with "..."`
  - Both anchors must resolve uniquely in the active file.
  - Ambiguous or missing anchors produce a structured failure instead of guessing.
- `append import "..." if missing`
  - Currently supported for Go files and top-of-file JS/TS import sections.
  - If the import already exists, the daemon returns a structured no-change response.

These rules intentionally fail closed when the daemon cannot map the instruction to a unique edit.

## Goals

- The model must be semantic, inspectable, reversible, transportable, and independent of any single parser or editor.
- It should support simple text operations, symbol-aware edits, AST-aware resolution, and multi-file changes.

**Non-goals**

- Defaulting to raw “rewrite this whole file”.
- Coupling to tree-sitter internals.
- Depending on VS Code editor APIs.
- Relying solely on line numbers.

## High-Level Flow

User intent -> plan -> structured edit actions -> target resolution -> validation -> diff generation -> apply.

## Layers

### 1) Edit planning

- Interprets a user instruction into a narrow, deterministic edit intent.
- The current planner lives behind the daemon's `internal/agent` boundary so richer planners can replace the rules later without changing the extension contract.

### 2) Edit actions

- Describe what change should happen (create file, delete file, insert before symbol, replace function body, replace anchored range, add import, rename symbol, etc.).

### 3) Target resolution

- Determines where the action applies using file path, symbol summaries, text anchors, AST queries, or LSP data.

### 4) Application

- Performs the final mutation and generates a diff.

## Design Rule: Parser Independence

- Vocode can use tree-sitter, but the core edit model is not coupled to it.
- Edit actions stay stable while resolution strategies improve; tree-sitter is an implementation detail, not the contract.

## Edit Action Model

Every action should include:

- `kind`
- `path` or file target
- `target` descriptor
- `payload`
- optional constraints
- optional fallback anchors

**Example: insert after symbol**

```json
{
  "kind": "insert_after_symbol",
  "path": "src/auth.ts",
  "target": { "kind": "function", "name": "loginUser" },
  "content": "\nconst retries = 3;\n"
}
```

**Example: replace function body**

```json
{
  "kind": "replace_function_body",
  "path": "src/auth.ts",
  "target": { "kind": "function", "name": "loginUser" },
  "newBody": "if (!token) {\n  throw new Error(\"Missing token\");\n}\n"
}
```

**Example: anchored replacement**

```json
{
  "kind": "replace_anchored_block",
  "path": "src/auth.ts",
  "anchor": { "before": "export async function loginUser(", "after": "}\n" },
  "newText": "..."
}
```

**Example: create file**

```json
{
  "kind": "create_file",
  "path": "src/retry.ts",
  "content": "export function retry() {}\n"
}
```

## Target Descriptors

- Targets should be semantic whenever possible (function by name, class by name, method by name + parent, import section, file header, cursor enclosure, selected range).

**Suggested target schema**

```json
{
  "kind": "function",
  "name": "loginUser",
  "parent": null,
  "language": "typescript"
}
```

## Resolution Strategy

Attempt targets in this order:

1. AST-aware resolution (if available).
2. Symbol/index-based resolution.
3. Anchor-based text resolution.
4. Explicit line/range fallback.

This allows gradual upgrades without changing the edit model.

## Why Not Line Numbers Alone?

- Files change between planning and apply.
- Line offsets drift.
- Semantic intent is lost.

Line/range edits may exist as low-level primitives but should not drive planning.

## Validation

Every edit must be validated before apply by checking:

- file exists when required
- target resolves uniquely
- fallback anchors still match uniquely
- file has not changed incompatibly
- edits do not overlap
- resulting patch is well-formed

If validation fails, the system should request clarification, re-resolve, or fall back to preview-only mode.

## Diff Generation

- Generate a diff before writing files for preview, review, debugging, and logs.
- Mandatory for non-trivial edits.

## Safety Levels

- **Low risk:** insert into selected function, add import, create nearby helper file, replace small anchored block.
- **Medium risk:** modify multiple symbols in one file, update signatures, update imports in several files.
- **High risk:** delete files, move files, broad multi-file refactors, package/config mutations, shell-driven codegen.

Safety level controls whether Vocode auto-applies, previews first, or asks for confirmation.

## Multi-file Edits

- Group related edits into a transaction-like operation with an operation ID, ordered actions, validation results, and combined diff.
- Enables atomic preview, partial failure handling, and undo/revert.

## Undo and Revert

- Each apply produces an operation record with original snapshots or reverse patch, applied diff, timestamp, and action metadata.
- Supports daemon-level revert, editor integration, debugging, and auditability.

## Tree-sitter Strategy

- Tree-sitter is a resolution backend, not the core contract.
- Early resolvers may be text/symbol based; later ones can become AST-aware without changing action shapes.

## Recommended Implementation Order

- **Phase 1:** create file; replace anchored block; insert after symbol; add import; low-risk single-file apply using summaries, symbols, anchors, text validation.
- **Phase 2:** tree-sitter-backed resolution; function/class/method body replacement; better import handling; enclosing-node resolution.
- **Phase 3:** multi-file coordinated edits; rename/refactor flows; richer language-specific transforms.

## Summary

- Edit actions describe intent; resolvers determine location.
- The current rule-based planner is a safe starting slice, not the end state.
- This keeps the user experience magical while the system stays deterministic and evolvable.
