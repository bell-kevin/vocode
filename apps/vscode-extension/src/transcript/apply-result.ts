import type { VoiceTranscriptResult } from "@vocode/protocol";

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
 * Applies a daemon `VoiceTranscriptResult` to the workspace (edits, commands, navigation, undo).
 * Used by voice and by the manual “send transcript” command — not command-specific.
 * Returns one outcome per directive (stops after the first failure).
 */
export async function applyTranscriptResult(
  result: VoiceTranscriptResult,
  activeDocumentPath: string,
): Promise<DirectiveApplyOutcome[]> {
  if (!result.success) {
    return [];
  }

  const ctx: TranscriptApplyContext = {
    activeDocumentPath,
    editLocations: {},
  };

  const dirs = result.directives ?? [];
  const outcomes: DirectiveApplyOutcome[] = [];
  beginTranscriptUndoSession();
  try {
    for (let i = 0; i < dirs.length; i++) {
      const directive = dirs[i];
      const dispatchOutcome = await dispatchTranscript(directive, ctx);
      if (!dispatchOutcome.ok) {
        outcomes.push({
          status: "failed",
          message: dispatchOutcome.message ?? "Directive failed to apply.",
        });
        for (let j = i + 1; j < dirs.length; j++) {
          outcomes.push({ status: "skipped", message: "not attempted" });
        }
        return outcomes;
      }
      outcomes.push({ status: "ok" });
    }
  } finally {
    finalizeTranscriptUndoSessionIfEditsApplied();
  }
  return outcomes;
}
