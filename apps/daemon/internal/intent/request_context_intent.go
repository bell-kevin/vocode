package intent

type RequestContextKind string

const (
	RequestContextKindSymbols     RequestContextKind = "request_symbols"
	RequestContextKindFileExcerpt RequestContextKind = "request_file_excerpt"
	RequestContextKindUsages      RequestContextKind = "request_usages"
)

type RequestContextIntent struct {
	Kind RequestContextKind `json:"kind"`
	Path      string `json:"path,omitempty"`
	Query     string `json:"query,omitempty"`
	SymbolID  string `json:"symbolId,omitempty"`
	MaxResult int    `json:"maxResult,omitempty"`
}
