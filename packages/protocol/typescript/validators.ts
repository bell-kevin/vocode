import type {
  CommandDirective,
  EditAction,
  EditDirective,
  PingResult,
  VoiceTranscriptCompletion,
  VoiceTranscriptDirective,
} from "./types.generated";

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function hasOnlyKeys(
  value: Record<string, unknown>,
  allowedKeys: string[],
): boolean {
  const allowed = new Set(allowedKeys);
  return Object.keys(value).every((key) => allowed.has(key));
}

function isAnchor(value: unknown): value is { before: string; after: string } {
  return (
    isRecord(value) &&
    hasOnlyKeys(value, ["before", "after"]) &&
    typeof value.before === "string" &&
    typeof value.after === "string"
  );
}

export function isPingResult(value: unknown): value is PingResult {
  return isRecord(value) && value.message === "pong";
}

export function isEditAction(value: unknown): value is EditAction {
  const isReplaceBetweenAnchors =
    isRecord(value) &&
    hasOnlyKeys(value, ["kind", "path", "anchor", "newText", "editId"]) &&
    value.kind === "replace_between_anchors" &&
    typeof value.path === "string" &&
    isAnchor(value.anchor) &&
    typeof value.newText === "string" &&
    (value.editId === undefined || typeof value.editId === "string");
  if (isReplaceBetweenAnchors) return true;

  const isCreateFile =
    isRecord(value) &&
    hasOnlyKeys(value, ["kind", "path", "content", "editId"]) &&
    value.kind === "create_file" &&
    typeof value.path === "string" &&
    typeof value.content === "string" &&
    (value.editId === undefined || typeof value.editId === "string");
  if (isCreateFile) return true;

  return (
    isRecord(value) &&
    hasOnlyKeys(value, ["kind", "path", "text", "editId"]) &&
    value.kind === "append_to_file" &&
    typeof value.path === "string" &&
    typeof value.text === "string" &&
    (value.editId === undefined || typeof value.editId === "string")
  );
}

export function isEditDirective(value: unknown): value is EditDirective {
  if (!isRecord(value) || typeof value.kind !== "string") {
    return false;
  }

  switch (value.kind) {
    case "success":
      return (
        hasOnlyKeys(value, ["kind", "actions"]) &&
        Array.isArray(value.actions) &&
        value.actions.every(isEditAction)
      );
    case "noop":
      return (
        hasOnlyKeys(value, ["kind", "reason"]) &&
        typeof value.reason === "string"
      );
    default:
      return false;
  }
}

function isCommandDirective(value: unknown): value is CommandDirective {
  if (!isRecord(value)) {
    return false;
  }

  if (
    !hasOnlyKeys(value as Record<string, unknown>, [
      "command",
      "args",
      "timeoutMs",
    ])
  ) {
    return false;
  }

  if (typeof (value as Record<string, unknown>).command !== "string") {
    return false;
  }

  const args = (value as Record<string, unknown>).args;
  if (args !== undefined) {
    if (!Array.isArray(args) || !args.every((x) => typeof x === "string")) {
      return false;
    }
  }

  const timeoutMs = (value as Record<string, unknown>).timeoutMs;
  if (timeoutMs !== undefined && typeof timeoutMs !== "number") {
    return false;
  }

  return true;
}

function isNavigationAction(value: unknown): boolean {
  if (!isRecord(value) || typeof value.kind !== "string") return false;
  switch (value.kind) {
    case "open_file":
      return (
        hasOnlyKeys(value, ["kind", "openFile"]) &&
        isRecord(value.openFile) &&
        hasOnlyKeys(value.openFile, ["path"]) &&
        typeof value.openFile.path === "string"
      );
    case "reveal_symbol":
      return (
        hasOnlyKeys(value, ["kind", "revealSymbol"]) &&
        isRecord(value.revealSymbol) &&
        hasOnlyKeys(value.revealSymbol, ["path", "symbolName", "symbolKind"]) &&
        typeof value.revealSymbol.symbolName === "string" &&
        (value.revealSymbol.path === undefined ||
          typeof value.revealSymbol.path === "string") &&
        (value.revealSymbol.symbolKind === undefined ||
          typeof value.revealSymbol.symbolKind === "string")
      );
    case "move_cursor":
      return (
        hasOnlyKeys(value, ["kind", "moveCursor"]) &&
        isRecord(value.moveCursor) &&
        hasOnlyKeys(value.moveCursor, ["target"]) &&
        isRecord(value.moveCursor.target) &&
        hasOnlyKeys(value.moveCursor.target, ["path", "line", "char"]) &&
        typeof value.moveCursor.target.line === "number" &&
        typeof value.moveCursor.target.char === "number" &&
        (value.moveCursor.target.path === undefined ||
          typeof value.moveCursor.target.path === "string")
      );
    case "select_range":
      return (
        hasOnlyKeys(value, ["kind", "selectRange"]) &&
        isRecord(value.selectRange) &&
        hasOnlyKeys(value.selectRange, ["target"]) &&
        isRecord(value.selectRange.target) &&
        hasOnlyKeys(value.selectRange.target, [
          "path",
          "startLine",
          "startChar",
          "endLine",
          "endChar",
        ]) &&
        typeof value.selectRange.target.startLine === "number" &&
        typeof value.selectRange.target.startChar === "number" &&
        typeof value.selectRange.target.endLine === "number" &&
        typeof value.selectRange.target.endChar === "number" &&
        (value.selectRange.target.path === undefined ||
          typeof value.selectRange.target.path === "string")
      );
    case "reveal_edit":
      return (
        hasOnlyKeys(value, ["kind", "revealEdit"]) &&
        isRecord(value.revealEdit) &&
        hasOnlyKeys(value.revealEdit, ["editId"]) &&
        typeof value.revealEdit.editId === "string"
      );
    default:
      return false;
  }
}

function isNavigationDirective(value: unknown): boolean {
  if (!isRecord(value) || typeof value.kind !== "string") {
    return false;
  }
  if (value.kind === "success") {
    return (
      hasOnlyKeys(value, ["kind", "action"]) && isNavigationAction(value.action)
    );
  }
  if (value.kind === "noop") {
    return (
      hasOnlyKeys(value, ["kind", "reason"]) && typeof value.reason === "string"
    );
  }
  return false;
}

function isUndoDirective(value: unknown): boolean {
  return (
    isRecord(value) &&
    hasOnlyKeys(value, ["scope"]) &&
    typeof value.scope === "string" &&
    (value.scope === "last_edit" || value.scope === "last_transcript")
  );
}

export function isVoiceTranscriptDirective(
  value: unknown,
): value is VoiceTranscriptDirective {
  if (!isRecord(value) || typeof value.kind !== "string") {
    return false;
  }
  if (value.kind === "edit") {
    return (
      hasOnlyKeys(value, ["kind", "editDirective"]) &&
      isEditDirective(value.editDirective)
    );
  }
  if (value.kind === "command") {
    return (
      hasOnlyKeys(value, ["kind", "commandDirective"]) &&
      isCommandDirective(value.commandDirective)
    );
  }
  if (value.kind === "navigate") {
    return (
      hasOnlyKeys(value, ["kind", "navigationDirective"]) &&
      isNavigationDirective(value.navigationDirective)
    );
  }
  if (value.kind === "undo") {
    return (
      hasOnlyKeys(value, ["kind", "undoDirective"]) &&
      isUndoDirective(value.undoDirective)
    );
  }
  if (value.kind === "rename") {
    return (
      hasOnlyKeys(value, ["kind", "renameDirective"]) &&
      isRenameDirective(value.renameDirective)
    );
  }
  if (value.kind === "code_action") {
    return (
      hasOnlyKeys(value, ["kind", "codeActionDirective"]) &&
      isCodeActionDirective(value.codeActionDirective)
    );
  }
  if (value.kind === "format") {
    return (
      hasOnlyKeys(value, ["kind", "formatDirective"]) &&
      isFormatDirective(value.formatDirective)
    );
  }
  if (value.kind === "delete_file") {
    return (
      hasOnlyKeys(value, ["kind", "deleteFileDirective"]) &&
      isDeleteFileDirective(value.deleteFileDirective)
    );
  }
  if (value.kind === "move_path") {
    return (
      hasOnlyKeys(value, ["kind", "movePathDirective"]) &&
      isMovePathDirective(value.movePathDirective)
    );
  }
  if (value.kind === "create_folder") {
    return (
      hasOnlyKeys(value, ["kind", "createFolderDirective"]) &&
      isCreateFolderDirective(value.createFolderDirective)
    );
  }
  return false;
}

function isRenameDirective(value: unknown): boolean {
  return (
    isRecord(value) &&
    hasOnlyKeys(value, ["path", "position", "newName"]) &&
    typeof value.path === "string" &&
    value.path.trim() !== "" &&
    typeof value.newName === "string" &&
    value.newName.trim() !== "" &&
    isRecord(value.position) &&
    hasOnlyKeys(value.position as Record<string, unknown>, [
      "line",
      "character",
    ]) &&
    typeof (value.position as { line: unknown }).line === "number" &&
    Number.isInteger((value.position as { line: number }).line) &&
    (value.position as { line: number }).line >= 0 &&
    typeof (value.position as { character: unknown }).character === "number" &&
    Number.isInteger((value.position as { character: number }).character) &&
    (value.position as { character: number }).character >= 0
  );
}

const codeActionKinds = new Set([
  "refactor.extract.function",
  "refactor.extract.variable",
  "refactor.extract.constant",
  "refactor.inline",
  "source.organizeImports",
  "source.fixAll",
  "quickfix",
]);

function isCodeActionDirective(value: unknown): boolean {
  if (!isRecord(value)) return false;
  const keys = [
    "path",
    "actionKind",
    "range",
    "preferredTitleIncludes",
  ] as const;
  if (!hasOnlyKeys(value, [...keys])) return false;
  if (typeof value.path !== "string" || value.path.trim() === "") return false;
  if (
    typeof value.actionKind !== "string" ||
    !codeActionKinds.has(value.actionKind)
  ) {
    return false;
  }
  if (value.range !== undefined) {
    if (!isRecord(value.range)) return false;
    const r = value.range as Record<string, unknown>;
    if (
      !hasOnlyKeys(r, ["startLine", "startChar", "endLine", "endChar"]) ||
      typeof r.startLine !== "number" ||
      !Number.isInteger(r.startLine) ||
      r.startLine < 0 ||
      typeof r.startChar !== "number" ||
      !Number.isInteger(r.startChar) ||
      r.startChar < 0 ||
      typeof r.endLine !== "number" ||
      !Number.isInteger(r.endLine) ||
      r.endLine < 0 ||
      typeof r.endChar !== "number" ||
      !Number.isInteger(r.endChar) ||
      r.endChar < 0
    ) {
      return false;
    }
  }
  if (
    value.preferredTitleIncludes !== undefined &&
    typeof value.preferredTitleIncludes !== "string"
  ) {
    return false;
  }
  return true;
}

function isFormatDirective(value: unknown): boolean {
  if (!isRecord(value)) return false;
  if (!hasOnlyKeys(value, ["path", "scope", "range"])) return false;
  if (typeof value.path !== "string" || value.path.trim() === "") return false;
  if (value.scope !== "document" && value.scope !== "selection") return false;
  if (value.range !== undefined) {
    if (!isRecord(value.range)) return false;
    const r = value.range as Record<string, unknown>;
    if (
      !hasOnlyKeys(r, ["startLine", "startChar", "endLine", "endChar"]) ||
      typeof r.startLine !== "number" ||
      !Number.isInteger(r.startLine) ||
      r.startLine < 0 ||
      typeof r.startChar !== "number" ||
      !Number.isInteger(r.startChar) ||
      r.startChar < 0 ||
      typeof r.endLine !== "number" ||
      !Number.isInteger(r.endLine) ||
      r.endLine < 0 ||
      typeof r.endChar !== "number" ||
      !Number.isInteger(r.endChar) ||
      r.endChar < 0
    ) {
      return false;
    }
  }
  return true;
}

function isDeleteFileDirective(value: unknown): boolean {
  return (
    isRecord(value) &&
    hasOnlyKeys(value, ["path"]) &&
    typeof value.path === "string" &&
    value.path.trim() !== ""
  );
}

function isMovePathDirective(value: unknown): boolean {
  return (
    isRecord(value) &&
    hasOnlyKeys(value, ["from", "to"]) &&
    typeof value.from === "string" &&
    value.from.trim() !== "" &&
    typeof value.to === "string" &&
    value.to.trim() !== ""
  );
}

function isCreateFolderDirective(value: unknown): boolean {
  return (
    isRecord(value) &&
    hasOnlyKeys(value, ["path"]) &&
    typeof value.path === "string" &&
    value.path.trim() !== ""
  );
}

// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Validator is expected to be complex (exhaustive)
export function isVoiceTranscriptCompletion(
  value: unknown,
): value is VoiceTranscriptCompletion {
  if (!isRecord(value) || typeof value.success !== "boolean") {
    return false;
  }
  const allowedKeys = new Set([
    "success",
    "summary",
    "transcriptOutcome",
    "uiDisposition",
    "searchResults",
    "activeSearchIndex",
    "answerText",
    "clarifyTargetResolution",
    "fileSelectionFocusPath",
  ]);
  if (!Object.keys(value).every((k) => allowedKeys.has(k))) {
    return false;
  }
  if (value.summary !== undefined) {
    if (typeof value.summary !== "string") {
      return false;
    }
    if ([...value.summary].length > 8192) {
      return false;
    }
  }
  if (value.success !== true && value.summary !== undefined) {
    return false;
  }
  if (value.transcriptOutcome !== undefined) {
    if (value.success !== true) {
      return false;
    }
    if (
      value.transcriptOutcome !== "irrelevant" &&
      value.transcriptOutcome !== "completed" &&
      value.transcriptOutcome !== "clarify" &&
      value.transcriptOutcome !== "clarify_control" &&
      value.transcriptOutcome !== "search" &&
      value.transcriptOutcome !== "selection" &&
      value.transcriptOutcome !== "selection_control" &&
      value.transcriptOutcome !== "file_selection" &&
      value.transcriptOutcome !== "file_selection_control" &&
      value.transcriptOutcome !== "needs_workspace_folder" &&
      value.transcriptOutcome !== "answer"
    ) {
      return false;
    }
  }

  if (value.clarifyTargetResolution !== undefined) {
    if (value.success !== true) {
      return false;
    }
    if (typeof value.clarifyTargetResolution !== "string") {
      return false;
    }
  }

  if (value.uiDisposition !== undefined) {
    if (value.success !== true) {
      return false;
    }
    if (
      value.uiDisposition !== "shown" &&
      value.uiDisposition !== "skipped" &&
      value.uiDisposition !== "hidden"
    ) {
      return false;
    }
  }

  if (value.searchResults !== undefined) {
    if (value.success !== true) {
      return false;
    }
    if (!Array.isArray(value.searchResults)) {
      return false;
    }
    for (const item of value.searchResults) {
      if (!isRecord(item)) {
        return false;
      }
      if (!hasOnlyKeys(item, ["path", "line", "character", "preview"])) {
        return false;
      }
      const rec = item as Record<string, unknown>;
      if (typeof rec.path !== "string") {
        return false;
      }
      if (!Number.isInteger(rec.line) || (rec.line as number) < 0) {
        return false;
      }
      if (!Number.isInteger(rec.character) || (rec.character as number) < 0) {
        return false;
      }
      if (typeof rec.preview !== "string") {
        return false;
      }
    }
  }
  if (value.activeSearchIndex !== undefined) {
    if (value.success !== true) {
      return false;
    }
    if (
      typeof value.activeSearchIndex !== "number" ||
      !Number.isInteger(value.activeSearchIndex) ||
      value.activeSearchIndex < 0
    ) {
      return false;
    }
  }
  if (value.answerText !== undefined) {
    if (value.success !== true) {
      return false;
    }
    if (typeof value.answerText !== "string") {
      return false;
    }
    if ([...value.answerText].length > 8192) {
      return false;
    }
  }
  if (value.fileSelectionFocusPath !== undefined) {
    if (value.success !== true) {
      return false;
    }
    if (typeof value.fileSelectionFocusPath !== "string") {
      return false;
    }
  }
  if (value.transcriptOutcome === "file_selection_control") {
    if (
      typeof value.fileSelectionFocusPath !== "string" ||
      value.fileSelectionFocusPath.trim() === ""
    ) {
      return false;
    }
  }
  return true;
}
