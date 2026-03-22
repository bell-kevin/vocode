import * as vscode from "vscode";

import { DaemonClient } from "./client/daemon-client";
import { spawnDaemon } from "./daemon/spawn";

function getDaemonClient(
  context: vscode.ExtensionContext,
): DaemonClient | null {
  try {
    const daemon = spawnDaemon(context);
    console.log(`Vocode daemon started from ${daemon.binaryPath}`);
    return new DaemonClient(daemon.process);
  } catch (err) {
    const message =
      err instanceof Error ? err.message : "Unknown daemon startup error";

    console.error(message);
    void vscode.window.showErrorMessage(
      `Failed to start Vocode daemon: ${message}`,
    );
  }
  return null;
}

export function activate(context: vscode.ExtensionContext) {
  console.log("Vocode extension activated");

  const client = getDaemonClient(context);

  const pingDaemon = vscode.commands.registerCommand(
    "vocode.ping",
    async () => {
      if (!client) {
        void vscode.window.showErrorMessage("Vocode daemon is not running.");
        return;
      }

      try {
        const result = await client.ping({});
        void vscode.window.showInformationMessage(
          `Vocode daemon says: ${result.message}`,
        );
      } catch (error) {
        const message =
          error instanceof Error ? error.message : "Unknown ping error";

        console.error("[vocode] ping failed:", error);
        void vscode.window.showErrorMessage(`Ping failed: ${message}`);
      }
    },
  );

  const startVoice = vscode.commands.registerCommand(
    "vocode.startVoice",
    () => {
      vscode.window.showInformationMessage("Vocode: Start Voice");
    },
  );

  const stopVoice = vscode.commands.registerCommand("vocode.stopVoice", () => {
    vscode.window.showInformationMessage("Vocode: Stop Voice");
  });

  const applyEdit = vscode.commands.registerCommand("vocode.applyEdit", () => {
    vscode.window.showInformationMessage("Vocode: Apply Edit");
  });

  const runCommand = vscode.commands.registerCommand(
    "vocode.runCommand",
    () => {
      vscode.window.showInformationMessage("Vocode: Run Command");
    },
  );

  context.subscriptions.push(
    pingDaemon,
    startVoice,
    stopVoice,
    applyEdit,
    runCommand,
    {
      dispose: () => {
        client?.dispose();
      },
    },
  );
}

export function deactivate() {}
