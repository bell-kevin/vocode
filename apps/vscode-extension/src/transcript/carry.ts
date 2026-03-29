import type {
  VoiceTranscriptDirectiveApplyItem,
  VoiceTranscriptParams,
  VoiceTranscriptResult,
} from "@vocode/protocol";

export type DirectiveApplyOutcome = {
  ok: boolean;
  message?: string;
};

let pendingReportApplyBatchId: string | undefined;
let pendingLastBatchApply: VoiceTranscriptDirectiveApplyItem[] | undefined;

/** Merges one-shot lastBatchApply + reportApplyBatchId into the next voice.transcript params. */
export function mergeCarriedTranscriptParams(
  base: VoiceTranscriptParams,
): VoiceTranscriptParams {
  const out: VoiceTranscriptParams = { ...base };
  if (pendingReportApplyBatchId !== undefined) {
    out.reportApplyBatchId = pendingReportApplyBatchId;
    pendingReportApplyBatchId = undefined;
  }
  if (pendingLastBatchApply !== undefined) {
    out.lastBatchApply = pendingLastBatchApply;
    pendingLastBatchApply = undefined;
  }
  return out;
}

/** After applying directives, queue apply report fields for the next RPC. */
export function recordTranscriptApplyCycle(
  result: VoiceTranscriptResult,
  outcomes: DirectiveApplyOutcome[],
): void {
  const dirs = result.directives ?? [];
  const batchId = result.applyBatchId?.trim() ?? "";
  if (
    result.success &&
    dirs.length > 0 &&
    batchId !== "" &&
    outcomes.length === dirs.length
  ) {
    pendingReportApplyBatchId = batchId;
    pendingLastBatchApply = outcomes.map((o) => ({
      ok: o.ok,
      ...(o.message !== undefined && o.message !== ""
        ? { message: o.message }
        : {}),
    }));
  }
}
