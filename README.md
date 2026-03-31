<a name="readme-top"></a>

# Vocode

Voice-driven AI code editing system powered by a local daemon and VS Code extension.

> Vocode lets you **speak code changes**, and have them applied intelligently to your project using structured edits instead of raw text replacement.

---

## 🧠 What is Vocode?

Vocode is composed of three main parts:

1. VS Code Extension (TypeScript)

- Captures voice + user intent
- Displays UI (transcripts, diffs, status)
- Sends requests to the daemon

2. Core Daemon (Go)

- Runs locally
- Handles:
  - agent logic
  - code edits (AST/diff-based)
  - symbol resolution (tree-sitter tags, grep-backed name search)
  - command execution
  - transcript agent-loop orchestration

3. Voice Sidecar (Go)

- Runs locally as a separate process
- Handles:
  - native microphone capture
  - STT provider integration
  - transcript event emission to the extension

For now, these communicate over **stdio (JSON-RPC)**. Maybe WebSocket in the future

---

## 🏗️ Repo Structure

```
apps/
  daemon/ # Go daemon (core engine)
  voice/ # Go voice sidecar (mic + STT)
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

0. **VS Code workflow:** open **Vocode → Settings** (sidebar) and save your **ElevenLabs API key** (secret storage). Other knobs live under **Settings → Vocode**; defaults are defined in `apps/vscode-extension/package.json`. There is **no** committed `.env` — the extension does not load one for spawned daemon/voice.

   **Shell / `go run`:** export the same variable names yourself (see `apps/voice/internal/app/config.go` and daemon transcript env usage); match defaults from `package.json` where you need parity.

1. Install dependencies

```bash
corepack enable
pnpm install
```

### Native voice dependencies (Windows only)
The voice sidecar (`@vocode/voice`) uses `cgo` + PortAudio, so on Windows you must install native deps before builds.

From the repo root:
```powershell
pnpm setup-portaudio:win
```

If your MSYS2 install is not at `C:\tools\msys64`, pass your MSYS2 root:
```powershell
pnpm setup-portaudio:win "D:\msys64"
```

### Native voice dependencies (Linux only)
On Ubuntu/Debian, install PortAudio dev headers + pkg-config:

```bash
sudo apt-get update
sudo apt-get install -y pkg-config portaudio19-dev
```

2. Generate protocol types

```bash
pnpm codegen
```

### Voice STT rollout/tuning

The voice sidecar uses ElevenLabs streaming STT with local VAD gating.

Recommended baseline (VS Code): use extension defaults, or set `vocode.elevenLabsSttModelId` / other `vocode.*` keys in Settings.

Tuning guide (env var names when running from a terminal, or for reading daemon code):
- higher `VOCODE_VOICE_VAD_END_MS` -> fewer premature utterance commits
- higher `VOCODE_VOICE_VAD_PREROLL_MS` -> less start-of-speech clipping
- lower `VOCODE_VOICE_STREAM_MIN_CHUNK_MS` -> lower latency while speaking
- higher `VOCODE_VOICE_STREAM_MAX_CHUNK_MS` -> fewer websocket chunk sends

Daemon transcript queueing:
- the VS Code extension forwards committed transcripts to the daemon
- the daemon processes transcripts in FIFO order and can coalesce multiple committed transcripts that arrive within `VOCODE_DAEMON_VOICE_TRANSCRIPT_COALESCE_MS`
- queue + merge bounds are configurable via `VOCODE_DAEMON_VOICE_TRANSCRIPT_QUEUE_SIZE`, `VOCODE_DAEMON_VOICE_TRANSCRIPT_MAX_MERGE_JOBS`, and `VOCODE_DAEMON_VOICE_TRANSCRIPT_MAX_MERGE_CHARS`

Tree-sitter provisioning:
- daemon symbol resolution requires the `tree-sitter` CLI
- the extension always sets `VOCODE_TREE_SITTER_BIN` for the spawned daemon to the provisioned binary at `tools/tree-sitter/<platform-arch>/tree-sitter(.exe)` (dev repo or bundled under the extension)
- run `pnpm provision:tree-sitter` to populate that path locally (extension `build` also runs this via `prebuild`)

### Daemon agent protocol (iterative)

`voice.transcript` runs an iterative agent loop inside the daemon:
1. model returns one `intents.Intent` (control or executable; JSON has top-level `kind`)
2. daemon validates and executes it (or fulfills `request_context`)
3. daemon feeds accumulated context + completed actions back to the model
4. repeats until `done` or guardrail limits are hit

Current intent `kind` values (executable unless noted):
- `edit`, `command`, `navigate`, `undo`
- `request_context` (control), `done` (control)

`voice.transcript` returns `VoiceTranscriptResult`:
- `success: true` when the daemon finished successfully
- `directives[]` (ordered execution directives with `edit`, `command`, or `navigate`)
- `success: false` when the daemon rejects/aborts the transcript before emitting directives

### Agent loop troubleshooting

When `success` is `false`, no directives are emitted and the extension should ignore the transcript (or show an error in the transcript panel).

Useful knobs while debugging the agent loop:
- `VOCODE_DAEMON_VOICE_MAX_AGENT_TURNS`
- `VOCODE_DAEMON_VOICE_MAX_INTENT_RETRIES`
- `VOCODE_DAEMON_VOICE_MAX_CONTEXT_ROUNDS`
- `VOCODE_DAEMON_VOICE_MAX_CONTEXT_BYTES`
- `VOCODE_DAEMON_VOICE_MAX_CONSECUTIVE_CONTEXT_REQUESTS`

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
├── daemon/ (spawn + path resolution)
├── voice-sidecar/ (spawn + sidecar transport)
├── ui/
└── voice/

        ↓ stdio (JSON-RPC)

Go Daemon
├── rpc/
├── agent/
├── intents/
├── symbols/ (tree-sitter `tags` subpackage)
├── workspace/
└── transcript/

Voice Sidecar
├── app/ (stdio protocol loop)
├── mic/ (native capture)
└── stt/ (provider adapters)
```

### 🔑 Key Design Principles

#### 1. Structured edits only

We **never blindly rewrite files**.

All edits are:

- orchestrated in the daemon
- anchored
- validated
- diffed before apply

The current implementation intentionally supports a small deterministic slice instead of pretending the agent covers all edit styles.

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
- edits → incremental intent iteration (currently rule-based for a small safe slice)
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

Supported today: deterministic single-file edits for `insert statement "..." inside current function`, `replace block after "..." before "..." with "..."`, and `append import "..." if missing`. The daemon returns explicit `success`/`failure`/`noop` edit outcomes so the extension can display intent-preserving UX without another agent turn.

---

### 🧱 Roadmap (Short-Term)

- [ ] JSON-RPC over stdio
- [ ] daemon-client wiring
- [ ] workspace sync
- [ ] edit intents → applier
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

<p align="right"><a href="#readme-top">back to top</a></p>
