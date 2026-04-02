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
		if s.EditDirective == nil || s.CommandDirective != nil || s.NavigationDirective != nil || s.UndoDirective != nil ||
			s.RenameDirective != nil || s.CodeActionDirective != nil || s.FormatDirective != nil ||
			s.DeleteFileDirective != nil || s.MovePathDirective != nil || s.CreateFolderDirective != nil {
			return errors.New("voice transcript step: kind edit requires editDirective and no other directive payloads")
		}
		return s.EditDirective.Validate()
	case "command":
		if s.CommandDirective == nil || s.EditDirective != nil || s.NavigationDirective != nil || s.UndoDirective != nil ||
			s.RenameDirective != nil || s.CodeActionDirective != nil || s.FormatDirective != nil ||
			s.DeleteFileDirective != nil || s.MovePathDirective != nil || s.CreateFolderDirective != nil {
			return errors.New("voice transcript step: kind command requires commandDirective and no other directive payloads")
		}
		if strings.TrimSpace(s.CommandDirective.Command) == "" {
			return errors.New("voice transcript step: command requires non-empty commandDirective.command")
		}
		// CommandDirective has no additional protocol-level validation yet; host-side
		// policy executes the safety checks.
		return nil
	case "navigate":
		if s.NavigationDirective == nil || s.EditDirective != nil || s.CommandDirective != nil || s.UndoDirective != nil ||
			s.RenameDirective != nil || s.CodeActionDirective != nil || s.FormatDirective != nil ||
			s.DeleteFileDirective != nil || s.MovePathDirective != nil || s.CreateFolderDirective != nil {
			return errors.New("voice transcript step: kind navigate requires navigationDirective and no other directive payloads")
		}
		return validateNavigationDirective(s.NavigationDirective)
	case "undo":
		if s.UndoDirective == nil || s.EditDirective != nil || s.CommandDirective != nil || s.NavigationDirective != nil ||
			s.RenameDirective != nil || s.CodeActionDirective != nil || s.FormatDirective != nil ||
			s.DeleteFileDirective != nil || s.MovePathDirective != nil || s.CreateFolderDirective != nil {
			return errors.New("voice transcript step: kind undo requires undoDirective and no other directive payloads")
		}
		return validateUndoDirective(s.UndoDirective)
	case "rename":
		if s.RenameDirective == nil || s.EditDirective != nil || s.CommandDirective != nil || s.NavigationDirective != nil || s.UndoDirective != nil ||
			s.CodeActionDirective != nil || s.FormatDirective != nil ||
			s.DeleteFileDirective != nil || s.MovePathDirective != nil || s.CreateFolderDirective != nil {
			return errors.New("voice transcript step: kind rename requires renameDirective and no other directive payloads")
		}
		r := s.RenameDirective
		if strings.TrimSpace(r.Path) == "" || strings.TrimSpace(r.NewName) == "" {
			return errors.New("voice transcript step: rename requires path and newName")
		}
		if r.Position.Line < 0 || r.Position.Character < 0 {
			return errors.New("voice transcript step: rename requires non-negative position")
		}
		return nil
	case "code_action":
		if s.CodeActionDirective == nil || s.EditDirective != nil || s.CommandDirective != nil || s.NavigationDirective != nil || s.UndoDirective != nil ||
			s.RenameDirective != nil || s.FormatDirective != nil ||
			s.DeleteFileDirective != nil || s.MovePathDirective != nil || s.CreateFolderDirective != nil {
			return errors.New("voice transcript step: kind code_action requires codeActionDirective and no other directive payloads")
		}
		c := s.CodeActionDirective
		if strings.TrimSpace(c.Path) == "" || strings.TrimSpace(c.ActionKind) == "" {
			return errors.New("voice transcript step: code_action requires path and actionKind")
		}
		if c.Range != nil {
			if c.Range.StartLine < 0 || c.Range.StartChar < 0 || c.Range.EndLine < 0 || c.Range.EndChar < 0 {
				return errors.New("voice transcript step: code_action range positions must be non-negative")
			}
		}
		return nil
	case "format":
		if s.FormatDirective == nil || s.EditDirective != nil || s.CommandDirective != nil || s.NavigationDirective != nil || s.UndoDirective != nil ||
			s.RenameDirective != nil || s.CodeActionDirective != nil ||
			s.DeleteFileDirective != nil || s.MovePathDirective != nil || s.CreateFolderDirective != nil {
			return errors.New("voice transcript step: kind format requires formatDirective and no other directive payloads")
		}
		f := s.FormatDirective
		if strings.TrimSpace(f.Path) == "" {
			return errors.New("voice transcript step: format requires path")
		}
		scope := strings.TrimSpace(f.Scope)
		if scope != "document" && scope != "selection" {
			return fmt.Errorf("voice transcript step: format scope must be document or selection, got %q", f.Scope)
		}
		if f.Range != nil {
			if f.Range.StartLine < 0 || f.Range.StartChar < 0 || f.Range.EndLine < 0 || f.Range.EndChar < 0 {
				return errors.New("voice transcript step: format range positions must be non-negative")
			}
		}
		return nil
	case "delete_file":
		if s.DeleteFileDirective == nil || s.EditDirective != nil || s.CommandDirective != nil || s.NavigationDirective != nil || s.UndoDirective != nil ||
			s.RenameDirective != nil || s.CodeActionDirective != nil || s.FormatDirective != nil ||
			s.MovePathDirective != nil || s.CreateFolderDirective != nil {
			return errors.New("voice transcript step: kind delete_file requires deleteFileDirective and no other directive payloads")
		}
		if strings.TrimSpace(s.DeleteFileDirective.Path) == "" {
			return errors.New("voice transcript step: delete_file requires path")
		}
		return nil
	case "move_path":
		if s.MovePathDirective == nil || s.EditDirective != nil || s.CommandDirective != nil || s.NavigationDirective != nil || s.UndoDirective != nil ||
			s.RenameDirective != nil || s.CodeActionDirective != nil || s.FormatDirective != nil ||
			s.DeleteFileDirective != nil || s.CreateFolderDirective != nil {
			return errors.New("voice transcript step: kind move_path requires movePathDirective and no other directive payloads")
		}
		if strings.TrimSpace(s.MovePathDirective.From) == "" || strings.TrimSpace(s.MovePathDirective.To) == "" {
			return errors.New("voice transcript step: move_path requires from and to")
		}
		return nil
	case "create_folder":
		if s.CreateFolderDirective == nil || s.EditDirective != nil || s.CommandDirective != nil || s.NavigationDirective != nil || s.UndoDirective != nil ||
			s.RenameDirective != nil || s.CodeActionDirective != nil || s.FormatDirective != nil ||
			s.DeleteFileDirective != nil || s.MovePathDirective != nil {
			return errors.New("voice transcript step: kind create_folder requires createFolderDirective and no other directive payloads")
		}
		if strings.TrimSpace(s.CreateFolderDirective.Path) == "" {
			return errors.New("voice transcript step: create_folder requires path")
		}
		return nil
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

func (r VoiceTranscriptCompletion) Validate() error {
	if !r.Success && r.Summary != "" {
		return errors.New("voice transcript result must not include summary when success=false")
	}
	if !r.Success && strings.TrimSpace(r.TranscriptOutcome) != "" {
		return errors.New("voice transcript result must not include transcriptOutcome when success=false")
	}
	if len([]rune(r.Summary)) > 8192 {
		return errors.New("voice transcript result: summary exceeds max length")
	}

	out := strings.TrimSpace(r.TranscriptOutcome)
	if out != "" && out != "irrelevant" && out != "completed" && out != "clarify" && out != "clarify_control" &&
		out != "search" && out != "selection" && out != "selection_control" &&
		out != "file_selection" && out != "file_selection_control" && out != "needs_workspace_folder" && out != "answer" {
		return fmt.Errorf("voice transcript result: invalid transcriptOutcome %q", r.TranscriptOutcome)
	}
	if out == "answer" {
		if strings.TrimSpace(r.AnswerText) == "" {
			return errors.New("voice transcript result: answer requires answerText")
		}
		if len([]rune(r.AnswerText)) > 8192 {
			return errors.New("voice transcript result: answerText exceeds max length")
		}
	}
	if out == "search" || out == "selection" || out == "selection_control" {
		if r.SearchResults != nil && len(r.SearchResults) > 0 {
			for _, h := range r.SearchResults {
				if strings.TrimSpace(h.Path) == "" {
					return errors.New("voice transcript result: searchResults[].path is required")
				}
				if h.Line < 0 || h.Character < 0 {
					return errors.New("voice transcript result: searchResults[] requires non-negative line/character")
				}
				if strings.Contains(h.Preview, "\u0000") {
					return errors.New("voice transcript result: searchResults[].preview contains NUL")
				}
			}
		}
		if r.ActiveSearchIndex != nil && *r.ActiveSearchIndex < 0 {
			return errors.New("voice transcript result: activeSearchIndex must be non-negative")
		}
	}
	if out == "file_selection_control" {
		if strings.TrimSpace(r.FileSelectionFocusPath) == "" {
			return errors.New("voice transcript result: file_selection_control requires fileSelectionFocusPath")
		}
	}
	return nil
}
