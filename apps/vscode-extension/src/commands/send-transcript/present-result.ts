import type {
  CommandRunParams,
  EditApplyResult,
  VoiceTranscriptResult,
} from "@vocode/protocol";
import * as vscode from "vscode";

import { runAllowedCommand } from "../../commandexec/execute-command";
import { applyEditResultWorkspaceEdits } from "../../edits/apply-workspace-edits";
import {
  type EditLocationMap,
  executeNavigationStep,
} from "../../navigation/execute-navigation-intent";

/** Applies workspace edits and surfaces per-step edit/command outcomes in the UI. */
export async function presentTranscriptResult(
  result: VoiceTranscriptResult,
  activeDocumentPath: string,
): Promise<void> {
  const editLocations: EditLocationMap = {};

  if (result.planError) {
    void vscode.window.showErrorMessage(`Vocode: ${result.planError}`);
    return;
  }

  for (const step of result.steps ?? []) {
    switch (step.kind) {
      case "edit": {
        if (
          !(await handleEditStep(
            step.editResult,
            activeDocumentPath,
            editLocations,
          ))
        ) {
          return;
        }
        break;
      }

      case "run_command": {
        if (!(await handleCommandStep(step.commandParams))) {
          return;
        }
        break;
      }

      case "navigate": {
        try {
          await executeNavigationStep(step, activeDocumentPath, editLocations);
        } catch (err) {
          const message =
            err instanceof Error ? err.message : "navigation failed";
          void vscode.window.showErrorMessage(`Vocode navigation: ${message}`);
          return;
        }
        break;
      }
    }
  }
}

async function handleEditStep(
  edit: EditApplyResult | undefined,
  activeDocumentPath: string,
  editLocations: EditLocationMap,
): Promise<boolean> {
  if (!edit) {
    void vscode.window.showWarningMessage("Vocode: missing editResult.");
    return false;
  }
  if (edit.kind === "failure") {
    void vscode.window.showErrorMessage(`Vocode edit: ${edit.failure.message}`);
    return false;
  }
  const applyOutcome = await applyEditResultWorkspaceEdits(
    edit,
    activeDocumentPath,
  );
  if (!applyOutcome.ok) {
    return false;
  }
  for (const loc of applyOutcome.appliedEdits) {
    if (!loc.editId) continue;
    editLocations[loc.editId] = {
      path: loc.path,
      selectionStart: loc.selectionStart,
      selectionEnd: loc.selectionEnd,
    };
  }
  return true;
}

async function handleCommandStep(
  params: CommandRunParams | undefined,
): Promise<boolean> {
  if (!params) {
    void vscode.window.showWarningMessage("Vocode: missing commandParams.");
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

// Command execution lives in `commandexec/execute-command.ts`.
