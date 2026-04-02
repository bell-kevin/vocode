import type { VoiceTranscriptDirective } from "@vocode/protocol";

import { dispatchTranscript } from "../directives/dispatch";
import {
  beginTranscriptUndoSession,
  finalizeTranscriptUndoSessionIfEditsApplied,
} from "../directives/undo/transcript-undo-ledger";
import type { TranscriptApplyContext } from "./context";

export type DirectiveApplyOutcome = {
  status: "ok" | "failed" | "skipped";
  message?: string;
};

/**
 * Applies a daemon directive batch to the workspace (edits, commands, navigation, undo).
 * Used for every `voice.transcript` path (voice commits and command-palette send).
 * Returns one outcome per directive (stops after the first failure).
 */
export type ApplyTranscriptProgressEvent = {
  index: number;
  phase: "start" | "complete";
  outcome?: DirectiveApplyOutcome;
};

export async function applyDirectives(
  directives: readonly VoiceTranscriptDirective[],
  activeDocumentPath: string,
  options?: {
    onProgress?: (event: ApplyTranscriptProgressEvent) => void;
  },
): Promise<DirectiveApplyOutcome[]> {
  const ctx: TranscriptApplyContext = {
    activeDocumentPath,
    editLocations: {},
  };

  const dirs = directives ?? [];
  const outcomes: DirectiveApplyOutcome[] = [];
  beginTranscriptUndoSession();
  try {
    for (let i = 0; i < dirs.length; i++) {
      const directive = dirs[i];
      options?.onProgress?.({ index: i, phase: "start" });
      const dispatchOutcome = await dispatchTranscript(directive, ctx);
      if (!dispatchOutcome.ok) {
        const failed: DirectiveApplyOutcome = {
          status: "failed",
          message: dispatchOutcome.message ?? "Directive failed to apply.",
        };
        outcomes.push(failed);
        options?.onProgress?.({ index: i, phase: "complete", outcome: failed });
        for (let j = i + 1; j < dirs.length; j++) {
          const skipped: DirectiveApplyOutcome = {
            status: "skipped",
            message: "not attempted",
          };
          outcomes.push(skipped);
          options?.onProgress?.({
            index: j,
            phase: "complete",
            outcome: skipped,
          });
        }
        return outcomes;
      }
      const ok: DirectiveApplyOutcome = { status: "ok" };
      outcomes.push(ok);
      options?.onProgress?.({ index: i, phase: "complete", outcome: ok });
    }
  } finally {
    finalizeTranscriptUndoSessionIfEditsApplied();
  }
  return outcomes;
}
