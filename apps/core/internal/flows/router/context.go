package router

import "vocoding.net/vocode/v2/apps/core/internal/flows"

// Context is input for flow-scoped transcript routing (classifier).
type Context struct {
	Flow        flows.ID
	Instruction string
	Editor      EditorSnapshot

	HitCount    int
	ActiveIndex int

	FocusPath       string
	ListCount       int
	ListActiveIndex int
}

// EditorSnapshot is editor context passed into routing.
type EditorSnapshot struct {
	ActiveFilePath string
	WorkspaceRoot  string
	CursorSymbol   *SymbolRef
}

// SymbolRef is a lightweight cursor symbol reference.
type SymbolRef struct {
	Name string
	Kind string
}
