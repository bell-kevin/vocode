import type { UndoDirective } from "@vocode/protocol";
import * as vscode from "vscode";

let lastTranscriptUndoPaths: string[] = [];
let currentSessionUndoPaths: string[] = [];

/** Start recording workspace edit paths for the current voice transcript apply. */
export function beginTranscriptUndoSession(): void {
  currentSessionUndoPaths = [];
}

/** Record paths from one successful edit directive (apply order). */
export function recordAppliedEditUndoPaths(paths: string[]): void {
  if (paths.length === 0) {
    return;
  }
  currentSessionUndoPaths.push(...paths);
}

/**
 * If this session applied any workspace edits, replace the saved transcript undo stack.
 * Sessions with only navigation/commands leave the previous stack intact.
 */
export function finalizeTranscriptUndoSessionIfEditsApplied(): void {
  if (currentSessionUndoPaths.length > 0) {
    lastTranscriptUndoPaths = [...currentSessionUndoPaths];
  }
}

async function undoOnceAtPath(fsPath: string): Promise<void> {
  const uri = vscode.Uri.file(fsPath);
  const doc = await vscode.workspace.openTextDocument(uri);
  await vscode.window.showTextDocument(doc, { preview: false });
  await vscode.commands.executeCommand("undo");
}

/** Applies an undo directive from the daemon (voice intent), not the command palette. */
export async function applyUndoDirective(
  directive: UndoDirective | undefined,
): Promise<boolean> {
  if (!directive) {
    return false;
  }

  if (directive.scope === "last_edit") {
    if (!vscode.window.activeTextEditor) {
      return false;
    }
    await vscode.commands.executeCommand("undo");
    return true;
  }

  if (directive.scope === "last_transcript") {
    const stack =
      currentSessionUndoPaths.length > 0
        ? [...currentSessionUndoPaths]
        : [...lastTranscriptUndoPaths];
    if (stack.length === 0) {
      return true;
    }
    for (let i = stack.length - 1; i >= 0; i--) {
      const p = stack[i];
      if (p !== undefined) {
        await undoOnceAtPath(p);
      }
    }
    currentSessionUndoPaths = [];
    lastTranscriptUndoPaths = [];
    return true;
  }

  return false;
}
