// Package flows defines voice/transcript flows: flow IDs, composed route specs (global routes
// plus flow-specific routes), validation, and small flow helpers.
//
// Package globalflow (flows/global) implements handlers for global route ids: control,
// workspace_select, select_file, command, irrelevant — used from rootflow, selectflow, and selectfileflow dispatchers.
//
// Subpackage router classifies transcripts to a route id (via the configured model client).
// selectflow/selectfileflow use dispatch.go for wiring and control.go for flow-local control routes
// (workspace_select_control, file_select_control). Route handlers for non-global routes live in
// edit.go, open.go, etc.
//
// Package selection holds shared list-navigation parsing ([selection.ParseNav]).
//
// File-selection flow: create_entry creates a new file under the focused path (including workspace root).
// move_path (move route) may create parent directories on the host; create_entry does not create empty folders.
// Workspace root cannot be deleted, renamed, or moved via file-select routes (same idea as VS Code).
package flows
