import type {
  EditAction,
  EditApplyResult,
  PingResult,
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

export function isVoiceTranscriptResult(
  value: unknown,
): value is VoiceTranscriptResult {
  return (
    isRecord(value) &&
    hasOnlyKeys(value, ["accepted"]) &&
    value.accepted === true
  );
}
