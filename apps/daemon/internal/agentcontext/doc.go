// Package agentcontext holds session-shaped state and structured values passed into the agent
// for voice.transcript: [Gathered] rolling context, [VoiceSession] / [VoiceSessionStore], flow stack
// and clarify rules, host apply batches, and the small context structs in model_context.go
// ([TranscriptClassifierContext], [ScopeIntentContext], [ScopedEditContext]) plus [EditorSnapshot].
//
// It sits beside package agent so the split is obvious: agent runs [agent.ModelClient]; agentcontext
// is the data those calls consume. The name avoids a top-level type called "Context", which collides
// mentally with [context.Context].
//
// The extension sends cursorPosition each RPC; the daemon may resolve [EditorSnapshot.CursorSymbol]
// via symbols/tags when wiring classifier / scope prompts.
package agentcontext
