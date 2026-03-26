import type { VoiceTranscriptResult } from "@vocode/protocol";
import * as vscode from "vscode";

import { resolveReplaceBetweenAnchors } from "./apply-edit-helpers";

function pathsMatch(a: string, b: string): boolean {
  return (
    a.replace(/\\/g, "/").toLowerCase() === b.replace(/\\/g, "/").toLowerCase()
  );
}

/** Applies successful edit steps from a transcript result using workspace edits. */
export async function applyWorkspaceEdits(
  result: VoiceTranscriptResult,
  activeDocumentPath: string,
): Promise<void> {
  const steps = result.steps ?? [];
  for (const step of steps) {
    if (step.kind !== "edit" || !step.editResult) {
      continue;
    }
    const er = step.editResult;
    if (er.kind !== "success" || er.actions.length === 0) {
      continue;
    }

    for (const action of er.actions) {
      if (!pathsMatch(action.path, activeDocumentPath)) {
        continue;
      }

      const editor = vscode.window.activeTextEditor;
      const doc = editor?.document;
      if (!doc || !pathsMatch(doc.uri.fsPath, activeDocumentPath)) {
        void vscode.window.showWarningMessage(
          "Vocode: active editor no longer matches the file used for this transcript; skipping remaining text edits.",
        );
        return;
      }

      const text = doc.getText();
      const { startOffset, endOffset } = resolveReplaceBetweenAnchors(
        text,
        action,
      );
      const wsEdit = new vscode.WorkspaceEdit();
      wsEdit.replace(
        doc.uri,
        new vscode.Range(
          doc.positionAt(startOffset),
          doc.positionAt(endOffset),
        ),
        action.newText,
      );
      const applied = await vscode.workspace.applyEdit(wsEdit);
      if (!applied) {
        void vscode.window.showWarningMessage(
          "Vocode: a workspace edit was not applied.",
        );
        return;
      }
    }
  }
}
