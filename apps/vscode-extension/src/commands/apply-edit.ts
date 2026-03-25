import type {
  EditAction,
  EditApplyParams,
  ReplaceBetweenAnchorsAction,
} from "@vocode/protocol";
import { isEditApplyResult } from "@vocode/protocol";
import * as vscode from "vscode";

import { resolveReplaceBetweenAnchors } from "./apply-edit-helpers";
import type { CommandDefinition } from "./types";

interface DocumentEditState {
  document: vscode.TextDocument;
  nextText: string;
}

function describeAction(action: EditAction): string {
  switch (action.kind) {
    case "replace_between_anchors":
      return describeReplaceBetweenAnchorsAction(action);
    default:
      return "Editing file…";
  }
}

function describeReplaceBetweenAnchorsAction(
  action: ReplaceBetweenAnchorsAction,
): string {
  const nextText = action.newText.toLowerCase();
  if (
    nextText.includes("for (") ||
    nextText.includes("for(") ||
    nextText.includes("while (") ||
    nextText.includes("while(")
  ) {
    return "Adding loop…";
  }

  return `Editing ${vscode.workspace.asRelativePath(action.path, false)}…`;
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

    const result = await vscode.window.withProgress(
      {
        location: vscode.ProgressLocation.Notification,
        title: "Vocode",
      },
      async (progress) => {
        progress.report({ message: "Understanding request…" });
        const response = await client.applyEdit(params);
        progress.report({ message: "Planning changes…" });
        return response;
      },
    );

    if (!isEditApplyResult(result)) {
      throw new Error("Daemon returned an invalid edit.apply result.");
    }

    switch (result.kind) {
      case "failure":
        if (!result.failure) {
          throw new Error("Daemon returned failure kind without failure data.");
        }
        void vscode.window.showWarningMessage(
          `Vocode could not produce a safe edit: ${result.failure.message}`,
        );
        return;
      case "noop":
        if (!result.reason) {
          throw new Error("Daemon returned noop kind without reason.");
        }
        void vscode.window.showInformationMessage(result.reason);
        return;
      case "success": {
        if (!result.actions) {
          throw new Error("Daemon returned success kind without actions.");
        }

        const workspaceEdit = new vscode.WorkspaceEdit();
        const documentsByPath = new Map<string, DocumentEditState>();

        for (const action of result.actions) {
          void vscode.window.setStatusBarMessage(describeAction(action), 2_000);

          let state = documentsByPath.get(action.path);
          if (!state) {
            const actionUri = vscode.Uri.file(action.path);
            const actionDocument =
              await vscode.workspace.openTextDocument(actionUri);
            state = {
              document: actionDocument,
              nextText: actionDocument.getText(),
            };
            documentsByPath.set(action.path, state);
          }

          switch (action.kind) {
            case "replace_between_anchors": {
              const replacement = resolveReplaceBetweenAnchors(
                state.nextText,
                action,
              );

              const range = new vscode.Range(
                state.document.positionAt(replacement.startOffset),
                state.document.positionAt(replacement.endOffset),
              );

              workspaceEdit.replace(
                state.document.uri,
                range,
                replacement.replacementText,
              );

              state.nextText = replacement.nextText;
              break;
            }
            default:
              throw new Error(`Unsupported action kind: ${action.kind}`);
          }
        }

        const success = await vscode.workspace.applyEdit(workspaceEdit);
        if (!success) {
          throw new Error("VS Code failed to apply the edit.");
        }

        void vscode.window.showInformationMessage("Vocode edit applied.");
        return;
      }
    }
  },
};
