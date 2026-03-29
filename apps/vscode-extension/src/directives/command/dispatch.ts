import type { CommandDirective } from "@vocode/protocol";
import * as vscode from "vscode";

import { runAllowedCommand } from "./execute-command";

/** Runs one allowed command directive (extension executes; daemon validated shape). */
export async function dispatchCommand(
  params: CommandDirective | undefined,
): Promise<boolean> {
  if (!params) {
    return false;
  }
  const outcome = await runAllowedCommand(params);
  if (!outcome.ok) {
    return false;
  }
  const line = outcome.stdout.trim();
  if (line.length > 0) {
    void vscode.window.showInformationMessage(`Vocode: ${line}`);
  }
  return true;
}
