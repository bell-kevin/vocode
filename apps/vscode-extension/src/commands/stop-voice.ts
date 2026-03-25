import * as vscode from "vscode";

import type { CommandDefinition } from "./types";

export const stopVoiceCommand: CommandDefinition = {
  id: "vocode.stopVoice",
  requiresDaemon: true,
  run: (_client, services) => {
    services.voiceSession.stop();
    services.voiceStatus.setIdle();
    void vscode.window.showInformationMessage("Vocode stopped listening.");
  },
};
