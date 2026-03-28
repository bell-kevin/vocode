import path from "node:path";
import type { EditDirective } from "@vocode/protocol";
import * as vscode from "vscode";

import { resolveReplaceBetweenAnchors } from "./dispatch-edit-helpers";
export interface AppliedEditLocation {
  editId?: string;
  path: string;
  selectionStart?: vscode.Position;
  selectionEnd?: vscode.Position;
}

export interface ApplyEditResultWorkspaceOutcome {
  ok: boolean;
  appliedEdits: AppliedEditLocation[];
  /** One path per successful `workspace.applyEdit` (reverse order for transcript undo). */
  undoStackOrderPaths: string[];
}

function toAbsolutePath(
  targetPath: string,
  activeDocumentPath: string,
): string {
  if (path.isAbsolute(targetPath)) {
    return targetPath;
  }
  const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
  if (workspaceRoot) {
    return path.resolve(workspaceRoot, targetPath);
  }
  return path.resolve(path.dirname(activeDocumentPath), targetPath);
}

/**
 * Applies one edit directive's actions by opening each target document and
 * submitting workspace edits. Also returns per-action edit locations for
 * follow-up navigation (e.g. reveal_edit).
 */
export async function dispatchEditResultWorkspaceEdits(
  editDirective: EditDirective,
  activeDocumentPath: string,
): Promise<ApplyEditResultWorkspaceOutcome> {
  if (editDirective.kind !== "success" || editDirective.actions.length === 0) {
    return { ok: true, appliedEdits: [], undoStackOrderPaths: [] };
  }

  const appliedEdits: AppliedEditLocation[] = [];
  const undoStackOrderPaths: string[] = [];

  for (const action of editDirective.actions) {
    const actionPath = toAbsolutePath(action.path, activeDocumentPath);
    const wsEdit = new vscode.WorkspaceEdit();

    if (action.kind === "replace_between_anchors") {
      const targetDoc = await vscode.workspace.openTextDocument(actionPath);
      const text = targetDoc.getText();
      const { startOffset, endOffset } = resolveReplaceBetweenAnchors(
        text,
        action,
      );
      const startPos = targetDoc.positionAt(startOffset);
      const endPos = targetDoc.positionAt(endOffset);
      wsEdit.replace(
        targetDoc.uri,
        new vscode.Range(startPos, endPos),
        action.newText,
      );
      appliedEdits.push({
        editId: action.editId,
        path: actionPath,
        selectionStart: startPos,
        selectionEnd: startPos,
      });
    } else if (action.kind === "create_file") {
      const uri = vscode.Uri.file(actionPath);
      wsEdit.createFile(uri, { overwrite: true, ignoreIfExists: false });
      wsEdit.insert(uri, new vscode.Position(0, 0), action.content);
      appliedEdits.push({
        editId: action.editId,
        path: actionPath,
        selectionStart: new vscode.Position(0, 0),
        selectionEnd: new vscode.Position(0, 0),
      });
    } else if (action.kind === "append_to_file") {
      const uri = vscode.Uri.file(actionPath);
      const targetDoc = await vscode.workspace.openTextDocument(uri);
      const end = targetDoc.positionAt(targetDoc.getText().length);
      wsEdit.insert(uri, end, action.text);
      appliedEdits.push({
        editId: action.editId,
        path: actionPath,
        selectionStart: end,
        selectionEnd: end,
      });
    } else {
      continue;
    }

    const applied = await vscode.workspace.applyEdit(wsEdit);
    if (!applied) {
      void vscode.window.showWarningMessage(
        "Vocode: workspace edit was not applied.",
      );
      return { ok: false, appliedEdits, undoStackOrderPaths };
    }

    const savedDocument = await vscode.workspace.openTextDocument(actionPath);
    const saved = await savedDocument.save();
    if (!saved) {
      void vscode.window.showWarningMessage(
        `Vocode: could not save ${path.basename(actionPath)} after applying edit.`,
      );
      return { ok: false, appliedEdits, undoStackOrderPaths };
    }

    undoStackOrderPaths.push(actionPath);
  }

  return { ok: true, appliedEdits, undoStackOrderPaths };
}
