import * as vscode from "vscode";

import {
  anthropicApiKeyIsConfigured,
  elevenLabsApiKeyIsConfigured,
  openaiApiKeyIsConfigured,
  PANEL_CONFIG_KEYS,
  type PanelConfigKey,
} from "./spawn-env";

export type VocodePanelConfigMessage = {
  type: "panelConfig";
  elevenLabsApiKeyConfigured: boolean;
  openaiApiKeyConfigured: boolean;
  anthropicApiKeyConfigured: boolean;
  voiceVadDebug: boolean;
  voiceSidecarLogProtocol: boolean;
  daemonAgentProvider: string;
  daemonOpenaiModel: string;
  daemonAnthropicModel: string;
  elevenLabsSttLanguage: string;
  elevenLabsSttModelId: string;
  voiceSttCommitResponseTimeoutMs: number;
  voiceVadThresholdMultiplier: number;
  voiceVadMinEnergyFloor: number;
  voiceVadStartMs: number;
  voiceVadEndMs: number;
  voiceVadPrerollMs: number;
  voiceStreamMinChunkMs: number;
  voiceStreamMaxChunkMs: number;
  voiceStreamMaxUtteranceMs: number;
  daemonVoiceTranscriptQueueSize: number;
  daemonVoiceTranscriptCoalesceMs: number;
  daemonVoiceTranscriptMaxMergeJobs: number;
  daemonVoiceTranscriptMaxMergeChars: number;
  sessionIdleResetMs: number;
};

const STRING_KEYS = new Set<string>([
  "daemonAgentProvider",
  "daemonOpenaiModel",
  "daemonOpenaiBaseUrl",
  "daemonAnthropicModel",
  "daemonAnthropicBaseUrl",
  "elevenLabsSttLanguage",
  "elevenLabsSttModelId",
]);

const BOOL_KEYS = new Set<string>(["voiceVadDebug", "voiceSidecarLogProtocol"]);

const NUMBER_KEYS = new Set<string>(
  PANEL_CONFIG_KEYS.filter(
    (k) => !STRING_KEYS.has(k) && !BOOL_KEYS.has(k),
  ) as string[],
);

function configurationTarget(): vscode.ConfigurationTarget {
  return vscode.workspace.workspaceFolders?.length
    ? vscode.ConfigurationTarget.Workspace
    : vscode.ConfigurationTarget.Global;
}

export async function buildVocodePanelConfigMessage(
  context: vscode.ExtensionContext,
): Promise<VocodePanelConfigMessage> {
  const c = vscode.workspace.getConfiguration("vocode");
  return {
    type: "panelConfig",
    elevenLabsApiKeyConfigured: await elevenLabsApiKeyIsConfigured(context),
    openaiApiKeyConfigured: await openaiApiKeyIsConfigured(context),
    anthropicApiKeyConfigured: await anthropicApiKeyIsConfigured(context),
    voiceVadDebug: c.get<boolean>("voiceVadDebug") === true,
    voiceSidecarLogProtocol: c.get<boolean>("voiceSidecarLogProtocol") === true,
    daemonAgentProvider: c.get<string>("daemonAgentProvider") ?? "stub",
    daemonOpenaiModel: c.get<string>("daemonOpenaiModel") ?? "gpt-4o-mini",
    daemonAnthropicModel:
      c.get<string>("daemonAnthropicModel") ?? "claude-3-5-haiku-latest",
    elevenLabsSttLanguage: c.get<string>("elevenLabsSttLanguage") ?? "en",
    elevenLabsSttModelId:
      c.get<string>("elevenLabsSttModelId") ?? "scribe_v2_realtime",
    voiceSttCommitResponseTimeoutMs: c.get<number>(
      "voiceSttCommitResponseTimeoutMs",
      5000,
    ),
    voiceVadThresholdMultiplier: c.get<number>(
      "voiceVadThresholdMultiplier",
      1.65,
    ),
    voiceVadMinEnergyFloor: c.get<number>("voiceVadMinEnergyFloor", 100),
    voiceVadStartMs: c.get<number>("voiceVadStartMs", 60),
    voiceVadEndMs: c.get<number>("voiceVadEndMs", 750),
    voiceVadPrerollMs: c.get<number>("voiceVadPrerollMs", 320),
    voiceStreamMinChunkMs: c.get<number>("voiceStreamMinChunkMs", 200),
    voiceStreamMaxChunkMs: c.get<number>("voiceStreamMaxChunkMs", 500),
    voiceStreamMaxUtteranceMs: c.get<number>("voiceStreamMaxUtteranceMs", 0),
    daemonVoiceTranscriptQueueSize: c.get<number>(
      "daemonVoiceTranscriptQueueSize",
      10,
    ),
    daemonVoiceTranscriptCoalesceMs: c.get<number>(
      "daemonVoiceTranscriptCoalesceMs",
      750,
    ),
    daemonVoiceTranscriptMaxMergeJobs: c.get<number>(
      "daemonVoiceTranscriptMaxMergeJobs",
      5,
    ),
    daemonVoiceTranscriptMaxMergeChars: c.get<number>(
      "daemonVoiceTranscriptMaxMergeChars",
      6000,
    ),
    sessionIdleResetMs: c.get<number>("sessionIdleResetMs", 1800000),
  };
}

function isPanelConfigKey(k: string): k is PanelConfigKey {
  return (PANEL_CONFIG_KEYS as readonly string[]).includes(k);
}

export async function applyVocodePanelConfigPatch(
  patch: Record<string, unknown>,
): Promise<void> {
  const config = vscode.workspace.getConfiguration("vocode");
  const target = configurationTarget();
  for (const [rawKey, rawVal] of Object.entries(patch)) {
    if (!isPanelConfigKey(rawKey)) {
      continue;
    }
    if (BOOL_KEYS.has(rawKey)) {
      if (typeof rawVal === "boolean") {
        await config.update(rawKey, rawVal, target);
      }
      continue;
    }
    if (STRING_KEYS.has(rawKey)) {
      if (typeof rawVal === "string") {
        await config.update(rawKey, rawVal, target);
      }
      continue;
    }
    if (NUMBER_KEYS.has(rawKey)) {
      if (typeof rawVal === "number" && Number.isFinite(rawVal)) {
        await config.update(rawKey, rawVal, target);
      }
    }
  }
}
