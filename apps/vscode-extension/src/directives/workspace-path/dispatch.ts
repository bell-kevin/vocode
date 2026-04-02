import * as fs from "node:fs/promises";
import * as path from "node:path";
import type { VoiceTranscriptDirective } from "@vocode/protocol";
import * as vscode from "vscode";

import type { DirectiveDispatchOutcome } from "../dispatch";

function workspaceRootPath(): string | undefined {
  return vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
}

/** True if target is the workspace root or a path inside it (after resolve). */
function isUnderWorkspaceRoot(root: string, targetPath: string): boolean {
  const rootResolved = path.resolve(root);
  const targetResolved = path.resolve(targetPath);
  if (targetResolved === rootResolved) {
    return true;
  }
  const prefix = rootResolved.endsWith(path.sep)
    ? rootResolved
    : `${rootResolved}${path.sep}`;
  return targetResolved.startsWith(prefix);
}

/**
 * Applies delete_file / move_path / create_folder under the first workspace folder (host-side jail).
 */
export async function dispatchWorkspacePath(
  d: VoiceTranscriptDirective,
): Promise<DirectiveDispatchOutcome> {
  const root = workspaceRootPath();
  if (!root) {
    return { ok: false, message: "No workspace folder is open." };
  }

  switch (d.kind) {
    case "delete_file": {
      const p = d.deleteFileDirective?.path?.trim() ?? "";
      if (!p || !isUnderWorkspaceRoot(root, p)) {
        return {
          ok: false,
          message: "delete_file: path missing or outside workspace.",
        };
      }
      try {
        await fs.unlink(p);
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        return { ok: false, message: `delete_file failed: ${msg}` };
      }
      return { ok: true };
    }
    case "move_path": {
      const from = d.movePathDirective?.from?.trim() ?? "";
      const to = d.movePathDirective?.to?.trim() ?? "";
      if (
        !from ||
        !to ||
        !isUnderWorkspaceRoot(root, from) ||
        !isUnderWorkspaceRoot(root, to)
      ) {
        return {
          ok: false,
          message: "move_path: paths missing or outside workspace.",
        };
      }
      try {
        await fs.mkdir(path.dirname(to), { recursive: true });
        await fs.rename(from, to);
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        return { ok: false, message: `move_path failed: ${msg}` };
      }
      return { ok: true };
    }
    case "create_folder": {
      const p = d.createFolderDirective?.path?.trim() ?? "";
      if (!p || !isUnderWorkspaceRoot(root, p)) {
        return {
          ok: false,
          message: "create_folder: path missing or outside workspace.",
        };
      }
      try {
        await fs.mkdir(p, { recursive: true });
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        return { ok: false, message: `create_folder failed: ${msg}` };
      }
      return { ok: true };
    }
    default:
      return {
        ok: false,
        message: "internal: not a workspace path directive",
      };
  }
}
