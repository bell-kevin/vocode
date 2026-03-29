import * as vscode from "vscode";

import { applyWorkspaceDotEnv } from "../voice/workspace-env";

/** VS Code SecretStorage key (never log). */
export const ELEVENLABS_API_KEY_SECRET = "vocode.elevenLabsApiKey";

function isExplicitlySet(
  ins:
    | {
        readonly globalValue?: unknown;
        readonly workspaceValue?: unknown;
        readonly workspaceFolderValue?: unknown;
      }
    | undefined,
): boolean {
  if (!ins) {
    return false;
  }
  return (
    ins.globalValue !== undefined ||
    ins.workspaceValue !== undefined ||
    ins.workspaceFolderValue !== undefined
  );
}

type ConfigBinding =
  | { configKey: string; envVar: string; kind: "string" }
  | { configKey: string; envVar: string; kind: "number" }
  | { configKey: string; envVar: string; kind: "float" };

const CONFIG_TO_ENV: readonly ConfigBinding[] = [
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
    configKey: "daemonVoiceMaxAgentTurns",
    envVar: "VOCODE_DAEMON_VOICE_MAX_AGENT_TURNS",
    kind: "number",
  },
  {
    configKey: "daemonVoiceMaxIntentRetries",
    envVar: "VOCODE_DAEMON_VOICE_MAX_INTENT_RETRIES",
    kind: "number",
  },
  {
    configKey: "daemonVoiceMaxContextRounds",
    envVar: "VOCODE_DAEMON_VOICE_MAX_CONTEXT_ROUNDS",
    kind: "number",
  },
  {
    configKey: "daemonVoiceMaxContextBytes",
    envVar: "VOCODE_DAEMON_VOICE_MAX_CONTEXT_BYTES",
    kind: "number",
  },
  {
    configKey: "daemonVoiceMaxConsecutiveContextRequests",
    envVar: "VOCODE_DAEMON_VOICE_MAX_CONSECUTIVE_CONTEXT_REQUESTS",
    kind: "number",
  },
  {
    configKey: "daemonSessionIdleResetMs",
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

function applyBinding(
  config: vscode.WorkspaceConfiguration,
  env: NodeJS.ProcessEnv,
  b: ConfigBinding,
): void {
  const ins = config.inspect(b.configKey);
  if (!isExplicitlySet(ins)) {
    return;
  }
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
 * Merges workspace `.env` (callers should run `applyWorkspaceDotEnv` first), then
 * VS Code settings (only keys the user explicitly set), then the ElevenLabs API
 * key from SecretStorage when present.
 */
export async function applyVocodeSpawnEnvironment(
  context: vscode.ExtensionContext,
  env: NodeJS.ProcessEnv,
): Promise<void> {
  const apiKey = await context.secrets.get(ELEVENLABS_API_KEY_SECRET);
  if (apiKey !== undefined && apiKey.trim() !== "") {
    env.ELEVENLABS_API_KEY = apiKey.trim();
  }

  const config = vscode.workspace.getConfiguration("vocode");
  for (const b of CONFIG_TO_ENV) {
    applyBinding(config, env, b);
  }

  const vadDbg = config.inspect("voiceVadDebug");
  if (isExplicitlySet(vadDbg)) {
    if (config.get<boolean>("voiceVadDebug") === true) {
      env.VOCODE_VOICE_VAD_DEBUG = "1";
    } else {
      delete env.VOCODE_VOICE_VAD_DEBUG;
    }
  }
}

export async function elevenLabsApiKeyIsConfigured(
  context: vscode.ExtensionContext,
): Promise<boolean> {
  const secret = await context.secrets.get(ELEVENLABS_API_KEY_SECRET);
  if (secret !== undefined && secret.trim() !== "") {
    return true;
  }
  const w = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
  if (w) {
    const env = { ...process.env };
    applyWorkspaceDotEnv(env, w);
    if (env.ELEVENLABS_API_KEY?.trim()) {
      return true;
    }
  }
  return !!process.env.ELEVENLABS_API_KEY?.trim();
}
