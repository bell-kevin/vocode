package agentcontext

import "vocoding.net/vocode/v2/apps/daemon/internal/symbols"

// FileExcerpt is a path plus text slice (typically from request_file_excerpt).
type FileExcerpt struct {
	Path    string
	Content string
}

// Gathered holds file excerpts, symbols, and notes carried in [VoiceSession] between RPCs.
// The executor seeds the active file from disk each utterance; [ApplyGatheredRollingCap] trims
// excerpts (never evicting the current active file) under daemon caps.
type Gathered struct {
	Symbols  []symbols.SymbolRef
	Excerpts []FileExcerpt
	Notes    []string
}
