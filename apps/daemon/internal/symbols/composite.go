package symbols

type CompositeResolver struct {
	resolvers []Resolver
}

func NewCompositeResolver(resolvers ...Resolver) *CompositeResolver {
	out := make([]Resolver, 0, len(resolvers))
	for _, r := range resolvers {
		if r != nil {
			out = append(out, r)
		}
	}
	return &CompositeResolver{resolvers: out}
}

func (c *CompositeResolver) ResolveSymbol(workspaceRoot, symbolName, symbolKind, hintPath string) ([]SymbolRef, error) {
	for _, r := range c.resolvers {
		matches, err := r.ResolveSymbol(workspaceRoot, symbolName, symbolKind, hintPath)
		if err != nil {
			continue
		}
		if len(matches) > 0 {
			return matches, nil
		}
	}
	return nil, nil
}
