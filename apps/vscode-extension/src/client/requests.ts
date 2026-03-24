function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function hasOnlyKeys(value: Record<string, unknown>, keys: string[]): boolean {
  const allowed = new Set(keys);
  return Object.keys(value).every((key) => allowed.has(key));
}

export interface VoiceTranscriptParams {
  text: string;
}

export interface VoiceTranscriptResult {
  accepted: boolean;
}

export function isVoiceTranscriptResult(
  value: unknown,
): value is VoiceTranscriptResult {
  return (
    isRecord(value) &&
    hasOnlyKeys(value, ["accepted"]) &&
    typeof value.accepted === "boolean"
  );
}
