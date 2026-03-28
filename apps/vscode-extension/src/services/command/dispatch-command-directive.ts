import type { CommandDirective } from "@vocode/protocol";
import * as vscode from "vscode";

import { runAllowedCommand } from "./execute-command";

/** Runs one allowed command directive (extension executes; daemon validated shape). */
export async function dispatchCommandDirective(
  params: CommandDirective | undefined,
): Promise<boolean> {
  if (!params) {
    void vscode.window.showWarningMessage("Vocode: missing commandDirective.");
    return false;
  }
  const outcome = await runAllowedCommand(params);
  if (!outcome.ok) {
    void vscode.window.showErrorMessage(`Vocode command: ${outcome.message}`);
    return false;
  }
  const line = outcome.stdout.trim();
  if (line.length > 0) {
    void vscode.window.showInformationMessage(`Vocode: ${line}`);
  }
  return true;
}
