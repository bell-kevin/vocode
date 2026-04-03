# Contributing to Vocode

Thanks for contributing to Vocode 🚀

This project is an early-stage, fast-moving system combining:

- a Go core backend (`vocode-cored`)
- a VS Code extension (UI + client)

Clarity, consistency, and clean boundaries matter a lot.

## 🧠 Before You Start

Please read:

- `README.md`
- `docs/architecture.md`
- `docs/editing-model.md` (important for edits)

Understand the separation:

`Extension (TS) → transport → Core (Go)`

Do not mix responsibilities across this boundary.

## ⚙️ Setup

1. Install dependencies
   ```bash
   pnpm install
   ```
2. Build the core backend
   ```bash
   pnpm --filter @vocode/core build
   ```
3. Run the extension

Press:

```
F5
```

This launches a VS Code Extension Development Host.

## 🧩 Project Structure

```
apps/
├── core/            # Go backend (vocode-cored)
└── vscode-extension/ # VS Code extension

packages/
├── protocol/        # shared schemas (TS + Go)
└── prompts/         # LLM prompts

docs/                # architecture + specs
```

### VS Code settings vs env vars

- **Settings UI copy** (`apps/vscode-extension/package.json` `description` fields) should stay user-oriented.
- **Technical mapping** (which `vocode.*` key becomes which `VOCODE_*` / `ELEVENLABS_*` env var, secrets, RPC-only knobs) lives in [`docs/vscode-settings-env.md`](docs/vscode-settings-env.md) and is implemented in `apps/vscode-extension/src/config/spawn-env.ts`. When you add a setting that affects spawned processes, update both the array and that doc.

## 🔑 Core Rules

1. Keep boundaries clean

   **Extension (TS):**
   - UI
   - user input (voice, commands)
   - RPC client
   - no business logic

   **Core (Go):**
   - agent logic
   - edit intent handling / orchestration (directives for the host to apply)
   - workspace + search integration
   - validated command shapes (the extension runs allowed commands)

   👉 If logic feels "smart", it belongs in the core backend.

2. No raw file rewrites

   All edits must go through:
   - structured edit model
   - diff generation
   - validation

   Never:
   - replace entire file blindly

3. stdout vs stderr (VERY IMPORTANT)

   For the core process:
   - stdout → reserved for JSON-RPC
   - stderr → logs only

   Breaking this will break the protocol later.

4. Keep things incremental

   Do NOT try to build everything at once.

   Preferred workflow:
   - minimal working version
   - test in extension host
   - expand

## 🧑‍💻 Development Workflow

- Run everything

```bash
pnpm build
```

- Lint

```bash
pnpm lint
```

- Auto-fix

```bash
pnpm lint:fix
```

- Go tests

```bash
go test ./...
```

After modifying protocol schemas:

```bash
pnpm codegen
```

(This will regenerate TypeScript and Go types and rebuild the @vocode/protocol package. If you forget to run this TypeScript and Go builds will fail.)

## 🧪 Testing Changes

### Extension

- Press `F5`
- Use Command Palette:
  - Vocode: Start Voice
  - etc.

Check logs in:

- `Extension Host → Debug Console`

### Core backend

Run manually:

```bash
pnpm --filter @vocode/core dev
```

## 🏗️ Adding Features

If adding extension functionality

- add command in:

  ```
  src/commands/
  ```

- wire to `vocode-cored` via the JSON-RPC client (historical paths under `src/daemon/`):

  ```
  src/daemon/client.ts
  ```

If adding core/backend functionality

- add logic under:

  ```
  apps/core/internal/<domain>/
  ```

- expose via RPC / handlers as appropriate for `vocode-cored`.

If adding protocol

- Update:

  ```
  packages/protocol/schema/
  ```

- Then (future):
  - regenerate TS + Go types

## 🧼 Code Style

**TypeScript**

- formatted with Biome

- run:

  ```bash
  pnpm lint:fix
  ```

**Go**

- use `gofmt`
- keep packages small and focused
- prefer explicit types over magic

## 🚫 Do Not Commit

These should NEVER be committed:

- node_modules/
- .turbo/
- dist/
- bin/
- .env
- \*.vsix

## 🔄 Pull Requests

**Before opening a PR**

- builds successfully
- no lint errors
- core (`@vocode/core`) compiles
- extension launches with `F5`

**PR Guidelines**

- keep PRs small and focused
- explain why, not just what
- link to relevant docs if needed

## 🧱 Architecture Notes

### Core backend lifecycle

- extension spawns `vocode-cored`
- core runs continuously
- extension communicates via stdio

### Future flow

Voice → Agent intent → Edits → Diff → Apply

## 🧠 Philosophy

### ✨ Magical UX, deterministic core

Vocode should feel magical to use — fast, intelligent, and effortless.

But internally, it must be deterministic, structured, and debuggable.

All operations should:

- produce predictable results
- be inspectable (diffs, structured intents)
- be reversible

The user experiences magic.  
The system runs on discipline.

## 🆘 Need Help?

- Ask in team chat
- Add a comment in docs
- Open a draft PR

## 🚀 Goal

Build a system where you can:

- Speak code changes
- See them applied instantly
- Trust the result

Speak code changes
See them applied instantly
Trust the result
