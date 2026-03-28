import path from "node:path";
import type { VoiceTranscriptDirective } from "@vocode/protocol";
import * as vscode from "vscode";

export interface EditLocationMap {
  [editId: string]: {
    path: string;
    selectionStart?: vscode.Position;
    selectionEnd?: vscode.Position;
  };
}

function openDoc(path: string): Thenable<vscode.TextEditor> {
  return vscode.workspace
    .openTextDocument(path)
    .then((doc) => vscode.window.showTextDocument(doc, { preview: false }));
}

function resolvePath(targetPath: string, activeDocumentPath: string): string {
  if (path.isAbsolute(targetPath)) return targetPath;
  const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
  if (workspaceRoot) {
    return path.resolve(workspaceRoot, targetPath);
  }
  return path.resolve(path.dirname(activeDocumentPath), targetPath);
}

export async function executeNavigationDirective(
  transcriptDirective: VoiceTranscriptDirective,
  activeDocumentPath: string,
  editLocations?: EditLocationMap,
): Promise<void> {
  const navigationDirective = transcriptDirective.navigationDirective;
  if (!navigationDirective) {
    throw new Error("missing navigationDirective");
  }
  if (navigationDirective.kind === "noop") {
    return;
  }
  const nav = navigationDirective.action;
  if (!nav) {
    throw new Error("missing navigationDirective.action");
  }

  switch (nav.kind) {
    case "open_file": {
      await openDoc(resolvePath(nav.openFile.path, activeDocumentPath));
      return;
    }
    case "reveal_symbol": {
      const targetPath = resolvePath(
        nav.revealSymbol.path || activeDocumentPath,
        activeDocumentPath,
      );
      const editor = await openDoc(targetPath);
      const symbols = (await vscode.commands.executeCommand(
        "vscode.executeDocumentSymbolProvider",
        editor.document.uri,
      )) as vscode.DocumentSymbol[] | undefined;
      const targetName = nav.revealSymbol.symbolName.trim().toLowerCase();
      const symbol = symbols?.find(
        (s) => s.name.trim().toLowerCase() === targetName,
      );
      if (symbol) {
        editor.selection = new vscode.Selection(
          symbol.selectionRange.start,
          symbol.selectionRange.start,
        );
        editor.revealRange(symbol.range, vscode.TextEditorRevealType.InCenter);
      }
      return;
    }
    case "move_cursor": {
      const targetPath = resolvePath(
        nav.moveCursor.target.path || activeDocumentPath,
        activeDocumentPath,
      );
      const editor = await openDoc(targetPath);
      const pos = new vscode.Position(
        nav.moveCursor.target.line,
        nav.moveCursor.target.char,
      );
      editor.selection = new vscode.Selection(pos, pos);
      editor.revealRange(
        new vscode.Range(pos, pos),
        vscode.TextEditorRevealType.InCenter,
      );
      return;
    }
    case "select_range": {
      const targetPath = resolvePath(
        nav.selectRange.target.path || activeDocumentPath,
        activeDocumentPath,
      );
      const editor = await openDoc(targetPath);
      const start = new vscode.Position(
        nav.selectRange.target.startLine,
        nav.selectRange.target.startChar,
      );
      const end = new vscode.Position(
        nav.selectRange.target.endLine,
        nav.selectRange.target.endChar,
      );
      editor.selection = new vscode.Selection(start, end);
      editor.revealRange(
        new vscode.Range(start, end),
        vscode.TextEditorRevealType.InCenter,
      );
      return;
    }
    case "reveal_edit": {
      const loc = editLocations?.[nav.revealEdit.editId];
      if (!loc) {
        void vscode.window.showWarningMessage(
          `Vocode: reveal_edit target not found (${nav.revealEdit.editId}).`,
        );
        return;
      }
      const editor = await openDoc(resolvePath(loc.path, activeDocumentPath));
      if (loc.selectionStart && loc.selectionEnd) {
        editor.selection = new vscode.Selection(
          loc.selectionStart,
          loc.selectionEnd,
        );
        editor.revealRange(
          new vscode.Range(loc.selectionStart, loc.selectionEnd),
          vscode.TextEditorRevealType.InCenter,
        );
      }
      return;
    }
  }
}
