# gRPC + Protobuf Duplex Backlog

## Why
The current “best UX” duplex implementation uses line-delimited JSON-RPC over stdio:
- daemon plans and sends directive batches to the extension (`host.applyDirectives`)
- extension executes and returns per-directive outcomes
- daemon continues planning until directives are empty

This works, but the protocol boundary would be cleaner with a proper streaming transport.

## Target streaming shape
Implement a duplex gRPC stream (one stream per committed transcript / `contextSessionId` key):

1. Daemon->Host messages: `DirectiveBatch`
   - `applyBatchId`
   - `activeFile`
   - `directives[]`
2. Host->Daemon messages: `ApplyOutcomeBatch`
   - `applyBatchId`
   - `items[]` with `status: ok|failed|skipped` and optional `message`
3. Daemon completes by sending a final `VoiceTranscriptResult` (with `success=true` and no directives).

## Mapping from current JSON-RPC
- `host.applyDirectives` request.params becomes `DirectiveBatch`
- `host.applyDirectives` result becomes `ApplyOutcomeBatch.items`
- The daemon-owned internal apply/repair loop becomes the server-side stream loop

## Notes / constraints
- Preserve the “no repeated successful directives” rule by feeding the cumulative `IntentApplyHistory` into the next planning iteration (same as today).
- Keep a repair cap (current: `VOCODE_DAEMON_VOICE_MAX_REPAIR_RPCS`) to avoid infinite retry loops.

