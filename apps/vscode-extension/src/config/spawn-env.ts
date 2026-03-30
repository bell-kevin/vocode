import * as vscode from "vscode";

import { readWorkspaceSttKeywords } from "./workspace-vocode";

/** VS Code SecretStorage key (never log). */
export const ELEVENLABS_API_KEY_SECRET = "vocode.elevenLabsApiKey";

type ConfigBinding =
  | { configKey: string; envVar: string; kind: "string" }
  | { configKey: string; envVar: string; kind: "number" }
  | { configKey: string; envVar: string; kind: "float" };

const CONFIG_TO_ENV: readonly ConfigBinding[] = [
  {
    configKey: "daemonAgentProvider",
    envVar: "VOCODE_AGENT_PROVIDER",
    kind: "string",
  },
  {
    configKey: "daemonOpenaiModel",
    envVar: "VOCODE_OPENAI_MODEL",
    kind: "string",
  },
  {
    configKey: "daemonOpenaiBaseUrl",
    envVar: "VOCODE_OPENAI_BASE_URL",
    kind: "string",
  },
  {
    configKey: "daemonAnthropicModel",
    envVar: "VOCODE_ANTHROPIC_MODEL",
    kind: "string",
  },
  {
    configKey: "daemonAnthropicBaseUrl",
    envVar: "VOCODE_ANTHROPIC_BASE_URL",
    kind: "string",
  },
  {
    configKey: "daemonVoiceLogTranscript",
    envVar: "VOCODE_DAEMON_VOICE_LOG_TRANSCRIPT",
    kind: "number",
  },
  {
    configKey: "elevenLabsSttLanguage",
    envVar: "ELEVENLABS_STT_LANGUAGE",
    kind: "string",
  },
  {
    configKey: "elevenLabsSttModelId",
    envVar: "ELEVENLABS_STT_MODEL_ID",
    kind: "string",
  },
  {
    configKey: "voiceSttCommitResponseTimeoutMs",
    envVar: "VOCODE_VOICE_STT_COMMIT_RESPONSE_TIMEOUT_MS",
    kind: "number",
  },
  {
    configKey: "voiceVadThresholdMultiplier",
    envVar: "VOCODE_VOICE_VAD_THRESHOLD_MULTIPLIER",
    kind: "float",
  },
  {
    configKey: "voiceVadMinEnergyFloor",
    envVar: "VOCODE_VOICE_VAD_MIN_ENERGY_FLOOR",
    kind: "float",
  },
  {
    configKey: "voiceVadStartMs",
    envVar: "VOCODE_VOICE_VAD_START_MS",
    kind: "number",
  },
  {
    configKey: "voiceVadEndMs",
    envVar: "VOCODE_VOICE_VAD_END_MS",
    kind: "number",
  },
  {
    configKey: "voiceVadPrerollMs",
    envVar: "VOCODE_VOICE_VAD_PREROLL_MS",
    kind: "number",
  },
  {
    configKey: "voiceStreamMinChunkMs",
    envVar: "VOCODE_VOICE_STREAM_MIN_CHUNK_MS",
    kind: "number",
  },
  {
    configKey: "voiceStreamMaxChunkMs",
    envVar: "VOCODE_VOICE_STREAM_MAX_CHUNK_MS",
    kind: "number",
  },
  {
    configKey: "voiceStreamMaxUtteranceMs",
    envVar: "VOCODE_VOICE_STREAM_MAX_UTTERANCE_MS",
    kind: "number",
  },
  {
    configKey: "daemonVoiceTranscriptQueueSize",
    envVar: "VOCODE_DAEMON_VOICE_TRANSCRIPT_QUEUE_SIZE",
    kind: "number",
  },
  {
    configKey: "daemonVoiceTranscriptCoalesceMs",
    envVar: "VOCODE_DAEMON_VOICE_TRANSCRIPT_COALESCE_MS",
    kind: "number",
  },
  {
    configKey: "daemonVoiceTranscriptMaxMergeJobs",
    envVar: "VOCODE_DAEMON_VOICE_TRANSCRIPT_MAX_MERGE_JOBS",
    kind: "number",
  },
  {
    configKey: "daemonVoiceTranscriptMaxMergeChars",
    envVar: "VOCODE_DAEMON_VOICE_TRANSCRIPT_MAX_MERGE_CHARS",
    kind: "number",
  },
  {
    configKey: "maxPlannerTurns",
    envVar: "VOCODE_DAEMON_VOICE_MAX_AGENT_TURNS",
    kind: "number",
  },
  {
    configKey: "maxTranscriptRepairRpcs",
    envVar: "VOCODE_DAEMON_VOICE_MAX_REPAIR_RPCS",
    kind: "number",
  },
  {
    configKey: "maxIntentsPerBatch",
    envVar: "VOCODE_DAEMON_VOICE_MAX_INTENTS_PER_BATCH",
    kind: "number",
  },
  {
    configKey: "maxIntentDispatchRetries",
    envVar: "VOCODE_DAEMON_VOICE_MAX_INTENT_RETRIES",
    kind: "number",
  },
  {
    configKey: "maxContextRounds",
    envVar: "VOCODE_DAEMON_VOICE_MAX_CONTEXT_ROUNDS",
    kind: "number",
  },
  {
    configKey: "maxContextBytes",
    envVar: "VOCODE_DAEMON_VOICE_MAX_CONTEXT_BYTES",
    kind: "number",
  },
  {
    configKey: "maxConsecutiveContextRequests",
    envVar: "VOCODE_DAEMON_VOICE_MAX_CONSECUTIVE_CONTEXT_REQUESTS",
    kind: "number",
  },
  {
    configKey: "sessionIdleResetMs",
    envVar: "VOCODE_DAEMON_SESSION_IDLE_RESET_MS",
    kind: "number",
  },
] as const;

export const PANEL_CONFIG_KEYS = [
  "voiceVadDebug",
  "voiceSidecarLogProtocol",
  ...CONFIG_TO_ENV.map((b) => b.configKey),
] as const;

export type PanelConfigKey = (typeof PANEL_CONFIG_KEYS)[number];

/**
 * Writes effective VS Code `vocode.*` values into `env` for child processes.
 * Uses `getConfiguration().get()` so **defaults come from package.json** and
 * **user/workspace overrides** apply automatically.
 */
function applyBinding(
  config: vscode.WorkspaceConfiguration,
  env: NodeJS.ProcessEnv,
  b: ConfigBinding,
): void {
  if (b.kind === "string") {
    const v = config.get<string>(b.configKey);
    if (typeof v === "string") {
      env[b.envVar] = v;
    }
    return;
  }
  if (b.kind === "number") {
    const v = config.get<number>(b.configKey);
    if (typeof v === "number" && Number.isFinite(v)) {
      env[b.envVar] = String(Math.trunc(v));
    }
    return;
  }
  const v = config.get<number>(b.configKey);
  if (typeof v === "number" && Number.isFinite(v)) {
    env[b.envVar] = String(v);
  }
}

/**
 * Two layers only for the extension:
 * 1. **Defaults** — `package.json` `contributes.configuration` defaults (via `get()`).
 * 2. **User** — VS Code user/workspace settings + panel edits, and the ElevenLabs API
 *    key from **SecretStorage** (not settings.json).
 *
 * Workspace `.env` is **not** read when spawning daemon/voice from the extension.
 * For terminal / `go run`, export variables in the shell (no repo `.env` template).
 */
export async function applyVocodeSpawnEnvironment(
  context: vscode.ExtensionContext,
  env: NodeJS.ProcessEnv,
): Promise<void> {
  const apiKey = await context.secrets.get(ELEVENLABS_API_KEY_SECRET);
  if (apiKey !== undefined && apiKey.trim() !== "") {
    env.ELEVENLABS_API_KEY = apiKey.trim();
  } else {
    delete env.ELEVENLABS_API_KEY;
  }

  const config = vscode.workspace.getConfiguration("vocode");
  const capConfigKeysToSkipEnv = new Set<PanelConfigKey>([
    "maxPlannerTurns",
    "maxIntentsPerBatch",
    "maxIntentDispatchRetries",
    "maxContextRounds",
    "maxContextBytes",
    "maxConsecutiveContextRequests",
    "maxTranscriptRepairRpcs",
    "sessionIdleResetMs",
  ]);
  for (const b of CONFIG_TO_ENV) {
    // Daemon consumes these caps from `voice.transcript` params (`daemonConfig`),
    // so we don't need env updates (no restart required).
    if (capConfigKeysToSkipEnv.has(b.configKey as PanelConfigKey)) {
      continue;
    }
    applyBinding(config, env, b);
  }

  if (config.get<boolean>("voiceVadDebug") === true) {
    env.VOCODE_VOICE_VAD_DEBUG = "1";
  } else {
    delete env.VOCODE_VOICE_VAD_DEBUG;
  }

  const sttKeywords = await readWorkspaceSttKeywords();
  if (sttKeywords.length > 0) {
    env.VOCODE_STT_KEYTERMS_JSON = JSON.stringify(sttKeywords);
  } else {
    delete env.VOCODE_STT_KEYTERMS_JSON;
  }
}

/** True when the user has stored an API key in SecretStorage (extension path). */
export async function elevenLabsApiKeyIsConfigured(
  context: vscode.ExtensionContext,
): Promise<boolean> {
  const secret = await context.secrets.get(ELEVENLABS_API_KEY_SECRET);
  return secret !== undefined && secret.trim() !== "";
}
