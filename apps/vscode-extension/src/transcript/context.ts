import type { EditLocationMap } from "../directives/navigation/execute-navigation-intent";

/** Shared mutable state while applying one transcript result (edits + navigation). */
export type TranscriptApplyContext = {
  activeDocumentPath: string;
  editLocations: EditLocationMap;
};
