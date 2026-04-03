package router

import (
	"encoding/json"
	"fmt"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
)

// ClassifierSystem builds the system prompt for flow route classification.
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
{ "route": "<one of the route ids above>", "search_query": "<string or empty>" }

Rules:
- For "workspace_select", set "search_query" to the exact fixed string ripgrep should search for in file contents (no paraphrase, no conversational filler). Decide intent:
  - Literal text: user gave an exact phrase, error line, log snippet, comment text, or quoted string → use that substring (strip outer quotes only).
  - Code / symbol: user names a function, method, class, or identifier → use spellings that appear in source (e.g. name(, func name, type Name); drop filler words like "the", "function", "method" that will not appear in code. Never use vague English like "stuff function" when they mean a callable — use a code-shaped needle such as "stuff(" or "func stuff".
- For "select_file", set "search_query" to a path or filename fragment (e.g. "test.js", "src/api") used to match file paths under the workspace — not text inside files.
- For "workspace_select" and "select_file", "search_query" must be non-empty.
- For all other routes, set "search_query" to "".
- No extra keys. No markdown.
`)
	return strings.TrimSpace(b.String())
}

// ClassifierUserJSON is the minimal user payload for route classification (flow + utterance).
func ClassifierUserJSON(in Context) ([]byte, error) {
	type payload struct {
		Flow        flows.ID `json:"flow"`
		Instruction string   `json:"instruction"`
	}
	p := payload{
		Flow:        in.Flow,
		Instruction: strings.TrimSpace(in.Instruction),
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
				"type": "string",
				"description": "workspace_select: exact ripgrep --fixed-strings needle for content search (literal phrase or code-shaped token, not English paraphrase). " +
					"select_file: path/filename substring for workspace path matching. Otherwise empty.",
			},
		},
		"required":             []string{"route", "search_query"},
		"additionalProperties": false,
	}
}
