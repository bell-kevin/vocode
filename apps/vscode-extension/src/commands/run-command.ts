import type { CommandRunParams } from "@vocode/protocol";
import { isCommandRunResult } from "@vocode/protocol";
import * as vscode from "vscode";

import type { CommandDefinition } from "./types";

export const runCommand: CommandDefinition = {
  id: "vocode.runCommand",
  requiresDaemon: true,
  run: async (client) => {
    const command = await vscode.window.showInputBox({
      title: "Vocode Run Command",
      prompt: "Enter the executable name (cmd.exe or powershell.exe etc.)",
      placeHolder: "cmd.exe",
      ignoreFocusOut: true,
    });

    if (!command) {
      return;
    }

    const argsRaw = await vscode.window.showInputBox({
      title: "Vocode Run Command",
      prompt: "Enter args JSON array (optional)",
      placeHolder: '["/c","echo","hi"]',
      ignoreFocusOut: true,
    });

    let args: string[] = [];
    if (argsRaw && argsRaw.trim().length > 0) {
      try {
        const parsed = JSON.parse(argsRaw) as unknown;
        if (
          !Array.isArray(parsed) ||
          !parsed.every((v) => typeof v === "string")
        ) {
          throw new Error("args must be a JSON array of strings");
        }
        args = parsed;
      } catch (err) {
        void vscode.window.showErrorMessage(
          `Invalid args JSON: ${err instanceof Error ? err.message : "unknown error"}`,
        );
        return;
      }
    }

    const timeoutRaw = await vscode.window.showInputBox({
      title: "Vocode Run Command",
      prompt: "Timeout milliseconds (optional)",
      placeHolder: "10000",
      ignoreFocusOut: true,
    });

    const timeoutMs = timeoutRaw ? Number(timeoutRaw) : undefined;
    const normalizedTimeoutMs =
      timeoutMs && Number.isFinite(timeoutMs) && timeoutMs > 0
        ? timeoutMs
        : undefined;

    const params: CommandRunParams = {
      command,
      args,
      timeoutMs: normalizedTimeoutMs,
    };

    const result = await client.commandRun(params);
    if (!isCommandRunResult(result)) {
      throw new Error("Daemon returned an invalid command.run result.");
    }

    if (result.kind === "success") {
      void vscode.window.showInformationMessage(
        `Command completed (exitCode=${result.exitCode}): ${result.stdout}`,
      );
      return;
    }

    void vscode.window.showWarningMessage(
      `Command failed: ${result.failure.message}`,
    );
  },
};
