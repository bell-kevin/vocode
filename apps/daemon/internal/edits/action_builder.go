package edits

import (
	"fmt"
	"path/filepath"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/actionplan"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type ActionBuilder struct {
	validator *Validator
}

func NewActionBuilder() *ActionBuilder {
	return &ActionBuilder{validator: NewValidator()}
}

func (b *ActionBuilder) BuildActions(params protocol.EditApplyParams, intent actionplan.EditIntent) ([]protocol.EditAction, *protocol.EditFailure) {
	switch intent.Kind {
	case actionplan.EditIntentKindInsert:
		action, failure := b.buildInsertStatementAction(params, intent)
		if failure != nil {
			return nil, failure
		}
		return []protocol.EditAction{replaceActionToEditAction(action)}, nil
	case actionplan.EditIntentKindReplace:
		target := intent.Replace.Target
		if target.Kind == actionplan.EditTargetKindAnchor {
			path := params.ActiveFile
			if p := strings.TrimSpace(target.Anchor.Path); p != "" {
				path = p
			}
			action := protocol.ReplaceBetweenAnchorsAction{
				Kind: "replace_between_anchors",
				Path: path,
				Anchor: protocol.Anchor{
					Before: target.Anchor.Before,
					After:  target.Anchor.After,
				},
				NewText: intent.Replace.NewText,
			}
			if samePath(path, params.ActiveFile) {
				if failure := b.validator.ValidateAction(params.FileText, action); failure != nil {
					return nil, failure
				}
			}
			return []protocol.EditAction{replaceActionToEditAction(action)}, nil
		}
		action, failure := b.buildReplaceCurrentFunctionBodyAction(params, intent)
		if failure != nil {
			return nil, failure
		}
		return []protocol.EditAction{replaceActionToEditAction(action)}, nil
	case actionplan.EditIntentKindInsertImport:
		action, failure := b.buildAppendImportAction(params, intent)
		if failure != nil {
			return nil, failure
		}
		if action == nil {
			return []protocol.EditAction{}, nil
		}
		return []protocol.EditAction{replaceActionToEditAction(*action)}, nil
	case actionplan.EditIntentKindDelete:
		target := intent.Delete.Target
		if target.Kind != actionplan.EditTargetKindAnchor || target.Anchor == nil {
			return nil, editFailure("unsupported_instruction", "Delete currently supports only anchor targets.")
		}
		path := params.ActiveFile
		if p := strings.TrimSpace(target.Anchor.Path); p != "" {
			path = p
		}
		action := protocol.EditAction{
			Kind: "replace_between_anchors",
			Path: path,
			Anchor: &protocol.Anchor{
				Before: target.Anchor.Before,
				After:  target.Anchor.After,
			},
			NewText: "",
		}
		if samePath(path, params.ActiveFile) {
			if failure := b.validator.ValidateAction(params.FileText, protocol.ReplaceBetweenAnchorsAction{
				Kind:    action.Kind,
				Path:    action.Path,
				Anchor:  *action.Anchor,
				NewText: action.NewText,
			}); failure != nil {
				return nil, failure
			}
		}
		return []protocol.EditAction{action}, nil
	case actionplan.EditIntentKindCreateFile:
		return []protocol.EditAction{{
			Kind:    "create_file",
			Path:    intent.CreateFile.Path,
			Content: intent.CreateFile.Content,
		}}, nil
	case actionplan.EditIntentKindAppendToFile:
		return []protocol.EditAction{{
			Kind: "append_to_file",
			Path: intent.AppendToFile.Path,
			Text: intent.AppendToFile.Text,
		}}, nil
	default:
		return nil, editFailure("unsupported_instruction", fmt.Sprintf("Unsupported intent kind %q.", intent.Kind))
	}
}

func (b *ActionBuilder) buildInsertStatementAction(params protocol.EditApplyParams, intent actionplan.EditIntent) (protocol.ReplaceBetweenAnchorsAction, *protocol.EditFailure) {
	block, failure := findSingleFunctionBlock(params.FileText)
	if failure != nil {
		return protocol.ReplaceBetweenAnchorsAction{}, failure
	}

	statement := strings.TrimSpace(intent.Insert.Text)
	if statement == "" {
		return protocol.ReplaceBetweenAnchorsAction{}, editFailure("unsupported_instruction", "Insert statement instruction did not include a statement.")
	}
	statement = strings.TrimRight(statement, "\n")
	if !strings.HasSuffix(statement, ";") {
		statement += ";"
	}

	newText := block.between
	if strings.TrimSpace(newText) == "" {
		newText = "\n"
	}
	if newText == "\n" {
		newText += block.indent + statement + "\n"
	} else {
		newText += block.indent + statement + "\n"
	}

	action := protocol.ReplaceBetweenAnchorsAction{
		Kind: "replace_between_anchors",
		Path: params.ActiveFile,
		Anchor: protocol.Anchor{
			Before: block.beforeLine,
			After:  block.afterAnchor,
		},
		NewText: newText,
	}

	if failure := b.validator.ValidateAction(params.FileText, action); failure != nil {
		return protocol.ReplaceBetweenAnchorsAction{}, failure
	}

	return action, nil
}

func (b *ActionBuilder) buildReplaceCurrentFunctionBodyAction(params protocol.EditApplyParams, intent actionplan.EditIntent) (protocol.ReplaceBetweenAnchorsAction, *protocol.EditFailure) {
	block, failure := findSingleFunctionBlock(params.FileText)
	if failure != nil {
		return protocol.ReplaceBetweenAnchorsAction{}, failure
	}
	newText := formatReplacementFunctionBody(block.indent, intent.Replace.NewText)
	action := protocol.ReplaceBetweenAnchorsAction{
		Kind: "replace_between_anchors",
		Path: params.ActiveFile,
		Anchor: protocol.Anchor{
			Before: block.beforeLine,
			After:  block.afterAnchor,
		},
		NewText: newText,
	}
	if failure := b.validator.ValidateAction(params.FileText, action); failure != nil {
		return protocol.ReplaceBetweenAnchorsAction{}, failure
	}
	return action, nil
}

func formatReplacementFunctionBody(indent, body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return "\n"
	}
	lines := strings.Split(body, "\n")
	var out strings.Builder
	out.WriteByte('\n')
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" {
			out.WriteByte('\n')
			continue
		}
		out.WriteString(indent)
		out.WriteString(trimmed)
		out.WriteByte('\n')
	}
	return out.String()
}

func (b *ActionBuilder) buildAppendImportAction(params protocol.EditApplyParams, intent actionplan.EditIntent) (*protocol.ReplaceBetweenAnchorsAction, *protocol.EditFailure) {
	importStmt := intent.InsertImport.Import
	if strings.Contains(params.FileText, importStmt) {
		return nil, editFailure("no_change_needed", fmt.Sprintf("Import %q is already present.", importStmt))
	}

	path := params.ActiveFile
	if p := strings.TrimSpace(intent.InsertImport.Path); p != "" {
		path = p
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return b.buildGoImportAction(path, params.FileText, importStmt)
	case ".ts", ".tsx", ".js", ".jsx":
		return b.buildJSImportAction(path, params.FileText, importStmt)
	default:
		return nil, editFailure("unsupported_instruction", fmt.Sprintf("Append import is not supported for %q files yet.", ext))
	}
}

func (b *ActionBuilder) buildGoImportAction(path, fileText, importStmt string) (*protocol.ReplaceBetweenAnchorsAction, *protocol.EditFailure) {
	lines := strings.Split(fileText, "\n")
	packageIndex := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "package ") {
			packageIndex = i
			break
		}
	}
	if packageIndex == -1 {
		return nil, editFailure("missing_anchor", "Could not find a package declaration for Go import insertion.")
	}

	for i, line := range lines {
		if strings.TrimSpace(line) != "import (" {
			continue
		}
		closeIndex := -1
		for j := i + 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == ")" {
				closeIndex = j
				break
			}
		}
		if closeIndex == -1 {
			return nil, editFailure("missing_anchor", "Could not find the end of the Go import block.")
		}

		between := "\n"
		if closeIndex > i+1 {
			between += strings.Join(lines[i+1:closeIndex], "\n")
			between += "\n"
		}
		newText := between + "\t" + strings.TrimPrefix(importStmt, "import ") + "\n"
		action := protocol.ReplaceBetweenAnchorsAction{
			Kind:    "replace_between_anchors",
			Path:    path,
			Anchor:  protocol.Anchor{Before: lines[i], After: lines[closeIndex]},
			NewText: newText,
		}
		if failure := b.validator.ValidateAction(fileText, action); failure != nil {
			return nil, failure
		}
		return &action, nil
	}

	before := lines[packageIndex]
	after := strings.Join(lines[packageIndex+1:], "\n")
	if after == "" {
		return nil, editFailure("missing_anchor", "Could not find a safe insertion point after the package declaration.")
	}

	action := protocol.ReplaceBetweenAnchorsAction{
		Kind:    "replace_between_anchors",
		Path:    path,
		Anchor:  protocol.Anchor{Before: before, After: after},
		NewText: "\n\n" + importStmt + "\n",
	}
	if failure := b.validator.ValidateAction(fileText, action); failure != nil {
		return nil, failure
	}
	return &action, nil
}

func (b *ActionBuilder) buildJSImportAction(path, fileText, importStmt string) (*protocol.ReplaceBetweenAnchorsAction, *protocol.EditFailure) {
	lines := strings.Split(fileText, "\n")
	lastImportIndex := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "import ") {
			lastImportIndex = i
			continue
		}
		if strings.TrimSpace(line) != "" && lastImportIndex != -1 {
			break
		}
	}

	if lastImportIndex != -1 {
		before := lines[lastImportIndex]
		after := strings.Join(lines[lastImportIndex+1:], "\n")
		if after == "" {
			return nil, editFailure("missing_anchor", "Could not find a safe insertion point after the final import.")
		}
		action := protocol.ReplaceBetweenAnchorsAction{
			Kind:    "replace_between_anchors",
			Path:    path,
			Anchor:  protocol.Anchor{Before: before, After: after},
			NewText: "\n" + importStmt + "\n",
		}
		if failure := b.validator.ValidateAction(fileText, action); failure != nil {
			return nil, failure
		}
		return &action, nil
	}

	if len(lines) < 2 {
		return nil, editFailure("missing_anchor", "Could not find a safe insertion point at the top of the file.")
	}

	action := protocol.ReplaceBetweenAnchorsAction{
		Kind:    "replace_between_anchors",
		Path:    path,
		Anchor:  protocol.Anchor{Before: lines[0], After: strings.Join(lines[1:], "\n")},
		NewText: "\n" + importStmt + "\n",
	}
	if failure := b.validator.ValidateAction(fileText, action); failure != nil {
		return nil, failure
	}
	return &action, nil
}

func replaceActionToEditAction(a protocol.ReplaceBetweenAnchorsAction) protocol.EditAction {
	return protocol.EditAction{
		Kind:    a.Kind,
		Path:    a.Path,
		Anchor:  &a.Anchor,
		NewText: a.NewText,
	}
}

func samePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
