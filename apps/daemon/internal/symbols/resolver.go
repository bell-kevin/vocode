package symbols

type SymbolRef struct {
	Path string
	Line int
	Kind string
}

type Resolver interface {
	ResolveSymbol(workspaceRoot, symbolName, symbolKind, hintPath string) ([]SymbolRef, error)
}
