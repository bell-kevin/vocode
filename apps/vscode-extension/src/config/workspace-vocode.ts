import * as vscode from "vscode";

/** Max keywords merged from all workspace `.vocode` files (before JSON env to sidecar). */
const MAX_WORKSPACE_STT_KEYWORDS = 80;

export type WorkspaceVocodeShape = {
  sttKeywords?: unknown;
  /** Builtin prompt skill ids, e.g. "react-native-expo". Opt-in only — omit or [] for no stack addenda. */
  skills?: unknown;
  /** Appended to scoped edit / file-create model prompts. */
  promptAddendum?: unknown;
};

export type WorkspaceVocodeTranscriptHints = {
  /** True if any `.vocode` file contained a `skills` array (even empty). */
  skillsExplicit: boolean;
  skillIds: string[];
  promptAddendum: string;
};

/**
 * Reads `.vocode` at each workspace folder root (JSON), merges `sttKeywords` string entries in
 * folder order, de-duplicated case-insensitively.
 */
export async function readWorkspaceSttKeywords(): Promise<string[]> {
  const folders = vscode.workspace.workspaceFolders;
  if (!folders?.length) {
    return [];
  }
  const merged: string[] = [];
  const seen = new Set<string>();
  for (const folder of folders) {
    const uri = vscode.Uri.joinPath(folder.uri, ".vocode");
    try {
      const data = await vscode.workspace.fs.readFile(uri);
      const text = new TextDecoder("utf-8").decode(data);
      let parsed: unknown;
      try {
        parsed = JSON.parse(text) as unknown;
      } catch {
        console.warn(
          `[vocode] Invalid JSON in ${uri.fsPath}; ignoring workspace STT keywords.`,
        );
        continue;
      }
      if (!parsed || typeof parsed !== "object") {
        continue;
      }
      const k = (parsed as WorkspaceVocodeShape).sttKeywords;
      if (!Array.isArray(k)) {
        continue;
      }
      for (const item of k) {
        if (typeof item !== "string") {
          continue;
        }
        const t = item.trim();
        if (!t) {
          continue;
        }
        const lk = t.toLowerCase();
        if (seen.has(lk)) {
          continue;
        }
        seen.add(lk);
        merged.push(t);
        if (merged.length >= MAX_WORKSPACE_STT_KEYWORDS) {
          return merged;
        }
      }
    } catch {
      // Missing file or unreadable — skip this folder.
    }
  }
  return merged;
}

const MAX_WORKSPACE_SKILL_IDS = 32;

function mergeSkillsFromVocodeObject(
  o: WorkspaceVocodeShape,
  skillOrder: string[],
  skillSeen: Set<string>,
): void {
  if (!("skills" in o) || !Array.isArray(o.skills)) {
    return;
  }
  for (const item of o.skills) {
    if (typeof item !== "string") {
      continue;
    }
    const t = item.trim();
    if (!t) {
      continue;
    }
    const lk = t.toLowerCase();
    if (skillSeen.has(lk)) {
      continue;
    }
    skillSeen.add(lk);
    skillOrder.push(t);
    if (skillOrder.length >= MAX_WORKSPACE_SKILL_IDS) {
      break;
    }
  }
}

function appendPromptAddendumFromVocodeObject(
  o: WorkspaceVocodeShape,
  addendumParts: string[],
): void {
  if (typeof o.promptAddendum !== "string") {
    return;
  }
  const x = o.promptAddendum.trim();
  if (x) {
    addendumParts.push(x);
  }
}

/**
 * Reads `.vocode` transcript hints: `skills` and `promptAddendum` from each workspace folder (merged).
 * - `skills`: same-folder order, then next folder; de-duplicated case-insensitively. Builtin prompt add-ons
 *   (e.g. react-native-expo) are opt-in only — omit `skills` or use [] for none.
 * - If any file defines `skills` as an array (including []), `skillsExplicit` is true and we send
 *   `workspaceSkillIds` (possibly empty). If no file has a `skills` key, we omit the field on the RPC.
 */
export async function readWorkspaceVocodeTranscriptHints(): Promise<WorkspaceVocodeTranscriptHints> {
  const folders = vscode.workspace.workspaceFolders;
  if (!folders?.length) {
    return { skillsExplicit: false, skillIds: [], promptAddendum: "" };
  }
  const skillOrder: string[] = [];
  const skillSeen = new Set<string>();
  const addendumParts: string[] = [];
  let skillsExplicit = false;

  for (const folder of folders) {
    const uri = vscode.Uri.joinPath(folder.uri, ".vocode");
    try {
      const data = await vscode.workspace.fs.readFile(uri);
      const text = new TextDecoder("utf-8").decode(data);
      let parsed: unknown;
      try {
        parsed = JSON.parse(text) as unknown;
      } catch {
        continue;
      }
      if (!parsed || typeof parsed !== "object") {
        continue;
      }
      const o = parsed as WorkspaceVocodeShape;
      if ("skills" in o && Array.isArray(o.skills)) {
        skillsExplicit = true;
        mergeSkillsFromVocodeObject(o, skillOrder, skillSeen);
      }
      appendPromptAddendumFromVocodeObject(o, addendumParts);
    } catch {
      // missing / unreadable
    }
  }

  return {
    skillsExplicit,
    skillIds: skillOrder,
    promptAddendum: addendumParts.join("\n\n"),
  };
}

/**
 * Creates `.vocode` in the first workspace folder with `{"sttKeywords":[]}` when none exists.
 */
export async function createWorkspaceVocodeFile(): Promise<void> {
  const folders = vscode.workspace.workspaceFolders;
  if (!folders?.length) {
    void vscode.window.showErrorMessage(
      "Open a folder or multi-root workspace first to create .vocode.",
    );
    return;
  }
  const uri = vscode.Uri.joinPath(folders[0].uri, ".vocode");
  try {
    await vscode.workspace.fs.stat(uri);
    void vscode.window.showWarningMessage(
      `.vocode already exists: ${uri.fsPath}`,
    );
    await vscode.window.showTextDocument(uri);
    return;
  } catch {
    // create
  }
  const body = `${JSON.stringify({ sttKeywords: [] }, null, 2)}\n`;
  await vscode.workspace.fs.writeFile(uri, new TextEncoder().encode(body));
  void vscode.window.showInformationMessage(
    "Created .vocode with sttKeywords. Edit the file, then use “Apply changes and restart”.",
  );
  await vscode.window.showTextDocument(uri);
}
