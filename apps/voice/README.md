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
pwsh -ExecutionPolicy Bypass -File scripts/dev/setup-portaudio.ps1
```

Optional: if your MSYS2 install is not at `C:\tools\msys64`, pass a different
root:

```powershell
pwsh -ExecutionPolicy Bypass -File scripts/dev/setup-portaudio.ps1 -Msys2Root "D:\msys64"
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

## Binary

The sidecar command entrypoint is:

- `apps/voice/cmd/vocode-voiced`
