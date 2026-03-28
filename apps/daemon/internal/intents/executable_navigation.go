package intents

import (
	"fmt"
	"strings"
)

type NavigationIntentKind string

const (
	NavigationIntentKindOpenFile     NavigationIntentKind = "open_file"
	NavigationIntentKindRevealSymbol NavigationIntentKind = "reveal_symbol"
	NavigationIntentKindMoveCursor   NavigationIntentKind = "move_cursor"
	NavigationIntentKindSelectRange  NavigationIntentKind = "select_range"
	NavigationIntentKindRevealEdit   NavigationIntentKind = "reveal_edit"
)

type NavigationIntent struct {
	Kind         NavigationIntentKind          `json:"kind"`
	OpenFile     *OpenFileNavigationIntent     `json:"openFile,omitempty"`
	RevealSymbol *RevealSymbolNavigationIntent `json:"revealSymbol,omitempty"`
	MoveCursor   *MoveCursorNavigationIntent   `json:"moveCursor,omitempty"`
	SelectRange  *SelectRangeNavigationIntent  `json:"selectRange,omitempty"`
	RevealEdit   *RevealEditNavigationIntent   `json:"revealEdit,omitempty"`
}
type OpenFileNavigationIntent struct {
	Path string `json:"path"`
}
type RevealSymbolNavigationIntent struct {
	Path       string `json:"path,omitempty"`
	SymbolName string `json:"symbolName"`
	SymbolKind string `json:"symbolKind,omitempty"`
}
type CursorTarget struct {
	Path string `json:"path,omitempty"`
	Line int    `json:"line"`
	Char int    `json:"char"`
}
type MoveCursorNavigationIntent struct {
	Target CursorTarget `json:"target"`
}
type SelectRangeNavigationIntent struct {
	Target RangeTarget `json:"target"`
}
type RevealEditNavigationIntent struct {
	EditID string `json:"editId"`
}

func ValidateNavigationIntent(intent NavigationIntent) error {
	payloadCount := 0
	if intent.OpenFile != nil {
		payloadCount++
	}
	if intent.RevealSymbol != nil {
		payloadCount++
	}
	if intent.MoveCursor != nil {
		payloadCount++
	}
	if intent.SelectRange != nil {
		payloadCount++
	}
	if intent.RevealEdit != nil {
		payloadCount++
	}
	if payloadCount != 1 {
		return fmt.Errorf("navigation intent: exactly one payload field must be set")
	}
	switch intent.Kind {
	case NavigationIntentKindOpenFile:
		if intent.OpenFile == nil || strings.TrimSpace(intent.OpenFile.Path) == "" {
			return fmt.Errorf("navigation intent: open_file requires openFile.path")
		}
	case NavigationIntentKindRevealSymbol:
		if intent.RevealSymbol == nil || strings.TrimSpace(intent.RevealSymbol.SymbolName) == "" {
			return fmt.Errorf("navigation intent: reveal_symbol requires revealSymbol.symbolName")
		}
	case NavigationIntentKindMoveCursor:
		if intent.MoveCursor == nil {
			return fmt.Errorf("navigation intent: move_cursor requires moveCursor")
		}
	case NavigationIntentKindSelectRange:
		if intent.SelectRange == nil {
			return fmt.Errorf("navigation intent: select_range requires selectRange")
		}
	case NavigationIntentKindRevealEdit:
		if intent.RevealEdit == nil || strings.TrimSpace(intent.RevealEdit.EditID) == "" {
			return fmt.Errorf("navigation intent: reveal_edit requires revealEdit.editId")
		}
	default:
		return fmt.Errorf("navigation intent: unknown kind %q", intent.Kind)
	}
	return nil
}
