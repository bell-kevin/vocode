import type { VocodeConfig } from "./types";

/** Normalize extension `panelConfig` postMessage payload into panel state. */
export function vocodeConfigFromMessage(
  msg: Record<string, unknown>,
): VocodeConfig {
  return {
    elevenLabsApiKeyConfigured: msg.elevenLabsApiKeyConfigured === true,
    openaiApiKeyConfigured: msg.openaiApiKeyConfigured === true,
    anthropicApiKeyConfigured: msg.anthropicApiKeyConfigured === true,
    voiceVadDebug: msg.voiceVadDebug === true,
    voiceSidecarLogProtocol: msg.voiceSidecarLogProtocol === true,
    daemonAgentProvider: String(msg.daemonAgentProvider ?? "stub"),
    daemonOpenaiModel: String(msg.daemonOpenaiModel ?? "gpt-4o-mini"),
    daemonAnthropicModel: String(
      msg.daemonAnthropicModel ?? "claude-3-5-haiku-latest",
    ),
    elevenLabsSttLanguage: String(msg.elevenLabsSttLanguage ?? "en"),
    elevenLabsSttModelId: String(
      msg.elevenLabsSttModelId ?? "scribe_v2_realtime",
    ),
    voiceSttCommitResponseTimeoutMs: Number(
      msg.voiceSttCommitResponseTimeoutMs ?? 5000,
    ),
    voiceVadThresholdMultiplier: Number(
      msg.voiceVadThresholdMultiplier ?? 1.65,
    ),
    voiceVadMinEnergyFloor: Number(msg.voiceVadMinEnergyFloor ?? 100),
    voiceVadStartMs: Number(msg.voiceVadStartMs ?? 60),
    voiceVadEndMs: Number(msg.voiceVadEndMs ?? 750),
    voiceVadPrerollMs: Number(msg.voiceVadPrerollMs ?? 320),
    voiceStreamMinChunkMs: Number(msg.voiceStreamMinChunkMs ?? 200),
    voiceStreamMaxChunkMs: Number(msg.voiceStreamMaxChunkMs ?? 500),
    voiceStreamMaxUtteranceMs: Number(msg.voiceStreamMaxUtteranceMs ?? 0),
    daemonVoiceTranscriptQueueSize: Number(
      msg.daemonVoiceTranscriptQueueSize ?? 10,
    ),
    daemonVoiceTranscriptCoalesceMs: Number(
      msg.daemonVoiceTranscriptCoalesceMs ?? 750,
    ),
    daemonVoiceTranscriptMaxMergeJobs: Number(
      msg.daemonVoiceTranscriptMaxMergeJobs ?? 5,
    ),
    daemonVoiceTranscriptMaxMergeChars: Number(
      msg.daemonVoiceTranscriptMaxMergeChars ?? 6000,
    ),
    sessionIdleResetMs: Number(msg.sessionIdleResetMs ?? 1800000),
  };
}
