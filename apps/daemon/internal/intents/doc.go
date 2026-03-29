// Package intents defines agent output (structured intents) validated by the daemon and routed
// by the intents dispatch layer (control vs executable).

// # Taxonomy

// Intents are either control or executable (see [Intent]).

// Control intents ([ControlIntent]) affect the agent loop only; they do not produce host
// directives. Payload types:
//  - [DoneIntent] — optional summary string for the extension UI when stopping ([ControlIntentKindDone]).
//  - [RequestContextIntent] — symbols, file excerpts, usage notes for gathered context.

// Executable intents ([ExecutableIntent]) map to at most one protocol directive for the extension.
// Payload types:
//  - [EditIntent]
//  - [CommandIntent]
//  - [NavigationIntent]
//  - [UndoIntent]

// The "done" control kind may include [DoneIntent] with a human-readable summary.
package intents
