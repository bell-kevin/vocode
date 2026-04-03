# VS Code settings → environment variables

User-facing descriptions for `vocode.*` settings live in `apps/vscode-extension/package.json` (they appear in the VS Code Settings UI). This page is for **contributors and debugging**: how those values reach child processes and the RPC layer.

## Spawned core and voice processes

When the extension starts `vocode-cored` and `vocode-voiced`, it builds `process.env` in `apps/vscode-extension/src/config/spawn-env.ts` (`applyVocodeSpawnEnvironment`).

### API keys (Secret Storage, not `settings.json`)

| Secret (VS Code)              | Environment variable   |
| ----------------------------- | ------------------------ |
| `vocode.elevenLabsApiKey`     | `ELEVENLABS_API_KEY`    |
| `vocode.openaiApiKey`         | `OPENAI_API_KEY`       |
| `vocode.anthropicApiKey`      | `ANTHROPIC_API_KEY`    |

### `vocode.*` settings → env (see `CONFIG_TO_ENV` in `spawn-env.ts`)

| VS Code setting                             | Environment variable                            |
| ------------------------------------------- | ----------------------------------------------- |
| `vocode.daemonAgentProvider`                | `VOCODE_AGENT_PROVIDER`                         |
| `vocode.daemonOpenaiModel`                  | `VOCODE_OPENAI_MODEL`                           |
| `vocode.daemonOpenaiBaseUrl`                | `VOCODE_OPENAI_BASE_URL`                        |
| `vocode.daemonAnthropicModel`               | `VOCODE_ANTHROPIC_MODEL`                        |
| `vocode.daemonAnthropicBaseUrl`             | `VOCODE_ANTHROPIC_BASE_URL`                     |
| `vocode.daemonVoiceLogTranscript`           | `VOCODE_DAEMON_VOICE_LOG_TRANSCRIPT`            |
| `vocode.elevenLabsSttLanguage`              | `ELEVENLABS_STT_LANGUAGE`                       |
| `vocode.elevenLabsSttModelId`               | `ELEVENLABS_STT_MODEL_ID`                       |
| `vocode.voiceSttCommitResponseTimeoutMs`    | `VOCODE_VOICE_STT_COMMIT_RESPONSE_TIMEOUT_MS`   |
| `vocode.voiceVadThresholdMultiplier`        | `VOCODE_VOICE_VAD_THRESHOLD_MULTIPLIER`         |
| `vocode.voiceVadMinEnergyFloor`             | `VOCODE_VOICE_VAD_MIN_ENERGY_FLOOR`             |
| `vocode.voiceVadStartMs`                    | `VOCODE_VOICE_VAD_START_MS`                     |
| `vocode.voiceVadEndMs`                      | `VOCODE_VOICE_VAD_END_MS`                       |
| `vocode.voiceVadPrerollMs`                  | `VOCODE_VOICE_VAD_PREROLL_MS`                   |
| `vocode.voiceStreamMinChunkMs`              | `VOCODE_VOICE_STREAM_MIN_CHUNK_MS`              |
| `vocode.voiceStreamMaxChunkMs`              | `VOCODE_VOICE_STREAM_MAX_CHUNK_MS`              |
| `vocode.voiceStreamMaxUtteranceMs`          | `VOCODE_VOICE_STREAM_MAX_UTTERANCE_MS`          |
| `vocode.daemonVoiceTranscriptQueueSize`     | `VOCODE_DAEMON_VOICE_TRANSCRIPT_QUEUE_SIZE`     |
| `vocode.daemonVoiceTranscriptCoalesceMs`    | `VOCODE_DAEMON_VOICE_TRANSCRIPT_COALESCE_MS`    |
| `vocode.daemonVoiceTranscriptMaxMergeJobs`  | `VOCODE_DAEMON_VOICE_TRANSCRIPT_MAX_MERGE_JOBS` |
| `vocode.daemonVoiceTranscriptMaxMergeChars` | `VOCODE_DAEMON_VOICE_TRANSCRIPT_MAX_MERGE_CHARS`|

### Other env set from settings / workspace

| Source | Environment variable | Notes |
| ------ | ---------------------- | ----- |
| `vocode.voiceVadDebug` === true | `VOCODE_VOICE_VAD_DEBUG` = `1` | Omitted when false. |
| Workspace `.vocode` STT keywords | `VOCODE_STT_KEYTERMS_JSON` | JSON string array; see `workspace-vocode.ts`. |

The extension does **not** load a workspace `.env` for spawned children; use shell exports when running `go run` / binaries manually.

## Not passed as spawn env

These settings are still in `package.json` but are consumed elsewhere:

| Setting | Where it is used |
| ------- | ---------------- |
| `vocode.voiceSidecarLogProtocol` | Extension only: whether to log voice JSON protocol to Developer Tools (`voice/spawn.ts`). |
| `vocode.sessionIdleResetMs` | Passed per request as `daemonConfig.sessionIdleResetMs` on `voice.transcript` (`run-daemon-transcript.ts`), not as a global core env var. |

Live tuning from the panel uses `PANEL_CONFIG_KEYS` in `spawn-env.ts`; some keys overlap the table above, others are forwarded over stdin to the voice sidecar (see `VoiceSidecarClient.setConfig` / voice app config patch).

## Keeping docs in sync

When you add or rename a `vocode.*` setting that maps to a process:

1. Add it to `CONFIG_TO_ENV` (or the special cases) in `spawn-env.ts`.
2. Update this table in the same PR.

The **`CONFIG_TO_ENV` array is the source of truth** for the spawn mapping; this file is a human-readable mirror.
