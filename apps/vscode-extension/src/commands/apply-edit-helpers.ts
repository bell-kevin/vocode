import type { ReplaceBetweenAnchorsAction } from "@vocode/protocol";

function findUniqueOccurrence(
  documentText: string,
  needle: string,
  searchStart: number,
  label: string,
): number {
  const firstIndex = documentText.indexOf(needle, searchStart);
  if (firstIndex === -1) {
    throw new Error(
      `Could not find ${label} anchor: ${JSON.stringify(needle)}`,
    );
  }

  const secondIndex = documentText.indexOf(needle, firstIndex + 1);
  if (secondIndex !== -1) {
    throw new Error(
      `Anchor was ambiguous for ${label}: ${JSON.stringify(needle)}`,
    );
  }

  return firstIndex;
}

export function applyReplaceBetweenAnchors(
  documentText: string,
  action: ReplaceBetweenAnchorsAction,
): string {
  const beforeIndex = findUniqueOccurrence(
    documentText,
    action.anchor.before,
    0,
    "before",
  );

  const searchStart = beforeIndex + action.anchor.before.length;
  const afterIndex = findUniqueOccurrence(
    documentText,
    action.anchor.after,
    searchStart,
    "after",
  );

  const prefix = documentText.slice(0, searchStart);
  const suffix = documentText.slice(afterIndex);

  return `${prefix}${action.newText}${suffix}`;
}
