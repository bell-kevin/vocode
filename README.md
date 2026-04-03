<a name="readme-top"></a>

# Vocode

Voice-driven AI code editing system powered by a local Go core (`vocode-cored`) and VS Code extension.

> Vocode lets you **speak code changes**, and have them applied intelligently to your project using structured edits instead of raw text replacement.

---

## 🧠 What is Vocode?

Vocode is composed of three main parts:

1. VS Code Extension (TypeScript)

- Captures voice + user intent
- Displays UI (transcripts, diffs, status)
- Sends requests to `vocode-cored` (JSON-RPC)

2. Core backend (Go, `apps/core`)

- Runs locally
- Handles:
  - agent logic
  - code edits (AST/diff-based)
  - symbol resolution (host-provided LSP document symbols)
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
  core/ # Go core engine (vocode-cored)
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

0. **VS Code workflow:** open **Vocode → Settings** (sidebar) and save your **ElevenLabs API key** (secret storage). Other knobs live under **Settings → Vocode**; defaults are defined in `apps/vscode-extension/package.json`. There is **no** committed `.env` — the extension does not load one for spawned core / voice processes.

   **Contributors / shell parity:** see [`docs/vscode-settings-env.md`](docs/vscode-settings-env.md) for how each `vocode.*` setting maps to environment variables when spawning processes. For `go run` or manual binaries, export those names yourself and align values with `package.json` defaults where needed.

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

Tuning guide (env var names when running from a terminal, or for reading core code):
- higher `VOCODE_VOICE_VAD_END_MS` -> fewer premature utterance commits
- higher `VOCODE_VOICE_VAD_PREROLL_MS` -> less start-of-speech clipping
- lower `VOCODE_VOICE_STREAM_MIN_CHUNK_MS` -> lower latency while speaking
- higher `VOCODE_VOICE_STREAM_MAX_CHUNK_MS` -> fewer websocket chunk sends

Core transcript queueing:
- the VS Code extension forwards committed transcripts to `vocode-cored`
- the core processes transcripts in FIFO order and can coalesce multiple committed transcripts that arrive within `VOCODE_DAEMON_VOICE_TRANSCRIPT_COALESCE_MS`
- queue + merge bounds are configurable via `VOCODE_DAEMON_VOICE_TRANSCRIPT_QUEUE_SIZE`, `VOCODE_DAEMON_VOICE_TRANSCRIPT_MAX_MERGE_JOBS`, and `VOCODE_DAEMON_VOICE_TRANSCRIPT_MAX_MERGE_CHARS`

### Voice transcript (single-shot, core)

`voice.transcript` runs a **narrow-model** pipeline in `apps/core/internal/transcript/` (pipeline, searchapply, …). It is **not** an iterative `intents[]` loop.

Per committed utterance: at most one transcript pipeline pass and one `host.applyDirectives` batch. On apply failure (e.g. `stale_range`), the core returns `success: false`; the user re-speaks.

`voice.transcript` returns `VoiceTranscriptCompletion`:
- `success` and optional `summary`
- `transcriptOutcome` (e.g. `search`, `selection`, `selection_control`, `file_selection`, `file_selection_control`, `clarify`, `clarify_control`, `needs_workspace_folder`, `irrelevant`, `answer`, `completed`)
- `uiDisposition` (`shown | skipped | hidden`)
- optional `searchResults` + `activeSearchIndex`, `answerText`, etc.

`transcriptOutcome` + `uiDisposition` quick guide:
- `search` / `selection` / `selection_control` / `file_selection` / `file_selection_control` → usually `hidden` (panel flows)
- `clarify` / `clarify_control` → `hidden`
- `irrelevant` → `skipped` (or `hidden` during an active match-list session)
- `answer` → `hidden` (Chat UI)
- successful edit completion → `shown`

### Transcript troubleshooting

When `success` is `false`, treat directives as invalid. Session tuning uses `vocode.sessionIdleResetMs` and `daemonConfig` on the RPC (`maxGatheredBytes`, `maxGatheredExcerpts`).

3. Build the core backend (`vocode-cored`)

```bash
pnpm --filter @vocode/core build
```

This creates:

```
apps/core/bin/<platform-arch>/vocode-cored(.exe)
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
[vocode] using dev binary path (log label may still say daemon): ...
[vocode-cored stderr] vocode-cored starting...
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
├── daemon/ (spawn + JSON-RPC client for vocode-cored; historical folder name)
├── directives/ (host apply for protocol directives)
├── voice-transcript/ (voice.transcript RPC + apply batch)
├── ui/
│   └── panel/ (main webview + store)
└── voice/ (sidecar client + spawn)

        ↓ stdio (JSON-RPC)

Go core (`apps/core`)
├── rpc/
├── agent/
├── flows/ (route classification + per-flow handlers)
├── workspace/
├── search/ (e.g. ripgrep-backed search)
└── transcript/ (service, pipeline, session, searchapply, ...)

Voice Sidecar
├── app/ (stdio protocol loop)
├── mic/ (native capture)
└── stt/ (provider adapters)
```

### 🔑 Key Design Principles

#### 1. Structured edits only

We **never blindly rewrite files**.

All edits are:

- orchestrated in the core
- anchored
- validated
- diffed before apply

The current implementation intentionally supports a small deterministic slice instead of pretending the agent covers all edit styles.

#### 2. Core-first architecture

All intelligence lives in `vocode-cored`.

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
- ✅ Core (`vocode-cored`) spawns
- ✅ Cross-platform core build
- ✅ JSON-RPC over stdio (`vocode-cored` + extension)
- 🚧 Rich edit engine wiring beyond the initial safe slice
- 🚧 Voice pipeline

---

### 📦 Core binary build

`vocode-cored` is built per platform under `apps/core`:

```
apps/core/bin/
  win32-x64/vocode-cored.exe
  darwin-arm64/vocode-cored
  linux-x64/vocode-cored
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

Supported today: deterministic single-file edits for `insert statement "..." inside current function`, `replace block after "..." before "..." with "..."`, and `append import "..." if missing`. The core returns explicit `success`/`failure`/`noop` edit outcomes so the extension can display intent-preserving UX without another agent turn.

---

### 🧱 Roadmap (Short-Term)

- [ ] JSON-RPC over stdio (in progress)
- [ ] polish RPC client surface (`src/daemon/*` naming vs `vocode-cored`)
- [ ] workspace sync
- [ ] edit intents → applier
- [ ] diff UI panel
- [ ] streaming speech input

---

### 🧑‍💻 Contributing

See `CONTRIBUTING.md` for more information

1. Install deps: `pnpm install`
2. Generate protocol types: `pnpm codegen`
3. Build core: `pnpm --filter @vocode/core build`
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
- Core logs go to stderr
- Core stdout is reserved for JSON-RPC

---

### 📄 License

TBD

---

### 🧠 Vision

> Speak code.
> Watch it evolve.
> Stay in the flow.

<p align="right"><a href="#readme-top">back to top</a></p>
