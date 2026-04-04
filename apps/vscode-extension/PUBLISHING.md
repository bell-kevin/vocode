# Publishing Vocode to the VS Code Marketplace

## One-time setup

1. **Publisher** — In [Azure DevOps](https://dev.azure.com) (linked to Microsoft), create an organization if needed, then create a [Personal Access Token](https://learn.microsoft.com/azure/devops/organizations/accounts/use-personal-access-tokens-to-authenticate) with scope **Marketplace → Manage**.
2. **Marketplace publisher** — On [Visual Studio Marketplace](https://marketplace.visualstudio.com/manage), create a publisher. Set `publisher` in `apps/vscode-extension/package.json` to that **exact** id (must match the account you upload under, e.g. `SpencerLS`).
3. **Login** (from `apps/vscode-extension`):

   ```bash
   pnpm exec vsce login <your-publisher-id>
   ```

## Build a `.vsix` (current machine only)

The VSIX must include **native binaries** for the OS you build on (`vocode-cored`, optional `vocode-voiced`, and `ripgrep`) under **one** `platform-arch` triple.

`pnpm vscode:package` runs `vscode:prepublish`, which builds **protocol**, **core** (writes `vocode-cored` to `apps/vscode-extension/bin/<triple>/`), the **extension** (TypeScript + webview), then **staging** (merges voice if built, copies ripgrep, embeds protocol).

From the **repository root** (install + ripgrep once; optional voice before packaging):

```bash
pnpm install
pnpm codegen
pnpm provision:ripgrep
# Optional: pnpm --filter @vocode/voice build
pnpm vscode:package
```

This writes `apps/vscode-extension/release/vocode-<version>.vsix` (not the package root). Install locally with **Extensions: Install from VSIX…**.

Staging does **not** copy the core binary from `apps/core` — it must already exist under `apps/vscode-extension/bin/` from the core build step in prepublish. It copies **ripgrep**, merges **voice** if present, and embeds `@vocode/protocol` under `dist/protocol-pkg` (rewrites `dist/daemon/client.js`). Run `pnpm build` again after packaging if you want an unpatched `client.js` for local dev.

## Publish

Bump `version` in `package.json`, then:

```bash
cd apps/vscode-extension
pnpm exec vsce publish --no-dependencies --follow-symlinks
```

(`publish` runs `vscode:prepublish` the same way `package` does.)

`vsce publish` uses the PAT from `vsce login`.

## Voice in the same VSIX

You do **not** ship a separate VSIX for voice. `pnpm --filter @vocode/voice build` compiles **vocode-voiced** with **CGO + PortAudio** for the current OS/arch (see `scripts/dev/voice-build-lib.mjs`). Staging merges that binary into `apps/vscode-extension/bin/<triple>/` next to `vocode-cored`.

## One fat VSIX (all Marketplace platforms)

From the **repository root**:

```bash
pnpm install
pnpm codegen
pnpm vscode:package:fat
```

This cross-builds **vocode-cored**, runs **`node scripts/dev/build-voice-cross.mjs`** (PortAudio / CGO for every slug), downloads **ripgrep** per platform, builds the extension, then stages **all** `bin/<slug>/` and `tools/ripgrep/<slug>/` folders. Output: `apps/vscode-extension/release/vocode-<version>.vsix`.

**Voice (PortAudio):** `build-voice-cross.mjs` uses **native** MSYS2/Homebrew builds on Windows/macOS for the host’s `win32-*` / `darwin-*` triple, and **Docker** (`golang:*-bookworm` / `*-alpine`) for **linux-*** and **alpine-*** with `portaudio19-dev` / `portaudio-dev`. You need [Docker Desktop](https://www.docker.com/products/docker-desktop/) (or Docker Engine) for those Linux targets when not on a matching Linux host. Slugs you cannot build locally (e.g. **darwin-*** on Windows) are **skipped only if** `apps/voice/bin/<slug>/` already contains a binary—populate those from CI (see `.github/workflows/ci.yml` jobs `voice-darwin-*`, `voice-windows-arm64`) or copy from another machine, then re-run fat packaging.

Native artifacts under `apps/voice/bin/`, `apps/vscode-extension/bin/`, `tools/ripgrep/`, and staged `apps/vscode-extension/tools/ripgrep/` are **gitignored** — do not commit them.

## Does CI build all voice (PortAudio) binaries?

Yes, **together** the voice jobs in `.github/workflows/ci.yml` produce every VSIX slug:

| Job | What it builds |
|-----|------------------|
| `voice-linux` | `linux-*`, `alpine-*` (Docker on Ubuntu) |
| `voice-windows` | `win32-x64` |
| `voice-windows-arm64` | `win32-arm64` |
| `voice-darwin-x64` | `darwin-x64` (`macos-15-intel`) |
| `voice-darwin-arm64` | `darwin-arm64` |

Each job **uploads** an artifact named `voice-partial-*`. The **`voice-binaries-bundle`** job downloads those, merges them into one tree, and uploads **`voice-binaries-all`** (complete `apps/voice/bin/` layout).

### Download merged voice binaries (for local fat VSIX)

1. Install the [GitHub CLI](https://cli.github.com/) (`gh`) and run `gh auth login`.
2. Find a successful workflow run (push/PR to `main`/`master` after your changes):

   ```bash
   gh run list --workflow ci.yml --limit 5
   ```

3. **Recommended:** download the single merged artifact CI produces:

   ```bash
   gh run download <run-id> -n voice-binaries-all -D ./voice-dl
   ```

4. Copy into your clone (open `./voice-dl` and match what you see—often `apps/voice/bin/<slug>/`):

   ```bash
   mkdir -p apps/voice/bin
   cp -a ./voice-dl/apps/voice/bin/* apps/voice/bin/
   ```

   If slug folders sit at the top of the extract instead, copy those directories into `apps/voice/bin/`.

5. **Fallback** (only if `voice-binaries-all` is missing—e.g. `voice-binaries-bundle` failed or a job was skipped): download every `voice-partial-*` artifact and merge with the script:

   ```bash
   gh run download <run-id> -p 'voice-partial-*' -D ./voice-partials
   node scripts/dev/merge-voice-partial-artifacts.mjs ./voice-partials
   ```

6. Then run `pnpm vscode:package:fat` (or only `node scripts/dev/build-voice-cross.mjs` if you only needed missing slugs). The fat prepublish will **keep** existing `apps/voice/bin/*` binaries when it cannot rebuild a slug on your machine.

**UI:** On GitHub → **Actions** → select the workflow run → scroll to **Artifacts** → download `voice-binaries-all`.

## Multi-OS: platform-specific VSIXes (optional)

`pnpm vscode:package` still builds a **single-triple** VSIX for the machine you run on.

Alternatively, publish per-platform VSIXes with `vsce publish --target win32-x64` (and other `--target` values) so the Marketplace serves the right file per client ([docs](https://code.visualstudio.com/api/working-with-extensions/publishing-extension#platform-specific-extensions)).

## Notes

- Extension package **name** must be unscoped (`vocode`); scoped npm names are rejected by `vsce`.
- `@vocode/protocol` is a **devDependency** for TypeScript; the staging script copies its `dist` into `dist/protocol-pkg` for the packaged extension.
