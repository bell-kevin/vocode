import type { ReplaceBetweenAnchorsAction } from "@vocode/protocol";

export interface AnchoredReplacement {
  startOffset: number;
  endOffset: number;
  replacementText: string;
  nextText: string;
}

function findUniqueOccurrence(
  text: string,
  needle: string,
  start: number,
  label: "before" | "after",
): number {
  if (!needle) {
    throw new Error(`The ${label} anchor was empty.`);
  }

  const relativeIndex = text.indexOf(needle, start);
  if (relativeIndex === -1) {
    throw new Error(
      `Could not find ${label} anchor: ${JSON.stringify(needle)}`,
    );
  }

  const duplicateIndex = text.indexOf(needle, relativeIndex + 1);
  if (duplicateIndex !== -1) {
    throw new Error(
      `The ${label} anchor matched multiple locations: ${JSON.stringify(
        needle,
      )}`,
    );
  }

  return relativeIndex;
}

export function resolveReplaceBetweenAnchors(
  documentText: string,
  action: ReplaceBetweenAnchorsAction,
): AnchoredReplacement {
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
  const nextText = `${prefix}${action.newText}${suffix}`;

  return {
    startOffset: searchStart,
    endOffset: afterIndex,
    replacementText: action.newText,
    nextText,
  };
}
