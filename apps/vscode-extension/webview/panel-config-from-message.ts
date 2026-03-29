import type { VocodePanelConfig } from "./panel-config-types";

/** Normalize extension `panelConfig` postMessage payload into panel state. */
export function vocodePanelConfigFromMessage(
  msg: Record<string, unknown>,
): VocodePanelConfig {
  return {
    elevenLabsApiKeyConfigured: msg.elevenLabsApiKeyConfigured === true,
    voiceVadDebug: msg.voiceVadDebug === true,
    voiceSidecarLogProtocol: msg.voiceSidecarLogProtocol === true,
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
    daemonVoiceMaxAgentTurns: Number(msg.daemonVoiceMaxAgentTurns ?? 8),
    daemonVoiceMaxIntentRetries: Number(msg.daemonVoiceMaxIntentRetries ?? 2),
    daemonVoiceMaxContextRounds: Number(msg.daemonVoiceMaxContextRounds ?? 2),
    daemonVoiceMaxContextBytes: Number(msg.daemonVoiceMaxContextBytes ?? 12000),
    daemonVoiceMaxConsecutiveContextRequests: Number(
      msg.daemonVoiceMaxConsecutiveContextRequests ?? 3,
    ),
    daemonSessionIdleResetMs: Number(msg.daemonSessionIdleResetMs ?? 1800000),
  };
}
