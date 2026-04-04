/** Shown in the transcript panel when the daemon returns success: false. */
export const FAILED_TO_PROCESS_TRANSCRIPT = "Failed to process transcript.";

/**
 * Daemon business-rule failures use JSON-RPC error responses; the client surfaces them as
 * `Error` with message `[rpc] <code>: <human text>`. Strip the prefix for sidebar / toast copy.
 */
export function userFacingTranscriptRpcError(err: unknown): string {
  if (!(err instanceof Error)) {
    return typeof err === "string"
      ? err
      : "Unknown error while running the transcript.";
  }
  const m = err.message;
  const match = /^\[rpc\] -?\d+:\s*(.*)$/s.exec(m);
  if (match?.[1]?.trim()) {
    return match[1].trim();
  }
  return m;
}
