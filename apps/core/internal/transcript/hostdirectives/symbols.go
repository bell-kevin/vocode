package hostdirectives

import (
	"path/filepath"
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func pathsEqualFold(a, b string) bool {
	a = filepath.Clean(strings.TrimSpace(a))
	b = filepath.Clean(strings.TrimSpace(b))
	if a == "" || b == "" {
		return false
	}
	return strings.EqualFold(a, b)
}

// DocumentSymbol matches LSP document symbol JSON on the wire (activeFileSymbols / host.getDocumentSymbols).
type DocumentSymbol struct {
	Name           string
	Kind           string
	Range          DocumentRange
	SelectionRange DocumentRange
}

// DocumentRange is an LSP range (0-based lines / UTF-16 chars).
type DocumentRange struct {
	StartLine, StartChar, EndLine, EndChar int64
}

// DocumentSymbolHost fetches symbols for a file path (VS Code extension).
type DocumentSymbolHost interface {
	GetDocumentSymbols(protocol.HostGetDocumentSymbolsParams) (protocol.HostGetDocumentSymbolsResult, error)
}

// DocumentSymbolsFromParams returns symbols for params.ActiveFile (already on the wire).
func DocumentSymbolsFromParams(params protocol.VoiceTranscriptParams) []DocumentSymbol {
	out := make([]DocumentSymbol, 0, len(params.ActiveFileSymbols))
	for _, s := range params.ActiveFileSymbols {
		out = append(out, DocumentSymbol{
			Name: s.Name,
			Kind: s.Kind,
			Range: DocumentRange{
				StartLine: s.Range.StartLine,
				StartChar: s.Range.StartChar,
				EndLine:   s.Range.EndLine,
				EndChar:   s.Range.EndChar,
			},
			SelectionRange: DocumentRange{
				StartLine: s.SelectionRange.StartLine,
				StartChar: s.SelectionRange.StartChar,
				EndLine:   s.SelectionRange.EndLine,
				EndChar:   s.SelectionRange.EndChar,
			},
		})
	}
	return out
}

// DocumentSymbolsFromHostResult converts a host.getDocumentSymbols response.
func DocumentSymbolsFromHostResult(res protocol.HostGetDocumentSymbolsResult) []DocumentSymbol {
	out := make([]DocumentSymbol, 0, len(res.Symbols))
	for _, s := range res.Symbols {
		out = append(out, DocumentSymbol{
			Name: s.Name,
			Kind: s.Kind,
			Range: DocumentRange{
				StartLine: s.Range.StartLine,
				StartChar: s.Range.StartChar,
				EndLine:   s.Range.EndLine,
				EndChar:   s.Range.EndChar,
			},
			SelectionRange: DocumentRange{
				StartLine: s.SelectionRange.StartLine,
				StartChar: s.SelectionRange.StartChar,
				EndLine:   s.SelectionRange.EndLine,
				EndChar:   s.SelectionRange.EndChar,
			},
		})
	}
	return out
}

// DocumentSymbolsForPath prefers params.ActiveFile symbols when hitPath matches; otherwise asks the host.
func DocumentSymbolsForPath(host DocumentSymbolHost, params protocol.VoiceTranscriptParams, hitPath string) []DocumentSymbol {
	active := strings.TrimSpace(params.ActiveFile)
	if active != "" && pathsEqualFold(hitPath, active) && len(params.ActiveFileSymbols) > 0 {
		return DocumentSymbolsFromParams(params)
	}
	if host == nil {
		return nil
	}
	p := strings.TrimSpace(hitPath)
	if p == "" {
		return nil
	}
	res, err := host.GetDocumentSymbols(protocol.HostGetDocumentSymbolsParams{Path: p})
	if err != nil || len(res.Symbols) == 0 {
		return nil
	}
	return DocumentSymbolsFromHostResult(res)
}

// HitNavigateDirectivesExpandWithSymbols selects the smallest symbol range containing the anchor, else narrow rg span.
func HitNavigateDirectivesExpandWithSymbols(path string, line0, char0, length int, syms []DocumentSymbol) []protocol.VoiceTranscriptDirective {
	if len(syms) == 0 {
		return HitNavigateDirectives(path, line0, char0, length)
	}
	if sl, sc, el, ec, ok := smallestSymbolContainingPointSyms(syms, line0, char0); ok {
		return hitNavigateDirectivesRange(path, sl, sc, el, ec)
	}
	return HitNavigateDirectives(path, line0, char0, length)
}

// SmallestSymbolContainingRange returns the smallest document symbol range containing (line0, char0).
func SmallestSymbolContainingRange(symbols []DocumentSymbol, line0, char0 int) (startLine, startChar, endLine, endChar int, ok bool) {
	return smallestSymbolContainingPointSyms(symbols, line0, char0)
}

// SmallestDocumentSymbolAtPoint returns the tightest flattened document symbol whose LSP range contains (line0, char0).
func SmallestDocumentSymbolAtPoint(symbols []DocumentSymbol, line0, char0 int) (sym DocumentSymbol, startLine, startChar, endLine, endChar int, ok bool) {
	line := int64(line0)
	char := int64(char0)
	bestIdx := -1
	bestSize := 0
	for i := range symbols {
		r := symbols[i].Range
		if line < r.StartLine || line > r.EndLine {
			continue
		}
		if line == r.StartLine && char < r.StartChar {
			continue
		}
		if line == r.EndLine && char > r.EndChar {
			continue
		}
		sz := int(r.EndLine-r.StartLine)*1_000_000 + int(r.EndChar-r.StartChar)
		if bestIdx == -1 || sz < bestSize {
			bestIdx = i
			bestSize = sz
		}
	}
	if bestIdx == -1 {
		return DocumentSymbol{}, 0, 0, 0, 0, false
	}
	s := symbols[bestIdx]
	r := s.Range
	return s, int(r.StartLine), int(r.StartChar), int(r.EndLine), int(r.EndChar), true
}

func smallestSymbolContainingPointSyms(symbols []DocumentSymbol, line0, char0 int) (startLine, startChar, endLine, endChar int, ok bool) {
	_, sl, sc, el, ec, ok := SmallestDocumentSymbolAtPoint(symbols, line0, char0)
	if !ok {
		return 0, 0, 0, 0, false
	}
	return sl, sc, el, ec, true
}

// WorkspaceSearchStylePreviewFromSymbol formats a sidebar preview similar to workspace symbol hits (e.g. "function pong()").
func WorkspaceSearchStylePreviewFromSymbol(s DocumentSymbol) string {
	name := strings.TrimSpace(s.Name)
	name = strings.TrimSuffix(name, "()")
	k := strings.ToLower(s.Kind)
	switch {
	case strings.Contains(k, "function"):
		return "function " + name + "()"
	case strings.Contains(k, "method"):
		return "method " + name + "()"
	case strings.Contains(k, "class"):
		return "class " + name
	case strings.Contains(k, "interface"):
		return "interface " + name
	default:
		if t := strings.TrimSpace(s.Name); t != "" {
			return t
		}
		return name
	}
}

// CreateFlowHitPreview picks an LSP-style preview when the anchor lies inside a document symbol; otherwise returns fallback.
func CreateFlowHitPreview(syms []DocumentSymbol, line0, char0 int, fallback string) string {
	if len(syms) == 0 {
		return fallback
	}
	sym, _, _, _, _, ok := SmallestDocumentSymbolAtPoint(syms, line0, char0)
	if !ok {
		return fallback
	}
	return WorkspaceSearchStylePreviewFromSymbol(sym)
}

// HitNavigateDirectivesExpand uses active-file symbols only (legacy helper for tests).
func HitNavigateDirectivesExpand(
	params protocol.VoiceTranscriptParams,
	path string,
	line0, char0, length int,
) []protocol.VoiceTranscriptDirective {
	syms := DocumentSymbolsForPath(nil, params, path)
	return HitNavigateDirectivesExpandWithSymbols(path, line0, char0, length, syms)
}
