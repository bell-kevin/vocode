import * as vscode from "vscode";

import type { DaemonClient } from "../daemon/client";
import { applyTranscriptResult } from "../transcript/apply-result";
import {
  mergeCarriedTranscriptParams,
  recordTranscriptApplyCycle,
} from "../transcript/carry";
import type { ExtensionServices } from "./services";
import type { CommandDefinition } from "./types";

export const sendTranscriptCommand: CommandDefinition = {
  id: "vocode.sendTranscript",
  requiresDaemon: true,
  run: (client, services) => sendTranscript(client, services),
};

async function sendTranscript(
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
    const pos = editor.selection.active;
    const result = await client.transcript(
      mergeCarriedTranscriptParams({
        text: trimmedText,
        activeFile: activePath,
        workspaceRoot,
        cursorPosition: { line: pos.line, character: pos.character },
        contextSessionId: services.voiceSession.contextSessionId(),
      }),
    );
    const outcomes = await applyTranscriptResult(result, activePath);
    recordTranscriptApplyCycle(result, outcomes);
    const firstBad = outcomes.find((o) => !o.ok);
    if (result.accepted && firstBad) {
      const msg =
        firstBad.message && firstBad.message !== "not attempted"
          ? firstBad.message
          : "A directive failed to apply.";
      void vscode.window.showWarningMessage(`Vocode: ${msg}`);
    } else if (result.accepted && !firstBad) {
      services.transcriptStore.recordCompletedTranscript(trimmedText, {
        summary: result.summary?.trim() || undefined,
      });
    }
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
