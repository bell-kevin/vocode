### Planner DSL Contract: Intuitive and Deterministic

Goal: the planner DSL we describe in `System()` must be:

- **Intuitive from the outside** — shapes are simple and obvious.
- **Deterministic from the inside** — for any such shape, if referenced paths/symbolIds exist and basic preconditions hold, it is accepted **100% of the time** by validators/dispatch/host (no hidden heuristics).

Anything that sometimes fails due to hidden internal behavior must either:

- Be removed from the public DSL, or
- Have its constraints made explicit in the DSL (and enforced structurally, not as surprising runtime heuristics).

---

### 1. Edit Intents (`EditIntent`)

**Current surface (from `executable_edit.go`):**

- `EditIntent.kind` ∈:
  - `replace`
  - `insert`
  - `delete`
  - `insert_import`
  - `create_file`
  - `append_to_file`
- `EditTarget.kind` ∈:
  - `current_file`
  - `current_cursor`
  - `current_selection`
  - `symbol_id`
  - `anchor`
  - `range`

#### 1.1 Patterns needing special treatment

- **`current_file` + `replace`**:
  - Hidden behavior today: action builder may reject with "ambiguous target" when the file has multiple candidate functions/symbols.
  - From the DSL perspective it looks like "replace this file", but in practice it is often invalid unless the file is simple.
  - **Decision**:
    - For normal edits, `current_file` replace is **not** a primary planner pattern.
    - We may retain `current_file` replace as a **rare, explicit "nuclear option"** used only when the user clearly asks to rewrite the entire file.
    - In that nuclear mode, we should relax the ambiguous‑target rule so that `current_file` + `replace` always succeeds whenever the file exists (undo remains the safety net).

- **`anchor` with broad before/after**:
  - Works only if anchors are present and unique; fails silently or with errors otherwise.
  - Currently primarily used by action builder, not planner.
  - **Conclusion**: keep as an internal tool / advanced pattern, not a first‑class DSL example unless we document strong uniqueness constraints.

- **`range` without `path`**:
  - Implicitly uses active file; may be surprising if the user expects cross‑file behavior.
  - **Conclusion**: in the DSL, always include `path` when showing `range` targets.

#### 1.2 Patterns that should be the primary DSL

For the planner DSL, we should emphasize:

- **Replace**:
  - `kind:"replace"` + `target.kind:"symbol_id"` with `symbolId.id` taken from `gathered.symbols`.
  - `kind:"replace"` + `target.kind:"range"` with explicit `path`, `startLine`/`startChar`, `endLine`/`endChar`.

- **Insert**:
  - `kind:"insert"` + `target.kind:"current_cursor"` (with optional `placement`).
  - `kind:"insert"` + `target.kind:"current_selection"` when overwriting the current selection.

- **Delete**:
  - `kind:"delete"` + `target.kind:"current_selection"`.
  - `kind:"delete"` + `target.kind:"range"` for explicit spans.

- **File‑level ops**:
  - `insert_import` with explicit `path` and an import string starting with `import `.
  - `create_file` with `path` + `content`.
  - `append_to_file` with `path` + `text`.

##### 1.2.1 Full‑file replace (nuclear option)

We may choose to support full‑file replaces as an explicit "nuclear" capability:

- Shape:

  ```json
  {
    "kind":"edit",
    "edit":{
      "kind":"replace",
      "replace":{
        "target":{"kind":"current_file","currentFile":{}},
        "newText":"entire new file contents here"
      }
    }
  }
  ```

- Intended usage:
  - Only when the user clearly asks to rewrite the entire file (e.g. "rewrite this file from scratch").
  - Not as a default pattern for normal refactors.

- Implementation expectation:
  - When this pattern is emitted and the target file exists, the edit should **always** succeed (no ambiguous‑target rejection).
  - The user’s primary safety net is editor undo; diffs may be large but intentional in this mode.

**Required behavior:** for these patterns, if:

- The `path` exists and is readable (where required),
- The `symbolId.id` appears in `gathered.symbols`,
- The `range` is within file bounds,

then the edit **must always**:

- Pass `ValidateEditIntent`,
- Pass action builder validation,
- Produce a protocol edit directive the extension can apply.

If we have extra rejection reasons (e.g. "range too large", "symbol not found"), they must be:

- Expressed structurally in the DSL (e.g. "do not replace more than N lines at once"), and
- Visible to the model via prompt text/examples.

---

### 2. Navigation Intents (`NavigationIntent`)

Kinds:

- `open_file`
- `reveal_symbol`
- `move_cursor`
- `select_range`
- `reveal_edit`

#### 2.1 Safe, intuitive navigation patterns

- **Open file**:
  - `{"kind":"navigate","navigate":{"kind":"open_file","openFile":{"path":"..."}}}`
  - Deterministic once path resolution rules are clear.

- **Reveal symbol**:
  - `{"kind":"navigate","navigate":{"kind":"reveal_symbol","revealSymbol":{"path":"optional","symbolName":"...","symbolKind":"optional"}}}`
  - Should preferentially use `symbolName` values that appear in `gathered.symbols`.

- **Move cursor / select range**:
  - `move_cursor` with `target.path` (optional) + `line` / `char`.
  - `select_range` with `target.path` (optional) + `startLine`/`startChar`/`endLine`/`endChar`.
  - If positions are out of bounds, behavior should be a safe no‑op or clamped position, not a hard failure.

- **Reveal edit**:
  - `{"kind":"navigate","navigate":{"kind":"reveal_edit","revealEdit":{"editId":"..."} } }`
  - `editId` must be taken from prior edit actions we emitted or from `attemptHistory`.

#### 2.2 Internal behavior constraints

In `TreeSitterResolver` (symbols) and candidate file search:

- We currently shell out to `rg` for candidate files. If rg is missing or fails, this can surface as:
  - `"candidate file search failed: exec: 'rg': executable file not found in %PATH% (...)"`

For the planner DSL, we want:

- Navigation intents should **not** cause transcript‑level failures due to tooling (rg/tree-sitter) being missing.
- Candidate file search failures should:
  - Return **empty results** (no matches) with an optional note in `gathered`, and
  - Allow the turn to proceed, not abort.

---

### 3. Command Intents (`CommandIntent`)

Shape:

```json
{
  "kind":"command",
  "command":{
    "command":"cmd.exe" | "powershell.exe" | "powershell" | "pwsh" | "echo",
    "args":[ "...", "..." ],
    "timeoutMs":1234 // optional
  }
}
```

Policy:

- Daemon and extension both enforce an allowlist of executables.
- Any other `command` string is **always** rejected by policy.

Plan:

- **DSL**:
  - Only show allowed executable names.
  - Provide canonical patterns for Windows (`cmd.exe /c ...`, `powershell.exe -NoProfile -Command ...`) and UNIX‑like (`echo`).
  - State explicitly: “Any other `command` string will always be rejected.”

- **Implementation**:
  - Keep validation strictly aligned with the allowlist and arg sanity checks:
    - No context‑dependent rejections beyond clear OS errors (which should flow into `attemptHistory.message` but not break the contract).

---

### 4. Undo Intents (`UndoIntent`)

Shape:

```json
{
  "kind":"undo",
  "undo":{"scope":"last_edit" | "last_transcript"}
}
```

These are already simple and deterministic.

Plan:

- Keep DSL as is.
- Ensure both scopes always map to a well‑defined undo behavior (or a safe no‑op) without validator surprises.

---

### 5. Gather‑Context (`GatherContextSpec`)

Kinds:

- `request_symbols`
- `request_file_excerpt`
- `request_usages`

#### 5.1 Expected behavior

- **`request_symbols`**:
  - Requires non‑empty `query`.
  - Uses tree‑sitter tags + candidate file search (via `rg`) under a workspace root.
  - Returns a list of `symbols.SymbolRef` in `gathered.symbols`.

- **`request_file_excerpt`**:
  - Requires resolvable `path` via `workspace.ResolveTargetPath`.
  - Reads file, truncates to a max length, stores in `gathered.excerpts`.

- **`request_usages`**:
  - Requires valid `symbolId` (parsable).
  - Builds a `\bname\b` regex and searches with `rg`.
  - Emits `gathered.notes` like `"usage: path:line:..."`.

#### 5.2 Determinism rules

For the DSL:

- If:
  - `request_symbols.query` is non‑empty and tooling is present,
  - `request_file_excerpt.path` resolves and file is readable,
  - `request_usages.symbolId` parses and rg is available,
  - then:
    - Requests must **not** abort the transcript.
    - They may legally return empty results but should not cause hard errors to the agent.

- Tooling / environment failures (missing rg, missing tree‑sitter, etc.) should:
  - Return empty results (symbols/notes) and optionally add a note to `gathered.notes`, and
  - **Not** cause `FulfillSpec` to return an error that aborts the turn.

Only truly invalid requests (like empty query or unparseable symbolId) should be hard errors, and those constraints must be stated in `System()` examples + text.

---

### 6. Repair Loop Semantics

The repair loop is driven by:

- `attemptHistory[]`:
  - `status` (`ok` | `failed` | `skipped`),
  - `phase` (`pre_execute`, `dispatch`, etc.),
  - `message` (human‑readable reason),
  - `intent` (the JSON of the prior intent).
- `gathered.notes`:
  - Notes appended by daemon logic (e.g. “daemon rejected %q intent before execution: ...; retry with corrected intent”).

#### 6.1 Principle

When a failure message in `attemptHistory`:

- Already **tells the model exactly how to fix the intent**, the correct behavior is:
  - Emit a **new `kind:"intents"` turn** with a corrected intent following the instructions,
  - Not to emit another `request_context` unless the message explicitly says context is missing.

Examples (to encode in `System()` explicitly):

- **Ambiguous target**:
  - Message: “the active file contains multiple candidate functions, so the current function is ambiguous. Fix: target a specific symbol_id (with a name), or use an anchor/range target; retry with corrected intent.”
  - **Expected repair**:
    - New `edit` with `target.kind:"symbol_id"` using one of the IDs from `gathered.symbols`, or a `range`/`anchor` as suggested.
    - No additional `request_context`.

- **Disallowed command**:
  - Message: “command is not allowed.”
  - **Expected repair**:
    - New `command` intent using one of the allowed executables with a safe arg pattern.

- **Missing path / unreadable file**:
  - Message indicates invalid path.
  - **Expected repair**:
    - Either emit a corrected path or avoid that edit entirely; not keep retrying the same invalid target.

These patterns should be backed by:

- Prompt text rules in `System()`,
- Concrete “before/after” JSON examples,
- Tests that simulate these failure paths and check that the DSL patterns are sufficient for a robust retry.

---

### 7. Concrete Next Steps

1. **Audit edit validators and action builder**:
   - Verify that:
     - `replace + symbol_id` always succeeds when symbol exists.
     - `replace + range` always succeeds when the range is valid.
   - Remove/relax any hidden heuristics that can reject these shapes unexpectedly.

2. **Adjust DSL examples**:
   - Ensure `System()` only shows:
     - `replace` with `symbol_id` / `range`,
     - `insert`/`delete` with cursor/selection/range,
     - insert_import / create_file / append_to_file with fully specified paths/content.
   - Do not show `current_file` full‑file replaces as the “normal” path.

3. **Make navigation and gather tooling‑failures soft**:
   - Confirm that rg / tree‑sitter failures result in empty results + notes, not transcript aborts.

4. **Expand repair examples in `System()`**:
   - For ambiguous target, invalid command, invalid path, etc., add:
     - an example failure message,
     - an example corrected intent,
     - a short rule that says “do X, not Y”.

5. **Scenario tests**:
   - For a small set of representative transcripts (like the `thing(name)` example), log:
     - Turn contexts,
     - Model outputs,
     - Executor outcomes.
   - Verify:
     - No unexpected validator/dispatch failures for allowed shapes,
     - Repair steps follow the DSL guidance instead of spamming `request_context`.

This document is the working checklist for making the planner DSL “what you see is what you get”: intuitive from the outside, and accepted 100% of the time by our internals whenever the referenced entities exist.

