import type { EditDirective } from "@vocode/protocol";

import type { TranscriptApplyContext } from "../../voice-transcript/context";
import type { DirectiveDispatchOutcome } from "../dispatch";
import { recordAppliedEditUndoPaths } from "../undo/transcript-undo-ledger";
import { dispatchEditResultWorkspaceEdit } from "./dispatch-workspace-edit";

/** Applies one edit directive; records undo paths for the transcript ledger. */
export async function dispatchEdit(
  edit: EditDirective | undefined,
  ctx: TranscriptApplyContext,
): Promise<DirectiveDispatchOutcome> {
  if (!edit) {
    return { ok: false, message: "missing edit directive" };
  }
  try {
    const applyOutcome = await dispatchEditResultWorkspaceEdit(
      edit,
      ctx.activeDocumentPath,
    );
    if (!applyOutcome.ok) {
      return {
        ok: false,
        message:
          applyOutcome.message?.trim() ||
          "edit directive failed to apply workspace edit",
      };
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
    return { ok: true };
  } catch (err) {
    const message = err instanceof Error ? err.message : "edit dispatch failed";
    return { ok: false, message };
  }
}
