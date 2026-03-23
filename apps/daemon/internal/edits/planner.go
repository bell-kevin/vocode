package edits

import (
	"fmt"
	"path/filepath"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type Planner struct {
	validator *Validator
}

func NewPlanner() *Planner {
	return &Planner{validator: NewValidator()}
}

func (p *Planner) BuildActions(params protocol.EditApplyParams, plan agent.EditPlan) ([]protocol.EditAction, *protocol.EditFailure) {
	switch plan.Intent.Kind {
	case agent.EditIntentInsertStatementInCurrentFunction:
		action, failure := p.buildInsertStatementAction(params, plan.Intent)
		if failure != nil {
			return nil, failure
		}
		return []protocol.EditAction{action}, nil
	case agent.EditIntentReplaceAnchoredBlock:
		action := protocol.ReplaceBetweenAnchorsAction{
			Kind: "replace_between_anchors",
			Path: params.ActiveFile,
			Anchor: protocol.Anchor{
				Before: plan.Intent.Before,
				After:  plan.Intent.After,
			},
			NewText: plan.Intent.NewText,
		}
		if failure := p.validator.ValidateAction(params.FileText, action); failure != nil {
			return nil, failure
		}
		return []protocol.EditAction{action}, nil
	case agent.EditIntentAppendImportIfMissing:
		action, failure := p.buildAppendImportAction(params, plan.Intent)
		if failure != nil {
			return nil, failure
		}
		if action == nil {
			return []protocol.EditAction{}, nil
		}
		return []protocol.EditAction{*action}, nil
	default:
		return nil, editFailure("unsupported_instruction", fmt.Sprintf("Unsupported intent kind %q.", plan.Intent.Kind))
	}
}

func (p *Planner) buildInsertStatementAction(params protocol.EditApplyParams, intent agent.EditIntent) (protocol.ReplaceBetweenAnchorsAction, *protocol.EditFailure) {
	block, failure := findSingleFunctionBlock(params.FileText)
	if failure != nil {
		return protocol.ReplaceBetweenAnchorsAction{}, failure
	}

	statement := strings.TrimSpace(intent.Statement)
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

	if failure := p.validator.ValidateAction(params.FileText, action); failure != nil {
		return protocol.ReplaceBetweenAnchorsAction{}, failure
	}

	return action, nil
}

func (p *Planner) buildAppendImportAction(params protocol.EditApplyParams, intent agent.EditIntent) (*protocol.ReplaceBetweenAnchorsAction, *protocol.EditFailure) {
	if strings.Contains(params.FileText, intent.Import) {
		return nil, editFailure("no_change_needed", fmt.Sprintf("Import %q is already present.", intent.Import))
	}

	ext := strings.ToLower(filepath.Ext(params.ActiveFile))
	switch ext {
	case ".go":
		return p.buildGoImportAction(params, intent.Import)
	case ".ts", ".tsx", ".js", ".jsx":
		return p.buildJSImportAction(params, intent.Import)
	default:
		return nil, editFailure("unsupported_instruction", fmt.Sprintf("Append import is not supported for %q files yet.", ext))
	}
}

func (p *Planner) buildGoImportAction(params protocol.EditApplyParams, importStmt string) (*protocol.ReplaceBetweenAnchorsAction, *protocol.EditFailure) {
	lines := strings.Split(params.FileText, "\n")
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
			Path:    params.ActiveFile,
			Anchor:  protocol.Anchor{Before: lines[i], After: lines[closeIndex]},
			NewText: newText,
		}
		if failure := p.validator.ValidateAction(params.FileText, action); failure != nil {
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
		Path:    params.ActiveFile,
		Anchor:  protocol.Anchor{Before: before, After: after},
		NewText: "\n\n" + importStmt + "\n",
	}
	if failure := p.validator.ValidateAction(params.FileText, action); failure != nil {
		return nil, failure
	}
	return &action, nil
}

func (p *Planner) buildJSImportAction(params protocol.EditApplyParams, importStmt string) (*protocol.ReplaceBetweenAnchorsAction, *protocol.EditFailure) {
	lines := strings.Split(params.FileText, "\n")
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
			Path:    params.ActiveFile,
			Anchor:  protocol.Anchor{Before: before, After: after},
			NewText: "\n" + importStmt + "\n",
		}
		if failure := p.validator.ValidateAction(params.FileText, action); failure != nil {
			return nil, failure
		}
		return &action, nil
	}

	if len(lines) < 2 {
		return nil, editFailure("missing_anchor", "Could not find a safe insertion point at the top of the file.")
	}

	action := protocol.ReplaceBetweenAnchorsAction{
		Kind:    "replace_between_anchors",
		Path:    params.ActiveFile,
		Anchor:  protocol.Anchor{Before: lines[0], After: strings.Join(lines[1:], "\n")},
		NewText: "\n" + importStmt + "\n",
	}
	if failure := p.validator.ValidateAction(params.FileText, action); failure != nil {
		return nil, failure
	}
	return &action, nil
}
