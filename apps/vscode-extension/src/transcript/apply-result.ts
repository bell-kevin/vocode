import type { VoiceTranscriptResult } from "@vocode/protocol";
import * as vscode from "vscode";

import { dispatchTranscript } from "../directives/dispatch";
import {
  beginTranscriptUndoSession,
  finalizeTranscriptUndoSessionIfEditsApplied,
} from "../directives/undo/transcript-undo-ledger";
import type { TranscriptApplyContext } from "./context";

/**
 * Applies a daemon `VoiceTranscriptResult` to the workspace (edits, commands, navigation, undo).
 * Used by voice and by the manual “send transcript” command — not command-specific.
 */
export async function applyTranscriptResult(
  result: VoiceTranscriptResult,
  activeDocumentPath: string,
): Promise<void> {
  if (!result.accepted) {
    void vscode.window.showErrorMessage("Vocode: transcript was not accepted.");
    return;
  }

  const ctx: TranscriptApplyContext = {
    activeDocumentPath,
    editLocations: {},
  };

  beginTranscriptUndoSession();
  try {
    for (const directive of result.directives ?? []) {
      if (!(await dispatchTranscript(directive, ctx))) {
        return;
      }
    }
  } finally {
    finalizeTranscriptUndoSessionIfEditsApplied();
  }
}
