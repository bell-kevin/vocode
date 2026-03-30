// Package agentcontext holds the structured input the [agent.ModelClient] sees each turn
// (one voice.transcript / one [agent.Agent.NextTurn] call): transcript, editor snapshot, gathered extras, intent history.
//
// It lives beside package [agent] so the boundary is clear: [agent] runs the model client; agentcontext
// is the value passed into [agent.Agent.NextTurn]. The name avoids a top-level type called just
// "Context", which collides mentally with [context.Context].
//
// Turn shape:
//   - [TurnContext.TranscriptText]: user utterance for this RPC (stable across turns in one Execute).
//   - [TurnContext.SucceededIntents]: intents the host reported as applied ok, plus intents fulfilled in the current Execute
//     (executable → directive, or request_context → fulfilled).
//   - [TurnContext.FailedIntents]: pre-execute, dispatch, or extension apply failures ([PhaseExtension]).
//   - [TurnContext.SkippedIntents]: intents in a reported batch that were not attempted because a prior directive failed.
//   - [TurnContext.IntentApplyHistory]: cumulative per-intent outcomes across host batches for this context session.
//   - [TurnContext.Editor]: active path and caret symbol from this RPC’s params (host refreshes each transcript).
//   - [TurnContext.Gathered]: excerpts (active file + request_context), symbols, notes; retained in the daemon
//     between transcripts when the host sends the same contextSessionId ([VoiceSessionStore]).
//
// The extension sends cursorPosition; the daemon resolves [EditorSnapshot.CursorSymbol] via symbols/tags.
package agentcontext
