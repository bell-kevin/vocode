import * as vscode from "vscode";

import type { CommandDefinition } from "./types";

export const sendTranscriptCommand: CommandDefinition = {
  id: "vocode.sendTranscript",
  requiresDaemon: true,
  run: async (client, services) => {
    if (!services.voiceSession.isRunning()) {
      void vscode.window.showWarningMessage(
        "Voice is not active. Run 'Vocode: Start Voice' first.",
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

    try {
      services.voiceStatus.setProcessing();
      await client.voiceTranscript({ text: trimmedText });
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
  },
};
