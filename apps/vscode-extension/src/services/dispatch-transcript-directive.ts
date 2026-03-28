import type { VoiceTranscriptDirective } from "@vocode/protocol";

import { dispatchCommandDirective } from "./command/dispatch-command-directive";
import { dispatchEditDirective } from "./edits/dispatch-edit-directive";
import { dispatchNavigationDirective } from "./navigation/dispatch-navigation-directive";
import type { TranscriptPresentContext } from "./present-context";
import { dispatchUndoDirective } from "./undo/dispatch-undo-directive";

/**
 * Routes a single transcript directive to the matching service (parallel to daemon dispatch).
 */
export function dispatchTranscriptDirective(
  transcriptDirective: VoiceTranscriptDirective,
  ctx: TranscriptPresentContext,
): Promise<boolean> {
  switch (transcriptDirective.kind) {
    case "edit":
      return dispatchEditDirective(transcriptDirective.editDirective, ctx);
    case "command":
      return dispatchCommandDirective(transcriptDirective.commandDirective);
    case "navigate":
      return dispatchNavigationDirective(transcriptDirective, ctx);
    case "undo":
      return dispatchUndoDirective(transcriptDirective.undoDirective);
  }
}
