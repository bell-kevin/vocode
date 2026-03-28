import * as vscode from "vscode";

import type { DaemonClient } from "../../daemon/client";
import type { ExtensionServices } from "../services";
import { presentTranscriptResult } from "./present-result";

export async function runSendTranscript(
  client: DaemonClient,
  services: ExtensionServices,
): Promise<void> {
  if (!services.voiceSession.isRunning()) {
    void vscode.window.showWarningMessage(
      "Voice is not active. Run 'Vocode: Start Voice' first.",
    );
    return;
  }

  const editor = vscode.window.activeTextEditor;
  if (!editor) {
    void vscode.window.showWarningMessage(
      "Open a text editor so Vocode can run edit directives against the active file.",
    );
    return;
  }

  const text = await vscode.window.showInputBox({
    title: "Vocode Voice Transcript",
    prompt: "Enter transcript text to send to the daemon",
    placeHolder: "Refactor this function to handle empty input safely",
    ignoreFocusOut: true,
  });

  if (!services.voiceSession.isRunning()) {
    services.voiceStatus.setIdle();
    return;
  }

  const trimmedText = text?.trim();
  if (!trimmedText) {
    return;
  }

  const activePath = editor.document.uri.fsPath;
  const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;

  try {
    services.voiceStatus.setProcessing();
    const result = await client.transcript({
      text: trimmedText,
      activeFile: activePath,
      workspaceRoot,
    });
    await presentTranscriptResult(result, activePath);
  } catch (err) {
    const message =
      err instanceof Error ? err.message : "Failed to send transcript.";
    void vscode.window.showWarningMessage(message);
  } finally {
    if (services.voiceSession.isRunning()) {
      services.voiceStatus.setListening();
    } else {
      services.voiceStatus.setIdle();
    }
  }
}
