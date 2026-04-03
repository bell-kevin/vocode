package search

// Hit is one fixed-string ripgrep match under a workspace root (0-based line/column indices).
type Hit struct {
	Path    string
	Line0   int
	Char0   int
	Len     int
	Preview string
}
