package router

import (
	"encoding/json"
	"fmt"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
)

// ClassifierSystem builds the system prompt for flow route classification.
// It starts from flows.Spec (intro + per-route descriptions), then appends shared JSON output Rules.
// Flow-specific tie-break bullets are appended below (workspace select, select file); workspace select uses classifier user JSON.
// flows.Execution policy is host metadata only; it must never appear here (or in user JSON / schema).
func ClassifierSystem(flow flows.ID) string {
	spec := flows.SpecFor(flow)
	var b strings.Builder
	b.WriteString(strings.TrimSpace(spec.Intro))
	b.WriteString("\n\nRoutes:\n")
	for _, r := range spec.Routes {
		b.WriteString(fmt.Sprintf("- %s: %s\n", r.ID, strings.TrimSpace(r.Description)))
	}
	b.WriteString(`
Return exactly ONE JSON object:
{ "route": "<one of the route ids above>", "search_query": "<string or empty>", "search_symbol_kind": "<string or empty>" }

Rules:
- For "workspace_select", set "search_query" to the primary symbol or identifier name only (e.g. deltaTime, parseHeader, MyClass) — not a prose phrase like "delta time".
  - Exception — literal text search: user gave an exact phrase, error line, log snippet, comment text, or quoted string to find verbatim in files → put that substring in "search_query" (strip outer quotes only) and omit "search_symbol_kind".
  - Optional "search_symbol_kind" (workspace_select only): when you know what kind of symbol they mean, set one of: function, method, class, variable, constant, interface, enum, property, field, constructor, module, struct, type. Omit or use "" when unsure; never guess if ambiguous.
- For "select_file", set "search_query" to a single file or folder name only (basename): e.g. game.js, README.md, Res. No slashes, no path segments, no absolute paths — never paste activeFile or any full system path. STT may say "dot" for a period in the name — rewrite to real punctuation in that one segment only. Set "search_symbol_kind" to "".
- For "workspace_select" and "select_file", "search_query" must be non-empty.
- For all other routes, set "search_query" to "" and "search_symbol_kind" to "".
- "command" vs "question": use "command" when they want something executed in the terminal (install, run, build, test, git, npx/pnpm scaffold, start dev server). Use "question" when they want an explanation or how-to without running a command.
- No other keys. No markdown.
`)
	if flow == flows.WorkspaceSelect {
		b.WriteString(`

Workspace select — user JSON may include hasNonemptySelection (true when the editor selection is non-empty) and activeFile:
- When hasNonemptySelection is true, a selection does not by itself mean "edit": if the utterance matches global "create" or "rename" per the route list, prefer those when appropriate.
- When hasNonemptySelection is true, the utterance is an imperative to change existing code, and they are not asking to find or search the workspace, prefer "edit" over "workspace_select".
`)
	}
	if flow == flows.SelectFile {
		b.WriteString(`

Select file — global "create" vs "create_entry": user JSON flow is select_file; activeFile may still be set from the editor. Use create_entry when they name a new file or folder on disk under the list row (add, make, create, new + a name; STT may say "dot" for "."). Use create only for changing the open editor buffer (code, comments, imports, placement), not for creating a path from this flow.
`)
	}
	return strings.TrimSpace(b.String())
}

// ClassifierUserJSON is the minimal user payload for route classification (flow + utterance).
func ClassifierUserJSON(in Context) ([]byte, error) {
	type payload struct {
		Flow                 flows.ID `json:"flow"`
		Instruction          string   `json:"instruction"`
		ActiveFile           string   `json:"activeFile,omitempty"`
		HasNonemptySelection bool     `json:"hasNonemptySelection,omitempty"`
		WorkspaceRoot        string   `json:"workspaceRoot,omitempty"`
		HostPlatform         string   `json:"hostPlatform,omitempty"`
		WorkspaceFolderOpen  bool     `json:"workspaceFolderOpen,omitempty"`
	}
	p := payload{
		Flow:                 in.Flow,
		Instruction:          strings.TrimSpace(in.Instruction),
		ActiveFile:           strings.TrimSpace(in.ActiveFile),
		HasNonemptySelection: in.HasNonemptySelection,
		WorkspaceRoot:        strings.TrimSpace(in.WorkspaceRoot),
		HostPlatform:         strings.TrimSpace(in.HostPlatform),
		WorkspaceFolderOpen:  in.WorkspaceFolderOpen,
	}
	return json.MarshalIndent(p, "", "  ")
}

// ClassifierResponseJSONSchema is the JSON Schema for classification (passed to the model client).
func ClassifierResponseJSONSchema(flow flows.ID) map[string]any {
	routes := flows.SpecFor(flow).RouteIDs()
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"route": map[string]any{
				"type": "string",
				"enum": routes,
			},
			"search_query": map[string]any{
				"type":        "string",
				"description": "workspace_select: symbol/identifier name or exact literal substring to find in file contents. select_file: single file or folder basename only (no slashes, no absolute path). command, create_entry, create, move, rename, delete, control, irrelevant, question: always empty.",
			},
			"search_symbol_kind": map[string]any{
				"type":        "string",
				"description": "workspace_select only: optional kind of symbol they mean — function, method, class, variable, constant, interface, enum, property, field, constructor, module, struct, or type. Empty when unknown or for select_file.",
			},
		},
		"required":             []string{"route", "search_query", "search_symbol_kind"},
		"additionalProperties": false,
	}
}
