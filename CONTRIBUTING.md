# Contributing to Vocode

Thanks for contributing to Vocode 🚀

This project is an early-stage, fast-moving system combining:

- a Go daemon (core engine)
- a VS Code extension (UI + client)

Clarity, consistency, and clean boundaries matter a lot.

## 🧠 Before You Start

Please read:

- `README.md`
- `docs/architecture.md`
- `docs/editing-model.md` (important for edits)

Understand the separation:

`Extension (TS) → transport → Daemon (Go)`

Do not mix responsibilities across this boundary.

## ⚙️ Setup

1. Install dependencies
   ```bash
   pnpm install
   ```
2. Build the daemon
   ```bash
   pnpm --filter @vocode/daemon build
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
├── daemon/          # Go backend
└── vscode-extension/ # VS Code extension

packages/
├── protocol/        # shared schemas (TS + Go)
└── prompts/         # LLM prompts

docs/                # architecture + specs
```

## 🔑 Core Rules

1. Keep boundaries clean

   **Extension (TS):**
   - UI
   - user input (voice, commands)
   - RPC client
   - no business logic

   **Daemon (Go):**
   - agent logic
   - edit intent handling / applying
   - symbol resolution (tree-sitter tags)
   - command execution
   - speech processing

   👉 If logic feels "smart", it belongs in the daemon.

2. No raw file rewrites

   All edits must go through:
   - structured edit model
   - diff generation
   - validation

   Never:
   - replace entire file blindly

3. stdout vs stderr (VERY IMPORTANT)

   For the daemon:
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

### Daemon

Run manually:

```bash
pnpm --filter @vocode/daemon dev
```

## 🏗️ Adding Features

If adding extension functionality

- add command in:

  ```
  src/commands/
  ```

- wire to daemon via:

  ```
  src/client/daemon-client.ts
  ```

If adding daemon functionality

- add logic under:

  ```
  internal/<domain>/
  ```

- expose via:

  ```
  internal/rpc/handler_*.go
  ```

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
- daemon compiles
- extension launches with `F5`

**PR Guidelines**

- keep PRs small and focused
- explain why, not just what
- link to relevant docs if needed

## 🧱 Architecture Notes

### Daemon lifecycle

- extension spawns daemon
- daemon runs continuously
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
