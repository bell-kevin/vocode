import type {
  CommandDirective,
  EditAction,
  EditDirective,
  PingResult,
  VoiceTranscriptDirective,
  VoiceTranscriptResult,
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
  return false;
}

function isVoiceTranscriptResultApplyBatchIdField(
  value: Record<string, unknown>,
): boolean {
  const idRaw = value.applyBatchId;
  if (idRaw !== undefined && typeof idRaw !== "string") {
    return false;
  }
  if (value.success !== true) {
    return idRaw === undefined;
  }
  const batchID = typeof idRaw === "string" ? idRaw.trim() : "";
  const dirs = value.directives;
  if (Array.isArray(dirs) && dirs.length > 0) {
    return batchID !== "";
  }
  return batchID === "";
}

export function isVoiceTranscriptResult(
  value: unknown,
): value is VoiceTranscriptResult {
  if (!isRecord(value) || typeof value.success !== "boolean") {
    return false;
  }
  const allowedKeys = new Set([
    "success",
    "directives",
    "summary",
    "applyBatchId",
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
  if (value.directives !== undefined) {
    if (!Array.isArray(value.directives)) {
      return false;
    }
    if (!value.directives.every(isVoiceTranscriptDirective)) {
      return false;
    }
  }
  if (value.success !== true && value.directives !== undefined) {
    return false;
  }
  if (value.success !== true && value.summary !== undefined) {
    return false;
  }
  return isVoiceTranscriptResultApplyBatchIdField(value);
}
