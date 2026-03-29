/** Mirrors extension `VocodePanelConfigMessage` (without `type`). */
export type VocodePanelConfig = {
  elevenLabsApiKeyConfigured: boolean;
  voiceVadDebug: boolean;
  voiceSidecarLogProtocol: boolean;
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
  daemonVoiceMaxAgentTurns: number;
  daemonVoiceMaxIntentRetries: number;
  daemonVoiceMaxContextRounds: number;
  daemonVoiceMaxContextBytes: number;
  daemonVoiceMaxConsecutiveContextRequests: number;
  daemonSessionIdleResetMs: number;
};
