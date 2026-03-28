import type { VoiceTranscriptDirective } from "@vocode/protocol";
import * as vscode from "vscode";

import type { TranscriptPresentContext } from "../present-context";
import { executeNavigationDirective } from "./execute-navigation-intent";

/** Applies one navigation directive; surfaces failures as messages and returns ok flag. */
export async function dispatchNavigationDirective(
  transcriptDirective: VoiceTranscriptDirective,
  ctx: TranscriptPresentContext,
): Promise<boolean> {
  try {
    await executeNavigationDirective(
      transcriptDirective,
      ctx.activeDocumentPath,
      ctx.editLocations,
    );
    return true;
  } catch (err) {
    const message = err instanceof Error ? err.message : "navigation failed";
    void vscode.window.showErrorMessage(`Vocode navigation: ${message}`);
    return false;
  }
}
