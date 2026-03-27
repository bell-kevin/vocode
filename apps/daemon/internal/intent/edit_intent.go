package intent

import (
	"fmt"
	"strings"
)

type EditIntentKind string

const (
	EditIntentKindReplace      EditIntentKind = "replace"
	EditIntentKindInsert       EditIntentKind = "insert"
	EditIntentKindDelete       EditIntentKind = "delete"
	EditIntentKindInsertImport EditIntentKind = "insert_import"
	EditIntentKindCreateFile   EditIntentKind = "create_file"
	EditIntentKindAppendToFile EditIntentKind = "append_to_file"
)

type EditIntent struct {
	Kind EditIntentKind `json:"kind"`

	Replace      *ReplaceEditIntent      `json:"replace,omitempty"`
	Insert       *InsertEditIntent       `json:"insert,omitempty"`
	Delete       *DeleteEditIntent       `json:"delete,omitempty"`
	InsertImport *InsertImportEditIntent `json:"insertImport,omitempty"`
	CreateFile   *CreateFileEditIntent   `json:"createFile,omitempty"`
	AppendToFile *AppendToFileEditIntent `json:"appendToFile,omitempty"`
}

type EditTargetKind string

const (
	EditTargetKindCurrentFile      EditTargetKind = "current_file"
	EditTargetKindCurrentCursor    EditTargetKind = "current_cursor"
	EditTargetKindCurrentSelection EditTargetKind = "current_selection"
	EditTargetKindSymbolID         EditTargetKind = "symbol_id"
	EditTargetKindAnchor           EditTargetKind = "anchor"
	EditTargetKindRange            EditTargetKind = "range"
)

type EditTarget struct {
	Kind EditTargetKind `json:"kind"`

	CurrentFile      *CurrentFileTarget      `json:"currentFile,omitempty"`
	CurrentCursor    *CurrentCursorTarget    `json:"currentCursor,omitempty"`
	CurrentSelection *CurrentSelectionTarget `json:"currentSelection,omitempty"`
	SymbolID         *SymbolIDTarget         `json:"symbolId,omitempty"`
	Anchor           *AnchorTarget           `json:"anchor,omitempty"`
	Range            *RangeTarget            `json:"range,omitempty"`
}

type CurrentFileTarget struct{}
type CursorPlacement string

const (
	CursorPlacementAt     CursorPlacement = "at"
	CursorPlacementBefore CursorPlacement = "before"
	CursorPlacementAfter  CursorPlacement = "after"
)

type CurrentCursorTarget struct{ Placement CursorPlacement `json:"placement,omitempty"` }
type CurrentSelectionTarget struct{}
type SymbolIDTarget struct{ ID string `json:"id"` }
type AnchorTarget struct {
	Path   string `json:"path,omitempty"`
	Before string `json:"before"`
	After  string `json:"after"`
}
type RangeTarget struct {
	Path      string `json:"path,omitempty"`
	StartLine int    `json:"startLine"`
	StartChar int    `json:"startChar"`
	EndLine   int    `json:"endLine"`
	EndChar   int    `json:"endChar"`
}
type ReplaceEditIntent struct {
	Target  EditTarget `json:"target"`
	NewText string     `json:"newText"`
}
type InsertEditIntent struct {
	Target EditTarget `json:"target"`
	Text   string     `json:"text"`
}
type DeleteEditIntent struct{ Target EditTarget `json:"target"` }
type InsertImportEditIntent struct {
	Path   string `json:"path,omitempty"`
	Import string `json:"import"`
}
type CreateFileEditIntent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}
type AppendToFileEditIntent struct {
	Path string `json:"path"`
	Text string `json:"text"`
}

func ValidateEditIntent(intent EditIntent) error {
	payloadCount := 0
	if intent.Replace != nil { payloadCount++ }
	if intent.Insert != nil { payloadCount++ }
	if intent.Delete != nil { payloadCount++ }
	if intent.InsertImport != nil { payloadCount++ }
	if intent.CreateFile != nil { payloadCount++ }
	if intent.AppendToFile != nil { payloadCount++ }
	if payloadCount != 1 {
		return fmt.Errorf("edit intent: exactly one payload field must be set")
	}
	switch intent.Kind {
	case EditIntentKindReplace:
		if intent.Replace == nil || strings.TrimSpace(intent.Replace.NewText) == "" { return fmt.Errorf("edit intent: replace requires non-empty payload") }
		return validateTarget(intent.Replace.Target)
	case EditIntentKindInsert:
		if intent.Insert == nil || strings.TrimSpace(intent.Insert.Text) == "" { return fmt.Errorf("edit intent: insert requires non-empty payload") }
		return validateTarget(intent.Insert.Target)
	case EditIntentKindDelete:
		if intent.Delete == nil { return fmt.Errorf("edit intent: delete requires payload") }
		return validateTarget(intent.Delete.Target)
	case EditIntentKindInsertImport:
		if intent.InsertImport == nil || strings.TrimSpace(intent.InsertImport.Import) == "" { return fmt.Errorf("edit intent: insert_import requires import") }
		if !strings.HasPrefix(strings.TrimSpace(intent.InsertImport.Import), "import ") { return fmt.Errorf("edit intent: import must start with %q", "import ") }
		return nil
	case EditIntentKindCreateFile:
		if intent.CreateFile == nil || strings.TrimSpace(intent.CreateFile.Path) == "" { return fmt.Errorf("edit intent: create_file requires path") }
		return nil
	case EditIntentKindAppendToFile:
		if intent.AppendToFile == nil || strings.TrimSpace(intent.AppendToFile.Path) == "" || strings.TrimSpace(intent.AppendToFile.Text) == "" { return fmt.Errorf("edit intent: append_to_file requires non-empty path and text") }
		return nil
	default:
		return fmt.Errorf("edit intent: unknown kind %q", intent.Kind)
	}
}

func validateTarget(t EditTarget) error {
	switch t.Kind {
	case EditTargetKindCurrentFile:
		if t.CurrentFile == nil { return fmt.Errorf("edit target: current_file requires currentFile") }
	case EditTargetKindCurrentCursor:
		if t.CurrentCursor == nil { return fmt.Errorf("edit target: current_cursor requires currentCursor") }
	case EditTargetKindCurrentSelection:
		if t.CurrentSelection == nil { return fmt.Errorf("edit target: current_selection requires currentSelection") }
	case EditTargetKindSymbolID:
		if t.SymbolID == nil || strings.TrimSpace(t.SymbolID.ID) == "" { return fmt.Errorf("edit target: symbol_id requires id") }
	case EditTargetKindAnchor:
		if t.Anchor == nil || strings.TrimSpace(t.Anchor.Before) == "" || strings.TrimSpace(t.Anchor.After) == "" { return fmt.Errorf("edit target: anchor requires before and after") }
	case EditTargetKindRange:
		if t.Range == nil { return fmt.Errorf("edit target: range requires range") }
	default:
		return fmt.Errorf("edit target: unknown kind %q", t.Kind)
	}
	return nil
}
