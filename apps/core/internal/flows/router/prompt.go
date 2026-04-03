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
- For routes "select" and "select_file" you MUST set "search_query" to the exact literal string to pass to ripgrep (workspace content search for "select", path/name fragment for "select_file"). Do not leave it empty for those routes.
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
				"type":        "string",
				"description": "For routes select and select_file: ripgrep query. Otherwise empty.",
			},
		},
		"required":             []string{"route", "search_query"},
		"additionalProperties": false,
	}
}
