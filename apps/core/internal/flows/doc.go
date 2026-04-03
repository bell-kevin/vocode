// Package flows defines voice/transcript flows: flow IDs, composed route specs (global routes
// plus flow-specific routes), validation, and small flow helpers.
//
// Package globalflow (flows/global) implements handlers for global route ids: control,
// workspace_select, select_file, irrelevant — used from rootflow, selectflow, and selectfileflow dispatchers.
//
// Subpackage router classifies transcripts to a route id. selectflow/selectfileflow use
// dispatch.go for wiring and control.go only for flow-local control routes (workspace_select_control,
// file_select_control). Per-route stubs for non-global routes may live in edit.go, open.go, etc.
//
// Package selection holds shared list-navigation parsing ([selection.ParseNav]).
package flows
