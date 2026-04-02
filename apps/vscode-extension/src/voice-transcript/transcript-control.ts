import type { VoiceTranscriptParams } from "@vocode/protocol";
import * as vscode from "vscode";

import type { ExtensionServices } from "../commands/services";
import { transcriptWorkspaceRoot } from "./workspace-root";

/**
 * Notifies the daemon to clear session state for clarify/search UI cancel, then the caller
 * should update the panel store (e.g. abortClarifyAsSkipped / dismissSearchState).
 */
export async function sendTranscriptControlRequest(
  services: ExtensionServices,
  kind: "cancel_clarify" | "cancel_selection",
  contextSessionId: string | undefined,
): Promise<boolean> {
  const { client } = services;
  if (!client) {
    void vscode.window.showWarningMessage(
      "Vocode daemon is not connected; cannot cancel.",
    );
    return false;
  }

  const editor = vscode.window.activeTextEditor;
  const activeFile = editor?.document.uri.fsPath?.trim() ?? "";

  const params: VoiceTranscriptParams = {
    controlRequest: kind,
    activeFile: activeFile || undefined,
    ...(activeFile ? { focusedWorkspacePath: activeFile } : {}),
    workspaceRoot: transcriptWorkspaceRoot(activeFile),
    cursorPosition: editor
      ? {
          line: editor.selection.active.line,
          character: editor.selection.active.character,
        }
      : { line: 0, character: 0 },
    activeSelection: editor
      ? {
          startLine: editor.selection.start.line,
          startChar: editor.selection.start.character,
          endLine: editor.selection.end.line,
          endChar: editor.selection.end.character,
        }
      : {
          startLine: 0,
          startChar: 0,
          endLine: 0,
          endChar: 0,
        },
    ...(contextSessionId ? { contextSessionId } : {}),
  };

  try {
    const result = await client.transcript(params);
    if (!result.success) {
      void vscode.window.showWarningMessage(
        result.summary?.trim() || "Cancel request was rejected.",
      );
      return false;
    }
    return true;
  } catch (err) {
    const message =
      err instanceof Error ? err.message : "Unknown error during cancel.";
    void vscode.window.showWarningMessage(message);
    return false;
  }
}
