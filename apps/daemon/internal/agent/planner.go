package agent

import (
	"fmt"
	"regexp"
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

var (
	quotedValuePattern = regexp.MustCompile("(?s)(?:\"([^\"]+)\"|`([^`]+)`)")

	insertStatementPattern = regexp.MustCompile(
		`(?is)^insert(?:\s+statement)?\s+(?:\"([^\"]+)\"|` + "`([^`]+)`" + `)\s+inside\s+current\s+function$`,
	)
	replaceAnchoredBlockPattern = regexp.MustCompile(
		`(?is)^replace\s+block\s+after\s+(?:\"([^\"]+)\"|` + "`([^`]+)`" + `)\s+before\s+(?:\"([^\"]+)\"|` + "`([^`]+)`" + `)\s+with\s+(?:\"([^\"]+)\"|` + "`([^`]+)`" + `)$`,
	)
	appendImportPattern = regexp.MustCompile(
		`(?is)^(?:append|add)\s+import\s+(?:\"([^\"]+)\"|` + "`([^`]+)`" + `)\s+if\s+missing$`,
	)
)

type Planner struct{}

func NewPlanner() *Planner {
	return &Planner{}
}

func (p *Planner) Plan(params protocol.EditApplyParams) EditPlanResult {
	instruction := strings.TrimSpace(params.Instruction)
	if instruction == "" {
		return EditPlanResult{Failure: failure("unsupported_instruction", "Instruction was empty.")}
	}

	if matches := insertStatementPattern.FindStringSubmatch(instruction); matches != nil {
		statement := firstNonEmpty(matches[1], matches[2])
		return EditPlanResult{Plan: &EditPlan{Intent: EditIntent{
			Kind:      EditIntentInsertStatementInCurrentFunction,
			Statement: strings.TrimSpace(statement),
		}}}
	}

	if matches := replaceAnchoredBlockPattern.FindStringSubmatch(instruction); matches != nil {
		before := firstNonEmpty(matches[1], matches[2])
		after := firstNonEmpty(matches[3], matches[4])
		newText := firstNonEmpty(matches[5], matches[6])
		return EditPlanResult{Plan: &EditPlan{Intent: EditIntent{
			Kind:    EditIntentReplaceAnchoredBlock,
			Before:  before,
			After:   after,
			NewText: newText,
		}}}
	}

	if matches := appendImportPattern.FindStringSubmatch(instruction); matches != nil {
		statement := strings.TrimSpace(firstNonEmpty(matches[1], matches[2]))
		if !strings.HasPrefix(statement, "import ") {
			return EditPlanResult{Failure: failure(
				"unsupported_instruction",
				fmt.Sprintf("Import instruction must include a full import statement, got %q.", statement),
			)}
		}
		return EditPlanResult{Plan: &EditPlan{Intent: EditIntent{
			Kind:   EditIntentAppendImportIfMissing,
			Import: statement,
		}}}
	}

	return EditPlanResult{Failure: failure(
		"unsupported_instruction",
		"Supported instructions: insert statement \"...\" inside current function; replace block after \"...\" before \"...\" with \"...\"; append import \"...\" if missing.",
	)}
}

func ExtractQuotedValues(input string) []string {
	matches := quotedValuePattern.FindAllStringSubmatch(input, -1)
	values := make([]string, 0, len(matches))
	for _, match := range matches {
		value := firstNonEmpty(match[1], match[2])
		if strings.TrimSpace(value) == "" {
			continue
		}
		values = append(values, value)
	}
	return values
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func failure(code string, message string) *protocol.EditFailure {
	return &protocol.EditFailure{Code: code, Message: message}
}
