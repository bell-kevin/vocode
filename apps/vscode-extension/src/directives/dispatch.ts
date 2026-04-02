import type { VoiceTranscriptDirective } from "@vocode/protocol";

import type { TranscriptApplyContext } from "../voice-transcript/context";
import { dispatchCodeAction } from "./code-action/dispatch";
import { dispatchCommand } from "./command/dispatch";
import { dispatchEdit } from "./edit/dispatch";
import { dispatchFormat } from "./format/dispatch";
import { dispatchNavigation } from "./navigation/dispatch";
import { dispatchRename } from "./rename/dispatch";
import { dispatchUndo } from "./undo/dispatch";
import { dispatchWorkspacePath } from "./workspace-path/dispatch";

export type DirectiveDispatchOutcome = {
  ok: boolean;
  message?: string;
};

/**
 * Routes one `VoiceTranscriptDirective` to the matching handler (parallel to daemon dispatch).
 */
export function dispatchTranscript(
  transcriptDirective: VoiceTranscriptDirective,
  ctx: TranscriptApplyContext,
): Promise<DirectiveDispatchOutcome> {
  switch (transcriptDirective.kind) {
    case "edit":
      return dispatchEdit(transcriptDirective.editDirective, ctx);
    case "command":
      return dispatchCommand(transcriptDirective.commandDirective);
    case "navigate":
      return dispatchNavigation(transcriptDirective, ctx);
    case "undo":
      return dispatchUndo(transcriptDirective.undoDirective);
    case "rename":
      return dispatchRename(transcriptDirective.renameDirective);
    case "code_action":
      return dispatchCodeAction(transcriptDirective.codeActionDirective);
    case "format":
      return dispatchFormat(transcriptDirective.formatDirective);
    case "delete_file":
    case "move_path":
    case "create_folder":
      return dispatchWorkspacePath(transcriptDirective);
  }
}
