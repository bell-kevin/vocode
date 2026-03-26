import * as vscode from "vscode";

import { applyWorkspaceEdits } from "./apply-workspace-edits";
import type { CommandDefinition } from "./types";

export const sendTranscriptCommand: CommandDefinition = {
  id: "vocode.sendTranscript",
  requiresDaemon: true,
  run: async (client, services) => {
    if (!services.voiceSession.isRunning()) {
      void vscode.window.showWarningMessage(
        "Voice is not active. Run 'Vocode: Start Voice' first.",
      );
      return;
    }

    const editor = vscode.window.activeTextEditor;
    if (!editor) {
      void vscode.window.showWarningMessage(
        "Open a text editor so Vocode can run edit steps against the active file.",
      );
      return;
    }

    const text = await vscode.window.showInputBox({
      title: "Vocode Voice Transcript",
      prompt: "Enter transcript text to send to the daemon",
      placeHolder: "Refactor this function to handle empty input safely",
      ignoreFocusOut: true,
    });

    if (!services.voiceSession.isRunning()) {
      services.voiceStatus.setIdle();
      return;
    }

    const trimmedText = text?.trim();
    if (!trimmedText) {
      return;
    }

    const activePath = editor.document.uri.fsPath;

    try {
      services.voiceStatus.setProcessing();
      const result = await client.voiceTranscript({
        text: trimmedText,
        activeFile: activePath,
      });

      if (result.planError) {
        void vscode.window.showErrorMessage(`Vocode: ${result.planError}`);
      } else {
        await applyWorkspaceEdits(result, activePath);
        for (const step of result.steps ?? []) {
          if (step.kind === "edit" && step.editResult?.kind === "failure") {
            void vscode.window.showErrorMessage(
              `Vocode edit: ${step.editResult.failure.message}`,
            );
          }
          if (
            step.kind === "run_command" &&
            step.commandResult?.kind === "success"
          ) {
            const line = step.commandResult.stdout.trim();
            if (line.length > 0) {
              void vscode.window.showInformationMessage(`Vocode: ${line}`);
            }
          }
          if (
            step.kind === "run_command" &&
            step.commandResult?.kind === "failure"
          ) {
            void vscode.window.showErrorMessage(
              `Vocode command: ${step.commandResult.failure.message}`,
            );
          }
        }
      }
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to send transcript.";
      void vscode.window.showWarningMessage(message);
    } finally {
      if (services.voiceSession.isRunning()) {
        services.voiceStatus.setListening();
      } else {
        services.voiceStatus.setIdle();
      }
    }
  },
};
