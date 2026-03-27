package symbols

// TreeSitterResolver is a placeholder for a future language-aware resolver.
// It intentionally returns no matches today so callers can fall back to
// lexical resolution (e.g. ripgrep) without changing resolver contracts.
type TreeSitterResolver struct{}

func NewTreeSitterResolver() *TreeSitterResolver {
	return &TreeSitterResolver{}
}

func (r *TreeSitterResolver) ResolveSymbol(workspaceRoot, symbolName, symbolKind, hintPath string) ([]SymbolRef, error) {
	_ = workspaceRoot
	_ = symbolName
	_ = symbolKind
	_ = hintPath
	return nil, nil
}
