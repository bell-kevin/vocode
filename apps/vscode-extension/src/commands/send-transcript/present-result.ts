import type {
  CommandDirective,
  EditDirective,
  VoiceTranscriptResult,
} from "@vocode/protocol";
import * as vscode from "vscode";

import { runAllowedCommand } from "../../commandexec/execute-command";
import { dispatchEditResultWorkspaceEdits } from "../../edits/dispatch-workspace-edits";
import {
  type EditLocationMap,
  executeNavigationStep,
} from "../../navigation/execute-navigation-intent";
import {
  applyUndoDirective,
  beginTranscriptUndoSession,
  finalizeTranscriptUndoSessionIfEditsApplied,
  recordAppliedEditUndoPaths,
} from "../../voice/transcript-undo-ledger";

/** Applies workspace edits and surfaces per-step edit/command outcomes in the UI. */
export async function presentTranscriptResult(
  result: VoiceTranscriptResult,
  activeDocumentPath: string,
): Promise<void> {
  const editLocations: EditLocationMap = {};

  if (!result.accepted) {
    void vscode.window.showErrorMessage("Vocode: transcript was not accepted.");
    return;
  }

  beginTranscriptUndoSession();
  try {
    for (const step of result.directives ?? []) {
      switch (step.kind) {
        case "edit": {
          if (
            !(await handleEditStep(
              step.editDirective,
              activeDocumentPath,
              editLocations,
            ))
          ) {
            return;
          }
          break;
        }

        case "command": {
          if (!(await handleCommandStep(step.commandDirective))) {
            return;
          }
          break;
        }

        case "navigate": {
          try {
            await executeNavigationStep(
              step,
              activeDocumentPath,
              editLocations,
            );
          } catch (err) {
            const message =
              err instanceof Error ? err.message : "navigation failed";
            void vscode.window.showErrorMessage(
              `Vocode navigation: ${message}`,
            );
            return;
          }
          break;
        }

        case "undo": {
          if (!(await applyUndoDirective(step.undoDirective))) {
            return;
          }
          break;
        }
      }
    }
  } finally {
    finalizeTranscriptUndoSessionIfEditsApplied();
  }
}

async function handleEditStep(
  edit: EditDirective | undefined,
  activeDocumentPath: string,
  editLocations: EditLocationMap,
): Promise<boolean> {
  if (!edit) {
    void vscode.window.showWarningMessage("Vocode: missing editDirective.");
    return false;
  }
  const applyOutcome = await dispatchEditResultWorkspaceEdits(
    edit,
    activeDocumentPath,
  );
  if (!applyOutcome.ok) {
    return false;
  }
  recordAppliedEditUndoPaths(applyOutcome.undoStackOrderPaths);
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

// Command execution lives in `commandexec/execute-command.ts`.
