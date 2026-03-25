import * as vscode from "vscode";

import type { CommandDefinition } from "./types";

export const startVoiceCommand: CommandDefinition = {
  id: "vocode.startVoice",
  requiresDaemon: true,
  run: (_client, services) => {
    if (services.voiceSession.isRunning()) {
      void vscode.window.showInformationMessage("Vocode is already listening.");
      return;
    }

    services.voiceSession.start();
    services.voiceStatus.setListening();
    void vscode.window.showInformationMessage("Vocode started listening.");
  },
};
