import path from "node:path";
import type { EditApplyResult } from "@vocode/protocol";
import * as vscode from "vscode";

import { resolveReplaceBetweenAnchors } from "./apply-edit-helpers";
export interface AppliedEditLocation {
  editId?: string;
  path: string;
  selectionStart?: vscode.Position;
  selectionEnd?: vscode.Position;
}

export interface ApplyEditResultWorkspaceOutcome {
  ok: boolean;
  appliedEdits: AppliedEditLocation[];
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
 * Applies one edit step's edit actions (anchor-based `replace_between_anchors`)
 * directly into the active VS Code editor.
 *
 * Returns `true` if the edit result either had nothing to apply or was applied
 * successfully. Returns `false` if the active editor/file no longer matches
 * or the workspace edit fails.
 */
export async function applyEditResultWorkspaceEdits(
  editResult: EditApplyResult,
  activeDocumentPath: string,
): Promise<ApplyEditResultWorkspaceOutcome> {
  if (editResult.kind !== "success" || editResult.actions.length === 0) {
    return { ok: true, appliedEdits: [] };
  }

  const appliedEdits: AppliedEditLocation[] = [];

  for (const action of editResult.actions) {
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
      return { ok: false, appliedEdits };
    }
  }

  return { ok: true, appliedEdits };
}
