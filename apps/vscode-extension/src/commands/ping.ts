import * as vscode from "vscode";

import type { CommandDefinition } from "./types";

export const pingCommand: CommandDefinition = {
  id: "vocode.ping",
  requiresDaemon: true,
  run: async (client) => {
    const result = await client.ping({});
    void vscode.window.showInformationMessage(
      `Vocode core says: ${result.message}`,
    );
  },
};
