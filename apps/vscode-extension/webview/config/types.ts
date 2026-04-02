/** Mirrors extension `VocodePanelConfigMessage` (without `type`). */
export type VocodeConfig = {
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
