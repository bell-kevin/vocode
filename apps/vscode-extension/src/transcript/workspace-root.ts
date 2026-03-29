import path from "node:path";
import * as vscode from "vscode";

/**
 * Directory the daemon should use to resolve relative paths and workspace-scoped search.
 * With a folder workspace open, that folder wins; with only a loose file, use its parent
 * (matches extension edit/navigation path resolution).
 */
export function transcriptWorkspaceRoot(
  activeFilePath: string,
): string | undefined {
  const folder = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath?.trim();
  if (folder) {
    return folder;
  }
  const active = activeFilePath.trim();
  if (!active) {
    return undefined;
  }
  return path.dirname(active);
}
