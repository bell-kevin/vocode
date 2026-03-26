import type { EditApplyResult } from "@vocode/protocol";
import * as vscode from "vscode";

import { resolveReplaceBetweenAnchors } from "./apply-edit-helpers";

function pathsMatch(a: string, b: string): boolean {
  return (
    a.replace(/\\/g, "/").toLowerCase() === b.replace(/\\/g, "/").toLowerCase()
  );
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
): Promise<boolean> {
  if (editResult.kind !== "success" || editResult.actions.length === 0) {
    return true;
  }

  const editor = vscode.window.activeTextEditor;
  const doc = editor?.document;
  if (!doc || !pathsMatch(doc.uri.fsPath, activeDocumentPath)) {
    void vscode.window.showWarningMessage(
      "Vocode: active editor no longer matches the file used for this transcript.",
    );
    return false;
  }

  for (const action of editResult.actions) {
    if (!pathsMatch(action.path, activeDocumentPath)) {
      continue;
    }

    const text = doc.getText();
    const { startOffset, endOffset } = resolveReplaceBetweenAnchors(
      text,
      action,
    );

    const wsEdit = new vscode.WorkspaceEdit();
    wsEdit.replace(
      doc.uri,
      new vscode.Range(doc.positionAt(startOffset), doc.positionAt(endOffset)),
      action.newText,
    );
    const applied = await vscode.workspace.applyEdit(wsEdit);
    if (!applied) {
      void vscode.window.showWarningMessage(
        "Vocode: workspace edit was not applied.",
      );
      return false;
    }
  }

  return true;
}
