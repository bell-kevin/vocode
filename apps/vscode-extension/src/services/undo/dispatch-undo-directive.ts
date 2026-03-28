import type { UndoDirective } from "@vocode/protocol";

import { applyUndoDirective } from "./transcript-undo-ledger";

/** Applies one undo directive (host undo stack / transcript ledger). */
export function dispatchUndoDirective(
  directive: UndoDirective | undefined,
): Promise<boolean> {
  return applyUndoDirective(directive);
}
