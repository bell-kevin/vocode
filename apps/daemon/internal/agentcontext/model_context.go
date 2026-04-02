package agentcontext

// ResolvedTarget is a daemon-resolved, bounded edit target.
// The model never chooses the range; it only returns replacement text for this range.
type ResolvedTarget struct {
	Path        string
	Range       Range
	Fingerprint string
}

// Range is a 0-based, inclusive-exclusive span.
// Lines/chars are in VS Code coordinates (LSP UTF-16 columns).
type Range struct {
	StartLine int `json:"startLine"`
	StartChar int `json:"startChar"`
	EndLine   int `json:"endLine"`
	EndChar   int `json:"endChar"`
}

// ScopedEditContext is everything the model sees for one scoped edit call.
type ScopedEditContext struct {
	Instruction string
	Editor      EditorSnapshot
	Target      ResolvedTarget
	TargetText  string
}

// ScopeIntentContext is everything the model sees for one scope-intent call.
type ScopeIntentContext struct {
	Instruction       string
	Editor            EditorSnapshot
	ActiveFileSymbols []struct {
		Name string `json:"name"`
		Kind string `json:"kind"`
	} `json:"activeFileSymbols,omitempty"`
}

// TranscriptClassifierContext is everything the model sees for the first-pass transcript router.
type TranscriptClassifierContext struct {
	Instruction string
	Editor      EditorSnapshot
}
