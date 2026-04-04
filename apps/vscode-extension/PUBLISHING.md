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

This produces `apps/vscode-extension/vocode-0.0.1.vsix`. Install locally with **Extensions: Install from VSIX…**.

Staging does **not** copy the core binary from `apps/core` — it must already exist under `apps/vscode-extension/bin/` from the core build step in prepublish. It copies **ripgrep**, merges **voice** if present, and embeds `@vocode/protocol` under `dist/protocol-pkg` (rewrites `dist/daemon/client.js`). Run `pnpm build` again after packaging if you want an unpatched `client.js` for local dev.

## Publish

Bump `version` in `package.json`, then:

```bash
cd apps/vscode-extension
pnpm exec vsce publish --no-dependencies --follow-symlinks
```

(`publish` runs `vscode:prepublish` the same way `package` does.)

`vsce publish` uses the PAT from `vsce login`.

## Multi-OS binaries

A single VSIX built on Windows only contains `win32-x64` (or your arch) under `bin/` and `tools/`. Users on macOS/Linux need builds that include **their** triples, or you merge multiple platforms into one VSIX (custom pipeline: cross-compile Go, build voice per OS, copy each under `bin/<platform>-<arch>/`). The staging script only copies the **host** triple; extending it for a release matrix is a follow-up.

## Notes

- Extension package **name** must be unscoped (`vocode`); scoped npm names are rejected by `vsce`.
- `@vocode/protocol` is a **devDependency** for TypeScript; the staging script copies its `dist` into `dist/protocol-pkg` for the packaged extension.
