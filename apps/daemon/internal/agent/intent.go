package agent

// EditIntentKind identifies the small deterministic edit slice the daemon can
// safely map today.
type EditIntentKind string

const (
	EditIntentInsertStatementInCurrentFunction EditIntentKind = "insert_statement_in_current_function"
	EditIntentReplaceAnchoredBlock             EditIntentKind = "replace_anchored_block"
	EditIntentAppendImportIfMissing            EditIntentKind = "append_import_if_missing"
)

// EditIntent is the instruction shape produced by the agent planner.
//
// Supported v1 intents:
//   - insert statement inside current function
//   - replace a selected/anchored block
//   - append import if missing
//
// The planner is intentionally rule-based so unsupported or ambiguous requests
// fail closed instead of guessing.
type EditIntent struct {
	Kind EditIntentKind

	Statement string
	Before    string
	After     string
	NewText   string
	Import    string
}
