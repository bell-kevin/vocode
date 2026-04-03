package flows

// ID names a transcript routing context (base flow).
type ID string

const (
	Root            ID = "root"
	WorkspaceSelect ID = "workspace_select" // workspace text/symbol hit list (parallel: SelectFile = paths)
	SelectFile      ID = "select_file"
)
