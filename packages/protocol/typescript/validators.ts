import type {
  EditAction,
  EditApplyResult,
  PingResult,
} from "./types.generated";

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isAnchor(value: unknown): value is { before: string; after: string } {
  return (
    isRecord(value) &&
    typeof value.before === "string" &&
    typeof value.after === "string"
  );
}

function isEditFailure(
  value: unknown,
): value is { code: string; message: string } {
  return (
    isRecord(value) &&
    typeof value.code === "string" &&
    typeof value.message === "string"
  );
}

export function isPingResult(value: unknown): value is PingResult {
  return isRecord(value) && value.message === "pong";
}

export function isEditAction(value: unknown): value is EditAction {
  return (
    isRecord(value) &&
    value.kind === "replace_between_anchors" &&
    typeof value.path === "string" &&
    isAnchor(value.anchor) &&
    typeof value.newText === "string"
  );
}

export function isEditApplyResult(value: unknown): value is EditApplyResult {
  return (
    isRecord(value) &&
    Array.isArray(value.actions) &&
    value.actions.every(isEditAction) &&
    (value.failure === undefined || isEditFailure(value.failure))
  );
}
