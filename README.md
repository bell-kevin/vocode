# Vocode

Voice-driven AI code editing system powered by a local daemon and VS Code extension.

> Vocode lets you **speak code changes**, and have them applied intelligently to your project using structured edits instead of raw text replacement.

---

## 🧠 What is Vocode?

Vocode is composed of two main parts:

1. VS Code Extension (TypeScript)

- Captures voice + user intent
- Displays UI (transcripts, diffs, status)
- Sends requests to the daemon

2. Core Daemon (Go)

- Runs locally
- Handles:
  - agent logic
  - code edits (AST/diff-based)
  - indexing (grep → symbols → AST)
  - command execution
  - speech processing (streaming STT)

For now, these communicate over **stdio (JSON-RPC)**. Maybe WebSocket in the future

---

## 🏗️ Repo Structure

```
apps/
  daemon/ # Go daemon (core engine)
  vscode-extension/ # VS Code extension (UI + client)

packages/
  protocol/ # Shared schemas (Go + TS)
  prompts/ # LLM prompts

docs/
  architecture.md
  editing-model.md
  indexing.md
  protocol.md

scripts/
  dev/ # Build + dev scripts
  codegen/ # Protocol/code generation (future)

config/
  vocode.example.json
```

---

## 🚀 Getting Started

1. Install dependencies

```bash
corepack enable
pnpm install
```

2. Generate protocol types

```bash
pnpm codegen
```

3. Build the daemon

```bash
pnpm --filter @vocode/daemon build
```

This creates:

```
apps/daemon/bin/<platform-arch>/vocoded(.exe)
```

4. Run the extension
   Press:

```
F5
```

This launches a **VS Code Extension Development Host**.

You should see logs like:

```
Vocode extension activated
[vocode] using dev daemon: ...
[vocoded stderr] vocoded starting...
```

---

## ⚙️ Development Workflow

### Build everything

```
pnpm build
```

### Run linting

```
pnpm lint
```

### Auto-fix formatting

```
pnpm lint:fix
```

### Run Go tests

```
go test ./...
```

### 🧩 Architecture Overview

```
VS Code Extension
├── commands/
├── client/ (RPC layer)
├── daemon/ (spawn + path resolution)
├── ui/
└── voice/

        ↓ stdio (JSON-RPC)

Go Daemon
├── rpc/
├── agent/
├── edits/
├── indexing/
├── workspace/
└── speech/
```

### 🔑 Key Design Principles

#### 1. Structured edits only

We **never blindly rewrite files**.

All edits are:

- planned in the daemon
- anchored
- validated
- diffed before apply

The current implementation intentionally supports a small deterministic slice instead of pretending the planner is finished.

#### 2. Daemon-first architecture

All intelligence lives in the daemon.

The extension is just:

- input/output
- UI
- transport

#### 3. Local-first (ultimately, but will use elevenlabs cloud service for the hackathon)

- No cloud dependency required
- Works offline (future: whisper.cpp)
- Fast + private

#### 4. Streaming everything

- voice → streaming STT
- edits → incremental planning (currently rule-based for a small safe slice)
- UI → live feedback

---

### 🛠️ Current Status

- ✅ Extension boots
- ✅ Daemon spawns
- ✅ Cross-platform daemon build
- 🚧 JSON-RPC transport (next)
- 🚧 Rich edit engine wiring beyond the initial safe slice
- 🚧 Voice pipeline

---

### 📦 Daemon Build Details

The daemon is built per platform:

```
bin/
  win32-x64/vocoded.exe
  darwin-arm64/vocoded
  linux-x64/vocoded
```

The extension automatically resolves:

- dev path (monorepo)
- bundled path (production)

---

### 🧪 Testing the Extension

Inside the Extension Development Host:

Open Command Palette:

```
Vocode: Start Voice
Vocode: Stop Voice
Vocode: Apply Edit
Vocode: Run Command
```

Supported today: deterministic single-file edits for `insert statement "..." inside current function`, `replace block after "..." before "..." with "..."`, and `append import "..." if missing`. The daemon now returns structured failures when it cannot produce a safe edit.

---

### 🧱 Roadmap (Short-Term)

- [ ] JSON-RPC over stdio
- [ ] daemon-client wiring
- [ ] workspace sync
- [ ] edit planner → applier
- [ ] diff UI panel
- [ ] streaming speech input

---

### 🧑‍💻 Contributing

See `CONTRIBUTING.md` for more information

1. Install deps: `pnpm install`
2. Generate protocol types: `pnpm codegen`
3. Build daemon: `pnpm --filter @vocode/daemon build`
4. Press `F5` to run extension
5. Make changes
6. Run:

```
pnpm lint:fix
```

---

### ⚠️ Notes

- Do not commit:
  - node_modules/
  - .turbo/
  - dist/
  - bin/
- Daemon logs go to stderr
- Daemon stdout will be reserved for RPC protocol

---

### 📄 License

TBD

---

### 🧠 Vision

> Speak code.
> Watch it evolve.
> Stay in the flow.
