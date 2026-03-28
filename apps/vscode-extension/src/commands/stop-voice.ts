import * as vscode from "vscode";

import type { CommandDefinition } from "./types";

export const stopVoiceCommand: CommandDefinition = {
  id: "vocode.stopVoice",
  requiresDaemon: true,
  run: (_client, services) => {
    services.voiceSession.stop();
    services.voiceSidecar?.stop();
    services.voiceStatus.setIdle();
    services.transcriptStore.setVoiceListening(false);
    void vscode.window.showInformationMessage("Vocode stopped listening.");
  },
};
