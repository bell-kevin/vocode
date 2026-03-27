import type {
  CommandRunParams,
  CommandRunResult,
  EditAction,
  EditApplyResult,
  PingResult,
  VoiceTranscriptResult,
  VoiceTranscriptStepResult,
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

function isEditFailure(
  value: unknown,
): value is { code: string; message: string } {
  const validCodes = new Set([
    "unsupported_instruction",
    "ambiguous_target",
    "missing_anchor",
    "validation_failed",
    "no_change_needed",
  ]);
  return (
    isRecord(value) &&
    hasOnlyKeys(value, ["code", "message"]) &&
    typeof value.code === "string" &&
    validCodes.has(value.code) &&
    typeof value.message === "string"
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

export function isEditApplyResult(value: unknown): value is EditApplyResult {
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
    case "failure":
      return (
        hasOnlyKeys(value, ["kind", "failure"]) && isEditFailure(value.failure)
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

export function isCommandRunResult(value: unknown): value is CommandRunResult {
  if (!isRecord(value) || typeof value.kind !== "string") {
    return false;
  }

  switch (value.kind) {
    case "success":
      return (
        hasOnlyKeys(value, ["kind", "exitCode", "stdout", "stderr"]) &&
        typeof value.exitCode === "number" &&
        typeof value.stdout === "string" &&
        typeof value.stderr === "string"
      );
    case "failure": {
      const failure = value.failure;
      const validCodes = new Set([
        "command_rejected",
        "execution_failed",
        "timeout",
      ]);

      return (
        hasOnlyKeys(value, ["kind", "failure", "stdout", "stderr"]) &&
        typeof value.stdout === "string" &&
        typeof value.stderr === "string" &&
        isRecord(failure) &&
        hasOnlyKeys(failure, ["code", "message"]) &&
        typeof failure.code === "string" &&
        validCodes.has(failure.code) &&
        typeof failure.message === "string"
      );
    }
    default:
      return false;
  }
}

function isCommandRunParams(value: unknown): value is CommandRunParams {
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

function isNavigationIntent(value: unknown): boolean {
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

export function isVoiceTranscriptStepResult(
  value: unknown,
): value is VoiceTranscriptStepResult {
  if (!isRecord(value) || typeof value.kind !== "string") {
    return false;
  }
  if (value.kind === "edit") {
    return (
      hasOnlyKeys(value, ["kind", "editResult"]) &&
      isEditApplyResult(value.editResult)
    );
  }
  if (value.kind === "run_command") {
    return (
      hasOnlyKeys(value, ["kind", "commandParams"]) &&
      isCommandRunParams(value.commandParams)
    );
  }
  if (value.kind === "navigate") {
    return (
      hasOnlyKeys(value, ["kind", "navigationIntent"]) &&
      isNavigationIntent(value.navigationIntent)
    );
  }
  return false;
}

export function isVoiceTranscriptResult(
  value: unknown,
): value is VoiceTranscriptResult {
  if (!isRecord(value) || value.accepted !== true) {
    return false;
  }
  const allowedKeys = new Set(["accepted", "planError", "steps"]);
  if (!Object.keys(value).every((k) => allowedKeys.has(k))) {
    return false;
  }
  if (value.planError !== undefined && typeof value.planError !== "string") {
    return false;
  }
  if (value.steps !== undefined) {
    if (!Array.isArray(value.steps)) {
      return false;
    }
    if (!value.steps.every(isVoiceTranscriptStepResult)) {
      return false;
    }
  }
  const planErr = value.planError;
  if (
    typeof planErr === "string" &&
    planErr !== "" &&
    Array.isArray(value.steps) &&
    value.steps.length > 0
  ) {
    return false;
  }
  return true;
}
