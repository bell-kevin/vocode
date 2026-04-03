# vocode-cored verification and debugging

Use this when validating the extension + `vocode-cored` + voice stack after changes.

## Baseline (automated)

From repo root:

```bash
pnpm codegen
pnpm lint
pnpm --filter @vocode/core build
go test $(go list ./... | grep -vE '/apps/voice($|/)' | grep -vE '/apps/daemon($|/)')
```

On Windows PowerShell, run `go test ./apps/core/...` (and other packages as needed) if the `go list` pipeline is awkward.

## Extension Development Host (F5)

1. Open the repo in VS Code; **Run and Debug** → launch the Vocode extension.
2. **Debug Console:** Extension logs; `[vocode-cored stderr]` lines from `vocode-cored`.
3. **Core transcript logging:** set `VOCODE_DAEMON_VOICE_LOG_TRANSCRIPT=1` in spawn env (see [AGENTS.md](../AGENTS.md)) to correlate utterances with routing.

### Manual matrix (once per milestone)

Run through these in the Extension Development Host after `pnpm codegen`, `pnpm --filter @vocode/core build`, and `go test ./apps/core/...`:

| Check | Pass criteria |
| ----- | ------------- |
| Startup | Extension activates; sidebar/status usable; no spawn error toast. |
| Send Transcript → deterministic edit | README-style edit phrases apply; undo ledger behaves. |
| `workspace_select` | Content search opens first hit; selection expands to smallest LSP symbol containing the rg anchor when the hit file is the active file and symbols were sent. |
| `select_file` | Path/filename fragment (e.g. `test.js`) lists files even when file bodies do not contain that text. |
| Ripgrep | `VOCODE_RG_BIN` set when bundled rg exists (`tools/ripgrep/...`); otherwise `rg` on PATH works. |

## Minimal repros to capture

For each issue, record:

- Exact utterance (or **Vocode: Send Transcript** text).
- Workspace layout (single-root / multi-root, file open).
- Full `voice.transcript` JSON result (Extension Host Debug Console or RPC log).
- Any `vocode-cored` stderr line.

### Apply / `host.applyDirectives`

- Symptom: edit or navigate batch fails.
- Trace: extension [`apply-directives.ts`](../apps/vscode-extension/src/voice-transcript/apply-directives.ts) → [`dispatch.ts`](../apps/vscode-extension/src/directives/dispatch.ts) → specific dispatcher; core errors often mention `host.applyDirectives` or `host apply failed`.

### Workspace content search (`workspace_select`)

- Symptom: no hits, wrong file, or navigation fails after hits.
- Check `VOCODE_RG_BIN` / `rg` on PATH; workspace root in params.
- Windows: compare rg output paths to `editor.document.uri.fsPath`.

### File path search (`select_file`)

- Symptom: `fileSelection.noHits` while a matching path exists, or sidebar list empty.
- Confirm completion includes `fileSelection.results` and `activeIndex` (protocol validator requires both when `results` is non-empty).

## Related code

- Core RPC: [`apps/core/internal/rpc/handlers.go`](../apps/core/internal/rpc/handlers.go)
- Root flow dispatch: [`apps/core/internal/flows/root/dispatch.go`](../apps/core/internal/flows/root/dispatch.go)
- File path search: [`apps/core/internal/transcript/searchapply/file_path_search.go`](../apps/core/internal/transcript/searchapply/file_path_search.go)
- Extension transcript runner: [`apps/vscode-extension/src/voice-transcript/run-daemon-transcript.ts`](../apps/vscode-extension/src/voice-transcript/run-daemon-transcript.ts)
