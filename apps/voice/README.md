# Voice Sidecar (`apps/voice`)

`apps/voice` is a dedicated process for voice I/O concerns (microphone capture
and speech-to-text orchestration), intentionally separate from:

- `apps/daemon` (planning + semantic policy + action-plan dispatch)
- `apps/vscode-extension` (UI + process orchestration + editor mechanics)

## Purpose

This sidecar is the place to implement:

- cross-platform microphone capture
- audio buffering/chunking
- STT integrations (cloud/local)
- transcript event emission back to the extension

It should not contain planning/action-plan logic.

## Native Dependencies

Native microphone capture is implemented with Go's `cgo` bindings to PortAudio.

To enable real mic capture:
- Ensure `CGO_ENABLED=1`
- Install PortAudio *and* `pkg-config` support for `portaudio-2.0`

## STT toggle

To disable ElevenLabs calls (and manually send transcripts via the VS Code command
palette), set:
- `VOCODE_VOICE_STT_ENABLED=false`

## STT model + tuning

STT uses ElevenLabs realtime websocket transcription with local VAD, `audio_meter` events for the VS Code panel, and optional `[vocode-vad]` traces on stderr.

Core model selection:
- `ELEVENLABS_STT_MODEL_ID` (default: `scribe_v2`)

Streaming VAD/segmentation knobs:
- `VOCODE_VOICE_VAD_THRESHOLD_MULTIPLIER` (default: `2.0`)
- `VOCODE_VOICE_VAD_START_MS` (default: `60`)
- `VOCODE_VOICE_VAD_END_MS` (default: `500`)
- `VOCODE_VOICE_VAD_PREROLL_MS` (default: `200`)
- `VOCODE_VOICE_STREAM_MIN_CHUNK_MS` (default: `200`)
- `VOCODE_VOICE_STREAM_MAX_CHUNK_MS` (default: `500`)
- `VOCODE_VOICE_STREAM_MAX_UTTERANCE_MS` (default: `0` = off; optional periodic commit cap during continuous speech, min `500` when set)

## Rollout / tuning checklist

1. Start with defaults and `ELEVENLABS_STT_MODEL_ID=scribe_v2`.
2. Validate transcript quality on real speech with pauses and interruptions.
3. If too many premature commits, increase `VOCODE_VOICE_VAD_END_MS`.
4. If speech starts are clipped, increase `VOCODE_VOICE_VAD_PREROLL_MS`.
5. If latency feels high during speech bursts, lower `VOCODE_VOICE_STREAM_MIN_CHUNK_MS`.
6. If chunk churn is high, raise `VOCODE_VOICE_STREAM_MAX_CHUNK_MS`.
7. For long spoken explanations without mid-sentence commits, leave `VOCODE_VOICE_STREAM_MAX_UTTERANCE_MS` unset or `0`; set e.g. `8000` only if you want forced segment cuts during very long continuous speech.

## Debugging VAD

- **Never log debug output to stdout** in the sidecar — stdout is JSON lines for the extension. Use **stderr** only.
- **Runtime traces:** set `VOCODE_VOICE_VAD_DEBUG=1` (or `true`) and restart the sidecar. You’ll see lines like `[vocode-vad] speech_start …`, `commit utterance_end (silence) …`, `commit utterance_max …`, and `commit flush …` on **stderr**. The VS Code extension forwards sidecar stderr to the host log as `[vocode-voiced stderr] …` (e.g. **Developer: Toggle Developer Tools** → Console). You can also run `vocode-voiced` from a terminal with the same env to read stderr directly.
- **Extension:** the repo root `.env` is **not** loaded into the extension host automatically. The extension **merges workspace `.env` into the spawned sidecar’s environment**, so `VOCODE_VOICE_VAD_DEBUG` there applies after reload. Alternatively enable **Settings → Vocode: Voice Vad Debug** (`vocode.voiceVadDebug`) to force `VOCODE_VOICE_VAD_DEBUG=1` without editing `.env`. On activation, the extension logs `[vocode] voice sidecar spawn env: VOCODE_VOICE_VAD_DEBUG=…` to the console so you can confirm what the child process received.
- **Unit tests:** `go test ./internal/app/... -run VAD -v` exercises commit behavior with synthetic PCM frames (`vad_test.go`). Adjust `t.Setenv` in a scratch test to reproduce your thresholds.
- **Tuning:** if commits never fire, try **lowering** `VOCODE_VOICE_VAD_END_MS` (faster “end of sentence”) or **raising** `VOCODE_VOICE_VAD_THRESHOLD_MULTIPLIER` if noise keeps you “in speech”. If speech is clipped at the start, raise `VOCODE_VOICE_VAD_PREROLL_MS` or `VOCODE_VOICE_VAD_START_MS`.

### Linux (Ubuntu/Debian)

Install native deps:

```bash
sudo apt-get update
sudo apt-get install -y pkg-config portaudio19-dev
```

Then build:

```bash
pnpm --filter @vocode/voice build
```

### Windows (Manual Setup)

1. Install MSYS2 via Chocolatey:
   - `choco install msys2`
2. Open the **MSYS2 MinGW x64** shell (usually `mingw64.exe`).
3. Install PortAudio + pkg-config (runs once per machine):
   - `pacman -Syu`
   - `pacman -S --needed mingw-w64-x86_64-gcc mingw-w64-x86_64-portaudio mingw-w64-x86_64-pkg-config`
4. Verify (run from inside the **mingw64** shell):
   - `pkg-config --modversion portaudio-2.0`
5. Build the voice sidecar:
   - `pnpm --filter @vocode/voice build`

### Windows (Automated Setup)

You can run a single PowerShell script that installs the
required MSYS2/MinGW packages and verifies `pkg-config` can find PortAudio:

```powershell
pnpm setup-portaudio:win
```

If your MSYS2 install is not at `C:\tools\msys64`, run the PowerShell script
directly and pass a different root:

```powershell
pnpm setup-portaudio:win "D:\msys64"
```

When building from PowerShell, `@vocode/voice`’s build script will try to
auto-configure `cgo` for PortAudio using your MSYS2 installation:
- Default MSYS2 root: `C:\tools\msys64`
- Override with `MSYS2_ROOT` if your MSYS2 is installed elsewhere

## Transport

The initial skeleton uses JSON lines over stdio:

- Extension writes requests to sidecar stdin.
- Sidecar writes events/responses to stdout.

Current request/event shapes are defined in `internal/app`.

### `audio_meter`

While the transcription loop is running, the sidecar emits throttled level + VAD snapshots for UI:

```json
{"type":"audio_meter","speaking":true,"rms":0.35}
```

- **`speaking`**: local VAD considers the current utterance “in speech” (same gate used for chunking).
- **`rms`**: normalized **0–1** from PCM16 frame RMS (20 ms frames); heuristic gain, not dBFS.

## Binary

The sidecar command entrypoint is:

- `apps/voice/cmd/vocode-voiced`
