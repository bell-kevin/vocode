package protocol

import (
	"errors"
	"fmt"
	"strings"
)

// EditDirective validation lives here alongside future protocol-level validators
// (mirrors typescript/validators.ts conceptually).

func (r EditDirective) Validate() error {
	switch r.Kind {
	case "success":
		if r.Actions == nil {
			return errors.New("success result must include actions")
		}
		if r.Reason != "" {
			return errors.New("success result must not contain reason")
		}
	case "noop":
		if r.Reason == "" {
			return errors.New("noop result must include reason")
		}
		if len(r.Actions) > 0 {
			return errors.New("noop result must not contain actions")
		}
	default:
		return errors.New("unknown edit.dispatch result kind")
	}

	return nil
}

func (s VoiceTranscriptDirective) Validate() error {
	switch s.Kind {
	case "edit":
		if s.EditDirective == nil || s.CommandDirective != nil || s.NavigationDirective != nil || s.UndoDirective != nil {
			return errors.New("voice transcript step: kind edit requires editDirective and no other directive payloads")
		}
		return s.EditDirective.Validate()
	case "command":
		if s.CommandDirective == nil || s.EditDirective != nil || s.NavigationDirective != nil || s.UndoDirective != nil {
			return errors.New("voice transcript step: kind command requires commandDirective and no other directive payloads")
		}
		if strings.TrimSpace(s.CommandDirective.Command) == "" {
			return errors.New("voice transcript step: command requires non-empty commandDirective.command")
		}
		// CommandDirective has no additional protocol-level validation yet; host-side
		// policy executes the safety checks.
		return nil
	case "navigate":
		if s.NavigationDirective == nil || s.EditDirective != nil || s.CommandDirective != nil || s.UndoDirective != nil {
			return errors.New("voice transcript step: kind navigate requires navigationDirective and no other directive payloads")
		}
		return validateNavigationDirective(s.NavigationDirective)
	case "undo":
		if s.UndoDirective == nil || s.EditDirective != nil || s.CommandDirective != nil || s.NavigationDirective != nil {
			return errors.New("voice transcript step: kind undo requires undoDirective and no other directive payloads")
		}
		return validateUndoDirective(s.UndoDirective)
	default:
		return fmt.Errorf("voice transcript step: unknown kind %q", s.Kind)
	}
}

func validateUndoDirective(u *UndoDirective) error {
	if u == nil {
		return errors.New("voice transcript step: undo requires undoDirective")
	}
	switch strings.TrimSpace(u.Scope) {
	case "last_edit", "last_transcript":
		return nil
	default:
		return fmt.Errorf("voice transcript step: undo scope must be last_edit or last_transcript, got %q", u.Scope)
	}
}

func validateNavigationDirective(n *NavigationDirective) error {
	if n == nil {
		return errors.New("voice transcript step: navigate requires navigationDirective")
	}
	switch n.Kind {
	case "success":
		if n.Action == nil {
			return errors.New("voice transcript step: navigate success requires navigationDirective.action")
		}
		if n.Reason != "" {
			return errors.New("voice transcript step: navigate success must not include reason")
		}
		return validateNavigationAction(n.Action)
	case "noop":
		if strings.TrimSpace(n.Reason) == "" {
			return errors.New("voice transcript step: navigate noop requires reason")
		}
		if n.Action != nil {
			return errors.New("voice transcript step: navigate noop must not include action")
		}
		return nil
	default:
		return fmt.Errorf("voice transcript step: unknown navigation directive kind %q", n.Kind)
	}
}

func validateNavigationAction(n *NavigationAction) error {
	if n == nil {
		return errors.New("voice transcript step: navigate requires navigationDirective.action")
	}
	kind := strings.TrimSpace(n.Kind)
	if kind == "" {
		return errors.New("voice transcript step: navigate requires non-empty navigationDirective.action.kind")
	}

	payloads := 0
	if n.OpenFile != nil {
		payloads++
	}
	if n.RevealSymbol != nil {
		payloads++
	}
	if n.MoveCursor != nil {
		payloads++
	}
	if n.SelectRange != nil {
		payloads++
	}
	if n.RevealEdit != nil {
		payloads++
	}
	if payloads != 1 {
		return errors.New("voice transcript step: navigate requires exactly one navigation action payload")
	}

	switch kind {
	case "open_file":
		if n.OpenFile == nil || strings.TrimSpace(n.OpenFile.Path) == "" {
			return errors.New("voice transcript step: open_file requires openFile.path")
		}
	case "reveal_symbol":
		if n.RevealSymbol == nil || strings.TrimSpace(n.RevealSymbol.SymbolName) == "" {
			return errors.New("voice transcript step: reveal_symbol requires revealSymbol.symbolName")
		}
	case "move_cursor":
		if n.MoveCursor == nil || n.MoveCursor.Target.Line < 0 || n.MoveCursor.Target.Char < 0 {
			return errors.New("voice transcript step: move_cursor requires moveCursor.target with non-negative line/char")
		}
	case "select_range":
		if n.SelectRange == nil {
			return errors.New("voice transcript step: select_range requires selectRange.target")
		}
		t := n.SelectRange.Target
		if t.StartLine < 0 || t.StartChar < 0 || t.EndLine < 0 || t.EndChar < 0 {
			return errors.New("voice transcript step: select_range requires non-negative target positions")
		}
	case "reveal_edit":
		if n.RevealEdit == nil || strings.TrimSpace(n.RevealEdit.EditId) == "" {
			return errors.New("voice transcript step: reveal_edit requires revealEdit.editId")
		}
	default:
		return fmt.Errorf("voice transcript step: unknown navigation kind %q", kind)
	}

	return nil
}

func (r VoiceTranscriptResult) Validate() error {
	if !r.Success && len(r.Directives) > 0 {
		return errors.New("voice transcript result must not include directives when success=false")
	}
	if !r.Success && r.Summary != "" {
		return errors.New("voice transcript result must not include summary when success=false")
	}
	if !r.Success && r.ApplyBatchId != "" {
		return errors.New("voice transcript result must not include applyBatchId when success=false")
	}
	if len([]rune(r.Summary)) > 8192 {
		return errors.New("voice transcript result: summary exceeds max length")
	}
	for i := range r.Directives {
		if err := r.Directives[i].Validate(); err != nil {
			return fmt.Errorf("voice transcript result directives[%d]: %w", i, err)
		}
	}
	if r.Success {
		if len(r.Directives) > 0 {
			if strings.TrimSpace(r.ApplyBatchId) == "" {
				return errors.New("voice transcript result requires applyBatchId when directives are present")
			}
		} else if r.ApplyBatchId != "" {
			return errors.New("voice transcript result must not include applyBatchId without directives")
		}
	}
	return nil
}
