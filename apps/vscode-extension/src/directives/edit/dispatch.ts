import type { EditDirective } from "@vocode/protocol";
import * as vscode from "vscode";

import type { TranscriptApplyContext } from "../../transcript/context";
import { recordAppliedEditUndoPaths } from "../undo/transcript-undo-ledger";
import { dispatchEditResultWorkspaceEdit } from "./dispatch-workspace-edit";

/** Applies one edit directive; records undo paths for the transcript ledger. */
export async function dispatchEdit(
  edit: EditDirective | undefined,
  ctx: TranscriptApplyContext,
): Promise<boolean> {
  if (!edit) {
    void vscode.window.showWarningMessage("Vocode: missing editDirective.");
    return false;
  }
  const applyOutcome = await dispatchEditResultWorkspaceEdit(
    edit,
    ctx.activeDocumentPath,
  );
  if (!applyOutcome.ok) {
    return false;
  }
  recordAppliedEditUndoPaths(applyOutcome.undoStackOrderPaths);
  for (const loc of applyOutcome.appliedEdits) {
    if (!loc.editId) continue;
    ctx.editLocations[loc.editId] = {
      path: loc.path,
      selectionStart: loc.selectionStart,
      selectionEnd: loc.selectionEnd,
    };
  }
  return true;
}
