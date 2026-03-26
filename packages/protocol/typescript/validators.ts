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
  return (
    isRecord(value) &&
    hasOnlyKeys(value, ["kind", "path", "anchor", "newText"]) &&
    value.kind === "replace_between_anchors" &&
    typeof value.path === "string" &&
    isAnchor(value.anchor) &&
    typeof value.newText === "string"
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
