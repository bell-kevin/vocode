import crypto from "node:crypto";
import path from "node:path";
import type { EditDirective } from "@vocode/protocol";
import * as vscode from "vscode";

import { resolveReplaceBetweenAnchors } from "./dispatch-edit-helpers";
export interface AppliedEditLocation {
  editId?: string;
  path: string;
  selectionStart?: vscode.Position;
  selectionEnd?: vscode.Position;
}

export interface ApplyEditResultWorkspaceOutcome {
  ok: boolean;
  appliedEdits: AppliedEditLocation[];
  /** One path per successful `workspace.applyEdit` (reverse order for transcript undo). */
  undoStackOrderPaths: string[];
  message?: string;
}

function toAbsolutePath(
  targetPath: string,
  activeDocumentPath: string,
): string {
  if (path.isAbsolute(targetPath)) {
    return targetPath;
  }
  const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
  if (workspaceRoot) {
    return path.resolve(workspaceRoot, targetPath);
  }
  return path.resolve(path.dirname(activeDocumentPath), targetPath);
}

/** UTF-16 length; matches VS Code document offsets for typical source. */
function utf16Len(s: string): number {
  return s.length;
}

/**
 * Selects [startOff, endOff) in any visible editor for this URI so the user sees what changed
 * (e.g. new lines above a function plus the updated range), instead of a stale search selection.
 */
function selectDocumentRangeInVisibleEditors(
  uri: vscode.Uri,
  doc: vscode.TextDocument,
  startOff: number,
  endOffExclusive: number,
): void {
  const max = doc.getText().length;
  const a = Math.max(0, Math.min(startOff, max));
  const b = Math.max(a, Math.min(endOffExclusive, max));
  const start = doc.positionAt(a);
  const end = doc.positionAt(b);
  const range = new vscode.Range(start, end);
  const active = vscode.window.activeTextEditor;
  const shouldReveal =
    active !== undefined && active.document.uri.toString() === uri.toString();
  for (const ed of vscode.window.visibleTextEditors) {
    if (ed.document.uri.toString() !== uri.toString()) {
      continue;
    }
    ed.selection = new vscode.Selection(start, end);
    if (shouldReveal && ed === active) {
      ed.revealRange(
        range,
        vscode.TextEditorRevealType.InCenterIfOutsideViewport,
      );
    }
  }
}

/**
 * Applies one edit directive's actions by opening each target document and
 * submitting workspace edits. Also returns per-action edit locations for
 * follow-up navigation (e.g. reveal_edit).
 */
export async function dispatchEditResultWorkspaceEdit(
  editDirective: EditDirective,
  activeDocumentPath: string,
): Promise<ApplyEditResultWorkspaceOutcome> {
  if (editDirective.kind !== "success" || editDirective.actions.length === 0) {
    return { ok: true, appliedEdits: [], undoStackOrderPaths: [] };
  }

  const appliedEdits: AppliedEditLocation[] = [];
  const undoStackOrderPaths: string[] = [];

  for (const action of editDirective.actions) {
    const actionPath = toAbsolutePath(action.path, activeDocumentPath);
    const wsEdit = new vscode.WorkspaceEdit();

    let replaceStartOffset: number | undefined;
    let replaceNewText = "";

    if (action.kind === "replace_between_anchors") {
      const targetDoc = await vscode.workspace.openTextDocument(actionPath);
      const text = targetDoc.getText();
      const { startOffset, endOffset } = resolveReplaceBetweenAnchors(
        text,
        action,
      );
      const startPos = targetDoc.positionAt(startOffset);
      const endPos = targetDoc.positionAt(endOffset);
      replaceStartOffset = startOffset;
      replaceNewText = action.newText;
      wsEdit.replace(
        targetDoc.uri,
        new vscode.Range(startPos, endPos),
        action.newText,
      );
      appliedEdits.push({
        editId: action.editId,
        path: actionPath,
        selectionStart: startPos,
        selectionEnd: startPos,
      });
    } else if (action.kind === "replace_range") {
      const uri = vscode.Uri.file(actionPath);
      const targetDoc = await vscode.workspace.openTextDocument(uri);
      const startPos = new vscode.Position(
        action.range.startLine,
        action.range.startChar,
      );
      const endPos = new vscode.Position(
        action.range.endLine,
        action.range.endChar,
      );
      if (action.expectedSha256) {
        const oldText = targetDoc.getText(new vscode.Range(startPos, endPos));
        const got = crypto.createHash("sha256").update(oldText).digest("hex");
        if (got !== action.expectedSha256) {
          return {
            ok: false,
            appliedEdits,
            undoStackOrderPaths,
            message: `stale_range: expectedSha256 mismatch (expected=${action.expectedSha256} got=${got})`,
          };
        }
      }
      replaceStartOffset = targetDoc.offsetAt(startPos);
      replaceNewText = action.newText;
      wsEdit.replace(uri, new vscode.Range(startPos, endPos), action.newText);
      appliedEdits.push({
        editId: action.editId,
        path: actionPath,
        selectionStart: startPos,
        selectionEnd: startPos,
      });
    } else if (action.kind === "replace_file") {
      const uri = vscode.Uri.file(actionPath);
      const fullRange = new vscode.Range(
        new vscode.Position(0, 0),
        // Use a large range to cover the entire document; VS Code will clamp it.
        new vscode.Position(Number.MAX_SAFE_INTEGER, Number.MAX_SAFE_INTEGER),
      );
      wsEdit.replace(uri, fullRange, action.content);
      appliedEdits.push({
        editId: action.editId,
        path: actionPath,
        selectionStart: new vscode.Position(0, 0),
        selectionEnd: new vscode.Position(0, 0),
      });
    } else if (action.kind === "create_file") {
      const uri = vscode.Uri.file(actionPath);
      wsEdit.createFile(uri, { overwrite: true, ignoreIfExists: false });
      wsEdit.insert(uri, new vscode.Position(0, 0), action.content);
      appliedEdits.push({
        editId: action.editId,
        path: actionPath,
        selectionStart: new vscode.Position(0, 0),
        selectionEnd: new vscode.Position(0, 0),
      });
    } else if (action.kind === "append_to_file") {
      const uri = vscode.Uri.file(actionPath);
      const targetDoc = await vscode.workspace.openTextDocument(uri);
      const endOff = targetDoc.getText().length;
      const end = targetDoc.positionAt(endOff);
      replaceStartOffset = endOff;
      replaceNewText = action.text;
      wsEdit.insert(uri, end, action.text);
      appliedEdits.push({
        editId: action.editId,
        path: actionPath,
        selectionStart: end,
        selectionEnd: end,
      });
    } else {
      continue;
    }

    const applied = await vscode.workspace.applyEdit(wsEdit);
    if (!applied) {
      return {
        ok: false,
        appliedEdits,
        undoStackOrderPaths,
        message: "workspace.applyEdit returned false",
      };
    }

    const savedDocument = await vscode.workspace.openTextDocument(actionPath);
    const saved = await savedDocument.save();
    if (!saved) {
      return {
        ok: false,
        appliedEdits,
        undoStackOrderPaths,
        message: "failed to save edited document",
      };
    }

    const lastApplied = appliedEdits[appliedEdits.length - 1];
    if (
      replaceStartOffset !== undefined &&
      (action.kind === "replace_range" ||
        action.kind === "replace_between_anchors" ||
        action.kind === "append_to_file")
    ) {
      const endSel = replaceStartOffset + utf16Len(replaceNewText);
      const selStart = savedDocument.positionAt(replaceStartOffset);
      const selEnd = savedDocument.positionAt(endSel);
      lastApplied.selectionStart = selStart;
      lastApplied.selectionEnd = selEnd;
      selectDocumentRangeInVisibleEditors(
        savedDocument.uri,
        savedDocument,
        replaceStartOffset,
        endSel,
      );
    } else if (
      action.kind === "replace_file" ||
      action.kind === "create_file"
    ) {
      const full = savedDocument.getText().length;
      const selStart = savedDocument.positionAt(0);
      const selEnd = savedDocument.positionAt(full);
      lastApplied.selectionStart = selStart;
      lastApplied.selectionEnd = selEnd;
      selectDocumentRangeInVisibleEditors(
        savedDocument.uri,
        savedDocument,
        0,
        full,
      );
    }

    undoStackOrderPaths.push(actionPath);
  }

  return { ok: true, appliedEdits, undoStackOrderPaths };
}
