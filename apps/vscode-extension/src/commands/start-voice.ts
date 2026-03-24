import * as vscode from "vscode";

import type { CommandDefinition } from "./types";

export const startVoiceCommand: CommandDefinition = {
  id: "vocode.startVoice",
  requiresDaemon: true,
  run: async (client) => {
    const text = await vscode.window.showInputBox({
      title: "Vocode Voice Transcript",
      prompt: "Enter transcript text to send to the daemon",
      placeHolder: "Refactor this function to handle empty input safely",
      ignoreFocusOut: true,
      validateInput: (value) =>
        value.trim().length === 0 ? "Transcript text cannot be empty." : null,
    });

    if (!text) {
      return;
    }

    const result = await client.voiceTranscript({ text });

    if (result.accepted) {
      void vscode.window.showInformationMessage(
        "Vocode transcript sent to daemon.",
      );
      return;
    }

    void vscode.window.showWarningMessage(
      "Daemon did not accept transcript.",
    );
  },
};
