import * as vscode from "vscode";

import type { DaemonClient } from "../daemon/client";
import { runDaemonTranscriptForPendingId } from "../voice-transcript/run-daemon-transcript";
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
  const text = await vscode.window.showInputBox({
    title: "Vocode Voice Transcript",
    prompt: "Enter transcript text to send to vocode-cored",
    placeHolder: "Refactor this function to handle empty input safely",
    ignoreFocusOut: true,
  });

  const trimmedText = text?.trim();
  if (!trimmedText) {
    return;
  }

  services.voiceSession.ensureContextSessionForManualTranscript();

  const clarify =
    services.mainPanelStore.consumeClarifyPromptAnswerForSend(trimmedText);
  const displayText = clarify?.displayText ?? trimmedText;
  const sendText = clarify?.sendText ?? trimmedText;

  const pendingId = services.mainPanelStore.enqueueCommitted(displayText);
  if (pendingId === null) {
    return;
  }

  const editor = vscode.window.activeTextEditor;
  if (!editor) {
    const message =
      "Open a text editor so Vocode can run actions against the active file.";
    services.mainPanelStore.markError(pendingId, message);
    void vscode.window.showWarningMessage(message);
    return;
  }

  services.voiceStatus.setProcessing();
  services.mainPanelStore.markProcessing(pendingId);

  try {
    await runDaemonTranscriptForPendingId(
      services,
      client,
      editor,
      pendingId,
      sendText,
    );
  } finally {
    if (services.voiceSession.isRunning()) {
      services.voiceStatus.setListening();
    } else {
      services.voiceStatus.setIdle();
    }
  }
}
