import * as vscode from "vscode";

import type { CommandDefinition } from "./types";

export const startVoiceCommand: CommandDefinition = {
  id: "vocode.startVoice",
  requiresDaemon: true,
  run: async (client, services) => {
    services.voiceStatus.setListening();

    const text = await vscode.window.showInputBox({
      title: "Vocode Voice Transcript",
      prompt: "Enter transcript text to send to the daemon",
      placeHolder: "Refactor this function to handle empty input safely",
      ignoreFocusOut: true,
    });

    if (!text) {
      services.voiceStatus.setIdle();
      return;
    }

    try {
      services.voiceStatus.setProcessing();
      await client.voiceTranscript({ text });

      void vscode.window.showInformationMessage(
        "Vocode transcript sent to daemon.",
      );
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to send transcript.";
      void vscode.window.showWarningMessage(message);
    } finally {
      services.voiceStatus.setIdle();
    }
  },
};
