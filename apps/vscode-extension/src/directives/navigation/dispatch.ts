import type { VoiceTranscriptDirective } from "@vocode/protocol";

import type { TranscriptApplyContext } from "../../voice-transcript/context";
import type { DirectiveDispatchOutcome } from "../dispatch";
import { executeNavigationDirective } from "./execute-navigation-intent";

/** Applies one navigation directive; surfaces failures as messages and returns ok flag. */
export async function dispatchNavigation(
  transcriptDirective: VoiceTranscriptDirective,
  ctx: TranscriptApplyContext,
): Promise<DirectiveDispatchOutcome> {
  try {
    await executeNavigationDirective(
      transcriptDirective,
      ctx.activeDocumentPath,
      ctx.editLocations,
    );
    return { ok: true };
  } catch (err) {
    const message = err instanceof Error ? err.message : "navigation failed";
    return { ok: false, message };
  }
}
