import type { ReplaceBetweenAnchorsAction } from "@vocode/protocol";

export function applyReplaceBetweenAnchors(
  documentText: string,
  action: ReplaceBetweenAnchorsAction,
): string {
  const beforeIndex = documentText.indexOf(action.anchor.before);
  if (beforeIndex === -1) {
    throw new Error(
      `Could not find before anchor: ${JSON.stringify(action.anchor.before)}`,
    );
  }

  const searchStart = beforeIndex + action.anchor.before.length;
  const afterIndex = documentText.indexOf(action.anchor.after, searchStart);
  if (afterIndex === -1) {
    throw new Error(
      `Could not find after anchor: ${JSON.stringify(action.anchor.after)}`,
    );
  }

  const prefix = documentText.slice(0, searchStart);
  const suffix = documentText.slice(afterIndex);

  return `${prefix}${action.newText}${suffix}`;
}
