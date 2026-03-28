import type { VoiceTranscriptDirective } from "@vocode/protocol";

import type { TranscriptApplyContext } from "../transcript/context";
import { dispatchCommand } from "./command/dispatch";
import { dispatchEdit } from "./edit/dispatch";
import { dispatchNavigation } from "./navigation/dispatch";
import { dispatchUndo } from "./undo/dispatch";

/**
 * Routes one `VoiceTranscriptDirective` to the matching handler (parallel to daemon dispatch).
 */
export function dispatchTranscript(
  transcriptDirective: VoiceTranscriptDirective,
  ctx: TranscriptApplyContext,
): Promise<boolean> {
  switch (transcriptDirective.kind) {
    case "edit":
      return dispatchEdit(transcriptDirective.editDirective, ctx);
    case "command":
      return dispatchCommand(transcriptDirective.commandDirective);
    case "navigate":
      return dispatchNavigation(transcriptDirective, ctx);
    case "undo":
      return dispatchUndo(transcriptDirective.undoDirective);
  }
}
