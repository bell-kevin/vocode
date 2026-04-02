import path from "node:path";
import * as vscode from "vscode";

/**
 * True when at least one workspace folder is open (VS Code–aligned gate for explorer-style flows).
 */
export function transcriptWorkspaceFolderOpen(): boolean {
  return (vscode.workspace.workspaceFolders?.length ?? 0) > 0;
}

/**
 * Directory the daemon should use to resolve relative paths and workspace-scoped search.
 * Single-folder: that folder. Multi-root: the workspace folder whose path is the longest
 * prefix of the active file (editor-based root); if none match, first folder. No folder:
 * parent of the active file when known.
 */
export function transcriptWorkspaceRoot(
  activeFilePath: string,
): string | undefined {
  const folders = vscode.workspace.workspaceFolders;
  if (folders?.length === 1) {
    return folders[0].uri.fsPath.trim();
  }
  if (folders && folders.length > 1) {
    const active = activeFilePath.trim();
    if (!active) {
      return folders[0].uri.fsPath.trim();
    }
    let best: string | undefined;
    let bestLen = -1;
    for (const f of folders) {
      const root = f.uri.fsPath.trim();
      if (!root) continue;
      const sep = path.sep;
      const isPrefix =
        active === root ||
        active.startsWith(root.endsWith(sep) ? root : root + sep);
      if (isPrefix && root.length > bestLen) {
        best = root;
        bestLen = root.length;
      }
    }
    return best ?? folders[0].uri.fsPath.trim();
  }
  const active = activeFilePath.trim();
  if (!active) {
    return undefined;
  }
  return path.dirname(active);
}
