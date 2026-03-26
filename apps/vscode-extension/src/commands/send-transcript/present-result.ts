import type {
  VoiceTranscriptResult,
  VoiceTranscriptStepResult,
} from "@vocode/protocol";
import * as vscode from "vscode";

import { applyWorkspaceEdits } from "../apply-workspace-edits";

/** Applies workspace edits and surfaces per-step edit/command outcomes in the UI. */
export async function presentTranscriptResult(
  result: VoiceTranscriptResult,
  activeDocumentPath: string,
): Promise<void> {
  if (result.planError) {
    void vscode.window.showErrorMessage(`Vocode: ${result.planError}`);
    return;
  }

  await applyWorkspaceEdits(result, activeDocumentPath);
  presentTranscriptStepMessages(result.steps ?? []);
}

function presentTranscriptStepMessages(
  steps: VoiceTranscriptStepResult[],
): void {
  for (const step of steps) {
    if (step.kind === "edit" && step.editResult?.kind === "failure") {
      void vscode.window.showErrorMessage(
        `Vocode edit: ${step.editResult.failure.message}`,
      );
    }

    if (step.kind === "run_command" && step.commandResult?.kind === "success") {
      const line = step.commandResult.stdout.trim();
      if (line.length > 0) {
        void vscode.window.showInformationMessage(`Vocode: ${line}`);
      }
    }

    if (step.kind === "run_command" && step.commandResult?.kind === "failure") {
      void vscode.window.showErrorMessage(
        `Vocode command: ${step.commandResult.failure.message}`,
      );
    }
  }
}
