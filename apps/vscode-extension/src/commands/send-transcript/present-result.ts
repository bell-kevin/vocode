import type { VoiceTranscriptResult } from "@vocode/protocol";
import * as vscode from "vscode";

import { dispatchTranscriptDirective } from "../../services/dispatch-transcript-directive";
import type { TranscriptPresentContext } from "../../services/present-context";
import {
  beginTranscriptUndoSession,
  finalizeTranscriptUndoSessionIfEditsApplied,
} from "../../services/undo/transcript-undo-ledger";

/** Applies workspace edits and surfaces per-directive edit/command outcomes in the UI. */
export async function presentTranscriptResult(
  result: VoiceTranscriptResult,
  activeDocumentPath: string,
): Promise<void> {
  if (!result.accepted) {
    void vscode.window.showErrorMessage("Vocode: transcript was not accepted.");
    return;
  }

  const ctx: TranscriptPresentContext = {
    activeDocumentPath,
    editLocations: {},
  };

  beginTranscriptUndoSession();
  try {
    for (const directive of result.directives ?? []) {
      if (!(await dispatchTranscriptDirective(directive, ctx))) {
        return;
      }
    }
  } finally {
    finalizeTranscriptUndoSessionIfEditsApplied();
  }
}
