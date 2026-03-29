import type { VoiceTranscriptDirective } from "@vocode/protocol";

import type { TranscriptApplyContext } from "../../transcript/context";
import { executeNavigationDirective } from "./execute-navigation-intent";

/** Applies one navigation directive; surfaces failures as messages and returns ok flag. */
export async function dispatchNavigation(
  transcriptDirective: VoiceTranscriptDirective,
  ctx: TranscriptApplyContext,
): Promise<boolean> {
  try {
    await executeNavigationDirective(
      transcriptDirective,
      ctx.activeDocumentPath,
      ctx.editLocations,
    );
    return true;
  } catch {
    return false;
  }
}
