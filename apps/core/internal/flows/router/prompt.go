package router

import (
	"encoding/json"
	"fmt"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
)

// ClassifierSystem builds the system prompt for flow route classification.
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
{ "route": "<one of the route ids above>" }

Rules:
- Output only the route id. Do not extract search strings, paths, numbers, or questions here — handlers do that after routing.
- Ambiguity and follow-up questions are handled by route handlers (not a separate "clarify" route).
- No extra keys. No markdown.
`)
	return strings.TrimSpace(b.String())
}

// ClassifierUserJSON builds user context for flow route classification.
func ClassifierUserJSON(in Context) ([]byte, error) {
	activeFile := strings.TrimSpace(in.Editor.ActiveFilePath)
	workspaceRoot := strings.TrimSpace(in.Editor.WorkspaceRoot)
	var cursor *struct {
		Name string `json:"name,omitempty"`
		Kind string `json:"kind,omitempty"`
	}
	if in.Editor.CursorSymbol != nil {
		cursor = &struct {
			Name string `json:"name,omitempty"`
			Kind string `json:"kind,omitempty"`
		}{Name: strings.TrimSpace(in.Editor.CursorSymbol.Name), Kind: strings.TrimSpace(in.Editor.CursorSymbol.Kind)}
	}

	type payload struct {
		Flow          flows.ID `json:"flow"`
		Instruction   string   `json:"instruction"`
		ActiveFile    string   `json:"activeFile"`
		WorkspaceRoot string   `json:"workspaceRoot"`
		CursorSymbol  *struct {
			Name string `json:"name,omitempty"`
			Kind string `json:"kind,omitempty"`
		} `json:"cursorSymbol,omitempty"`
		HitCount        int    `json:"hitCount,omitempty"`
		ActiveIndex     int    `json:"activeIndex,omitempty"`
		FocusPath       string `json:"focusPath,omitempty"`
		ListCount       int    `json:"listCount,omitempty"`
		ListActiveIndex int    `json:"listActiveIndex,omitempty"`
	}

	p := payload{
		Flow:          in.Flow,
		Instruction:   strings.TrimSpace(in.Instruction),
		ActiveFile:    activeFile,
		WorkspaceRoot: workspaceRoot,
		CursorSymbol:  cursor,
	}
	switch in.Flow {
	case flows.Select:
		p.HitCount = in.HitCount
		p.ActiveIndex = in.ActiveIndex
	case flows.SelectFile:
		p.FocusPath = strings.TrimSpace(in.FocusPath)
		p.ListCount = in.ListCount
		p.ListActiveIndex = in.ListActiveIndex
	}
	return json.MarshalIndent(p, "", "  ")
}

// ClassifierResponseJSONSchema is the JSON Schema for route-only classification (passed to the model client).
func ClassifierResponseJSONSchema(flow flows.ID) map[string]any {
	routes := flows.SpecFor(flow).RouteIDs()
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"route": map[string]any{
				"type": "string",
				"enum": routes,
			},
		},
		"required":             []string{"route"},
		"additionalProperties": false,
	}
}
