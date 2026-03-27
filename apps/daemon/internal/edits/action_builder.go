package edits

import (
	"fmt"
	"path/filepath"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/actionplan"
	"vocoding.net/vocode/v2/apps/daemon/internal/symbols"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type ActionBuilder struct {
	validator      *Validator
	symbolResolver symbols.Resolver
}

func NewActionBuilder() *ActionBuilder {
	return NewActionBuilderWithResolver(symbols.NewTreeSitterResolver())
}

func NewActionBuilderWithResolver(resolver symbols.Resolver) *ActionBuilder {
	return &ActionBuilder{
		validator:      NewValidator(),
		symbolResolver: resolver,
	}
}

func (b *ActionBuilder) BuildActions(ctx EditExecutionContext, intent actionplan.EditIntent) ([]protocol.EditAction, *protocol.EditFailure) {
	switch intent.Kind {
	case actionplan.EditIntentKindInsert:
		action, failure := b.buildInsertStatementAction(ctx, intent)
		if failure != nil {
			return nil, failure
		}
		return []protocol.EditAction{replaceActionToEditAction(action)}, nil
	case actionplan.EditIntentKindReplace:
		target := intent.Replace.Target
		if target.Kind == actionplan.EditTargetKindAnchor {
			path := ctx.ActiveFile
			if p := strings.TrimSpace(target.Anchor.Path); p != "" {
				path = ctx.ResolvePath(p)
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
			if samePath(path, ctx.ActiveFile) {
				if failure := b.validator.ValidateAction(ctx.FileText, action); failure != nil {
					return nil, failure
				}
			}
			return []protocol.EditAction{replaceActionToEditAction(action)}, nil
		}
		action, failure := b.buildReplaceCurrentFunctionBodyAction(ctx, intent)
		if failure != nil {
			return nil, failure
		}
		return []protocol.EditAction{replaceActionToEditAction(action)}, nil
	case actionplan.EditIntentKindInsertImport:
		action, failure := b.buildAppendImportAction(ctx, intent)
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
		path := ctx.ActiveFile
		if p := strings.TrimSpace(target.Anchor.Path); p != "" {
			path = ctx.ResolvePath(p)
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
		if samePath(path, ctx.ActiveFile) {
			if failure := b.validator.ValidateAction(ctx.FileText, protocol.ReplaceBetweenAnchorsAction{
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

func (b *ActionBuilder) buildInsertStatementAction(ctx EditExecutionContext, intent actionplan.EditIntent) (protocol.ReplaceBetweenAnchorsAction, *protocol.EditFailure) {
	target := intent.Insert.Target
	path, fileText, failure := b.resolveFunctionSource(ctx, target)
	if failure != nil {
		return protocol.ReplaceBetweenAnchorsAction{}, failure
	}

	block, failure := b.resolveFunctionBlock(fileText, target)
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
		Path: path,
		Anchor: protocol.Anchor{
			Before: block.beforeLine,
			After:  block.afterAnchor,
		},
		NewText: newText,
	}

	if failure := b.validator.ValidateAction(fileText, action); failure != nil {
		return protocol.ReplaceBetweenAnchorsAction{}, failure
	}

	return action, nil
}

func (b *ActionBuilder) buildReplaceCurrentFunctionBodyAction(ctx EditExecutionContext, intent actionplan.EditIntent) (protocol.ReplaceBetweenAnchorsAction, *protocol.EditFailure) {
	target := intent.Replace.Target
	path, fileText, failure := b.resolveFunctionSource(ctx, target)
	if failure != nil {
		return protocol.ReplaceBetweenAnchorsAction{}, failure
	}
	block, failure := b.resolveFunctionBlock(fileText, target)
	if failure != nil {
		return protocol.ReplaceBetweenAnchorsAction{}, failure
	}
	newText := formatReplacementFunctionBody(block.indent, intent.Replace.NewText)
	action := protocol.ReplaceBetweenAnchorsAction{
		Kind: "replace_between_anchors",
		Path: path,
		Anchor: protocol.Anchor{
			Before: block.beforeLine,
			After:  block.afterAnchor,
		},
		NewText: newText,
	}
	if failure := b.validator.ValidateAction(fileText, action); failure != nil {
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

func (b *ActionBuilder) buildAppendImportAction(ctx EditExecutionContext, intent actionplan.EditIntent) (*protocol.ReplaceBetweenAnchorsAction, *protocol.EditFailure) {
	importStmt := intent.InsertImport.Import
	path := ctx.ActiveFile
	if p := strings.TrimSpace(intent.InsertImport.Path); p != "" {
		path = ctx.ResolvePath(p)
	}

	fileText, err := ctx.GetFileText(path)
	if err != nil {
		return nil, editFailure("missing_anchor", fmt.Sprintf("read target file %q: %v", path, err))
	}
	if strings.Contains(fileText, importStmt) {
		return nil, editFailure("no_change_needed", fmt.Sprintf("Import %q is already present.", importStmt))
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return b.buildGoImportAction(path, fileText, importStmt)
	case ".ts", ".tsx", ".js", ".jsx":
		return b.buildJSImportAction(path, fileText, importStmt)
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

func targetPathFromTarget(target actionplan.EditTarget) string {
	if target.Symbol != nil {
		return strings.TrimSpace(target.Symbol.Path)
	}
	if target.Anchor != nil {
		return strings.TrimSpace(target.Anchor.Path)
	}
	if target.Range != nil {
		return strings.TrimSpace(target.Range.Path)
	}
	return ""
}

func resolveEditSource(ctx EditExecutionContext, targetPath string) (string, string, *protocol.EditFailure) {
	path := ctx.ResolvePath(targetPath)
	if path == "." || path == "" {
		return "", "", editFailure("unsupported_instruction", "No file path available for edit target.")
	}
	fileText, err := ctx.GetFileText(path)
	if err != nil {
		return "", "", editFailure("missing_anchor", fmt.Sprintf("read target file %q: %v", path, err))
	}
	return path, fileText, nil
}

func (b *ActionBuilder) resolveFunctionBlock(fileText string, target actionplan.EditTarget) (*lineBlock, *protocol.EditFailure) {
	if target.Kind != actionplan.EditTargetKindSymbol || target.Symbol == nil {
		return findSingleFunctionBlock(fileText)
	}
	name := strings.TrimSpace(target.Symbol.SymbolName)
	if name == "" || name == "current_function" {
		return findSingleFunctionBlock(fileText)
	}
	return findNamedFunctionBlock(fileText, name)
}

func (b *ActionBuilder) resolveFunctionSource(
	ctx EditExecutionContext,
	target actionplan.EditTarget,
) (string, string, *protocol.EditFailure) {
	targetPath := targetPathFromTarget(target)
	// Explicit path from target: resolve directly.
	if strings.TrimSpace(targetPath) != "" {
		return resolveEditSource(ctx, targetPath)
	}

	if target.Kind != actionplan.EditTargetKindSymbol || target.Symbol == nil {
		return resolveEditSource(ctx, "")
	}
	name := strings.TrimSpace(target.Symbol.SymbolName)
	if name == "" || name == "current_function" {
		return resolveEditSource(ctx, "")
	}

	if b.symbolResolver == nil {
		return "", "", editFailure("unsupported_instruction", "No symbol resolver configured.")
	}
	kind := ""
	if target.Symbol != nil {
		kind = target.Symbol.SymbolKind
	}
	matches, err := b.symbolResolver.ResolveSymbol(ctx.WorkspaceRoot, name, kind, ctx.ActiveFile)
	if err != nil {
		return "", "", editFailure("unsupported_instruction", fmt.Sprintf("symbol resolution failed for %q: %v", name, err))
	}
	switch len(matches) {
	case 0:
		return "", "", editFailure("missing_anchor", fmt.Sprintf("Could not resolve symbol %q via tree-sitter.", name))
	case 1:
		return resolveEditSource(ctx, matches[0].Path)
	default:
		return "", "", editFailure("ambiguous_target", fmt.Sprintf("Function symbol %q matched multiple files; provide target path.", name))
	}
}
