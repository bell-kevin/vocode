import * as vscode from "vscode";

/**
 * Collapses every non-empty selection in the active editor to a caret at each selection's active end.
 * Call when the user leaves workspace/file search so ROOT classification sees hasNonemptySelection false
 * (search hits often leave a range selected in the editor).
 */
export function collapseNonemptySelectionsInActiveEditor(): void {
  const ed = vscode.window.activeTextEditor;
  if (!ed) {
    return;
  }
  let changed = false;
  const next = ed.selections.map((sel) => {
    if (sel.isEmpty) {
      return sel;
    }
    changed = true;
    const p = sel.active;
    return new vscode.Selection(p, p);
  });
  if (changed) {
    ed.selections = next;
  }
}
