package symbols

type SymbolRef struct {
	ID   string
	Name string
	Path string
	Line int
	Kind string
}

type Resolver interface {
	ResolveSymbol(workspaceRoot, symbolName, symbolKind, hintPath string) ([]SymbolRef, error)
	// ResolveFileSymbols returns definition symbols for a specific file path.
	// Used to pre-populate gathered.symbols for the active file when building scope-intent context.
	ResolveFileSymbols(workspaceRoot, absFile string) ([]SymbolRef, error)
	// ResolveInnermostAtLine returns the innermost definition span for tree-sitter coordinates:
	// 0-based line and 0-based UTF-8 byte column within that line (after converting from LSP UTF-16 in the executor).
	ResolveInnermostAtLine(workspaceRoot, activeFile string, line0Based, byteCol0 int) (SymbolRef, bool)
}
