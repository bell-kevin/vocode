import type { EditLocationMap } from "./navigation/execute-navigation-intent";

/** Shared mutable state while applying one transcript result (edits + navigation). */
export type TranscriptPresentContext = {
  activeDocumentPath: string;
  editLocations: EditLocationMap;
};
