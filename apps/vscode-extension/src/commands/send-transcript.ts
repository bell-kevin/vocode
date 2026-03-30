import * as vscode from "vscode";

import type { DaemonClient } from "../daemon/client";
import { applyTranscriptResult } from "../transcript/apply-result";
import {
  mergeCarriedTranscriptParams,
  recordTranscriptApplyCycle,
} from "../transcript/carry";
import { FAILED_TO_PROCESS_TRANSCRIPT } from "../transcript/messages";
import { transcriptWorkspaceRoot } from "../transcript/workspace-root";
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

  try {
    services.voiceStatus.setProcessing();
    const pos = editor.selection.active;
    const result = await client.transcript(
      mergeCarriedTranscriptParams({
        text: trimmedText,
        activeFile: activePath,
        workspaceRoot: transcriptWorkspaceRoot(activePath),
        cursorPosition: { line: pos.line, character: pos.character },
        contextSessionId: services.voiceSession.contextSessionId(),
      }),
    );
    const outcomes = await applyTranscriptResult(result, activePath);
    recordTranscriptApplyCycle(result, outcomes);
    const firstFailed = outcomes.find((o) => o.status === "failed");
    if (!result.success) {
      services.mainPanelStore.recordCompletedTranscript(trimmedText, {
        errorMessage: FAILED_TO_PROCESS_TRANSCRIPT,
      });
    } else if (firstFailed) {
      const msg =
        firstFailed.message && firstFailed.message !== "not attempted"
          ? firstFailed.message
          : "A directive failed to apply.";
      services.mainPanelStore.recordCompletedTranscript(trimmedText, {
        errorMessage: msg,
      });
    } else {
      services.mainPanelStore.recordCompletedTranscript(trimmedText, {
        summary: result.summary?.trim() || undefined,
      });
    }
  } catch (err) {
    const message =
      err instanceof Error ? err.message : "Failed to send transcript.";
    services.mainPanelStore.recordCompletedTranscript(trimmedText, {
      errorMessage: message,
    });
  } finally {
    if (services.voiceSession.isRunning()) {
      services.voiceStatus.setListening();
    } else {
      services.voiceStatus.setIdle();
    }
  }
}
