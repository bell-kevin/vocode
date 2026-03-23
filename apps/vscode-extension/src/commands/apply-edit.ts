import type { EditApplyParams } from "@vocode/protocol";
import { isEditApplyResult } from "@vocode/protocol";
import * as vscode from "vscode";

import { applyReplaceBetweenAnchors } from "./apply-edit-helpers";
import type { CommandDefinition } from "./types";

async function replaceWholeDocument(
  editor: vscode.TextEditor,
  newText: string,
): Promise<void> {
  const document = editor.document;
  const lastLine = document.lineAt(document.lineCount - 1);
  const fullRange = new vscode.Range(
    new vscode.Position(0, 0),
    lastLine.rangeIncludingLineBreak.end,
  );

  const success = await editor.edit((editBuilder) => {
    editBuilder.replace(fullRange, newText);
  });

  if (!success) {
    throw new Error("VS Code failed to apply the edit.");
  }
}

export const applyEditCommand: CommandDefinition = {
  id: "vocode.applyEdit",
  requiresDaemon: true,
  run: async (client) => {
    const editor = vscode.window.activeTextEditor;
    if (!editor) {
      void vscode.window.showErrorMessage("No active editor.");
      return;
    }

    const instruction = await vscode.window.showInputBox({
      title: "Vocode Apply Edit",
      prompt: "Describe the edit to apply",
      placeHolder:
        'Insert statement "console.log(value)" inside current function',
      ignoreFocusOut: true,
    });

    if (!instruction) {
      return;
    }

    const document = editor.document;
    const params: EditApplyParams = {
      instruction,
      activeFile: document.uri.fsPath,
      fileText: document.getText(),
    };

    const result = await client.applyEdit(params);

    if (!isEditApplyResult(result)) {
      throw new Error("Daemon returned an invalid edit/apply result.");
    }

    if (result.failure) {
      void vscode.window.showWarningMessage(
        `Vocode could not produce a safe edit: ${result.failure.message}`,
      );
      return;
    }

    if (result.actions.length === 0) {
      void vscode.window.showWarningMessage(
        "Vocode could not produce a safe edit for that instruction.",
      );
      return;
    }

    let nextText = document.getText();

    for (const action of result.actions) {
      if (action.path !== document.uri.fsPath) {
        throw new Error(`Received action for unsupported file: ${action.path}`);
      }

      switch (action.kind) {
        case "replace_between_anchors":
          nextText = applyReplaceBetweenAnchors(nextText, action);
          break;
        default:
          throw new Error(`Unsupported action kind: ${action.kind}`);
      }
    }

    await replaceWholeDocument(editor, nextText);
    void vscode.window.showInformationMessage("Vocode edit applied.");
  },
};
