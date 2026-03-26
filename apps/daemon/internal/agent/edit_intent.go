// Package agent holds voice/agent runtime types and the edit intent model used
// by internal/edits to build protocol EditActions. There is no edit.apply RPC;
// integration is covered by Go tests (see internal/edits, internal/orchestration).
package agent

import (
	"fmt"
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// EditIntentKind identifies the small deterministic edit slice the daemon can
// safely map today.
type EditIntentKind string

const (
	EditIntentInsertStatementInCurrentFunction EditIntentKind = "insert_statement_in_current_function"
	EditIntentReplaceCurrentFunctionBody       EditIntentKind = "replace_current_function_body"
	EditIntentReplaceAnchoredBlock             EditIntentKind = "replace_anchored_block"
	EditIntentAppendImportIfMissing            EditIntentKind = "append_import_if_missing"
)

// EditIntent is structured edit semantics (from a future LLM or tests) before
// file-grounded actions are built. It appears inside [ActionPlan] as an edit
// [Step] (kind "edit").
//
// Supported v1 intents:
//   - insert statement inside current function
//   - replace entire body of the current (single unambiguous) function
//   - replace a selected/anchored block
//   - append import if missing
type EditIntent struct {
	Kind EditIntentKind `json:"kind"`

	Statement string `json:"statement,omitempty"`
	Before    string `json:"before,omitempty"`
	After     string `json:"after,omitempty"`
	NewText   string `json:"newText,omitempty"`
	Import    string `json:"import,omitempty"`
}

// PlanEditResult is the outcome of planning (or a stub planner in tests): either
// a concrete intent or a structured failure.
type PlanEditResult struct {
	Intent  *EditIntent
	Failure *protocol.EditFailure
}

// ValidateEditIntent checks required fields per EditIntentKind.
func ValidateEditIntent(intent EditIntent) error {
	switch intent.Kind {
	case EditIntentInsertStatementInCurrentFunction:
		if strings.TrimSpace(intent.Statement) == "" {
			return fmt.Errorf("edit intent: insert_statement_in_current_function requires non-empty statement")
		}
	case EditIntentReplaceCurrentFunctionBody:
		if strings.TrimSpace(intent.NewText) == "" {
			return fmt.Errorf("edit intent: replace_current_function_body requires non-empty newText")
		}
	case EditIntentReplaceAnchoredBlock:
		if strings.TrimSpace(intent.Before) == "" || strings.TrimSpace(intent.After) == "" {
			return fmt.Errorf("edit intent: replace_anchored_block requires non-empty before and after anchors")
		}
	case EditIntentAppendImportIfMissing:
		if strings.TrimSpace(intent.Import) == "" {
			return fmt.Errorf("edit intent: append_import_if_missing requires non-empty import")
		}
		if !strings.HasPrefix(strings.TrimSpace(intent.Import), "import ") {
			return fmt.Errorf("edit intent: import must be a full import statement starting with %q", "import ")
		}
	default:
		return fmt.Errorf("edit intent: unknown kind %q", intent.Kind)
	}
	return nil
}
