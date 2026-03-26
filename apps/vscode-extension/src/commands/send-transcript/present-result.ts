import type { VoiceTranscriptResult } from "@vocode/protocol";
import * as vscode from "vscode";

import { runAllowedCommand } from "../../commandexec/execute-command";
import { applyEditResultWorkspaceEdits } from "../../edits/apply-workspace-edits";

/** Applies workspace edits and surfaces per-step edit/command outcomes in the UI. */
export async function presentTranscriptResult(
  result: VoiceTranscriptResult,
  activeDocumentPath: string,
): Promise<void> {
  if (result.planError) {
    void vscode.window.showErrorMessage(`Vocode: ${result.planError}`);
    return;
  }

  for (const step of result.steps ?? []) {
    switch (step.kind) {
      case "edit": {
        const edit = step.editResult;
        if (!edit) {
          void vscode.window.showWarningMessage("Vocode: missing editResult.");
          return;
        }

        if (edit.kind === "failure") {
          void vscode.window.showErrorMessage(
            `Vocode edit: ${edit.failure.message}`,
          );
          return;
        }

        if (!(await applyEditResultWorkspaceEdits(edit, activeDocumentPath))) {
          return;
        }
        break;
      }

      case "run_command": {
        const params = step.commandParams;
        if (!params) {
          void vscode.window.showWarningMessage(
            "Vocode: missing commandParams.",
          );
          return;
        }

        const outcome = await runAllowedCommand(params);
        if (!outcome.ok) {
          void vscode.window.showErrorMessage(
            `Vocode command: ${outcome.message}`,
          );
          return;
        }

        const line = outcome.stdout.trim();
        if (line.length > 0) {
          void vscode.window.showInformationMessage(`Vocode: ${line}`);
        }
        break;
      }
    }
  }
}

// Command execution lives in `commandexec/execute-command.ts`.
