package openai

func chatResponseFormatFromSchema(schema map[string]any) *responseFormat {
	return &responseFormat{
		Type: "json_schema",
		JSONSchema: &namedJSONSchema{
			Name:   "vocode_structured_response",
			Strict: false,
			Schema: schema,
		},
	}
}

type responseFormat struct {
	Type       string           `json:"type"`
	JSONSchema *namedJSONSchema `json:"json_schema,omitempty"`
}

type namedJSONSchema struct {
	Name   string `json:"name"`
	Strict bool   `json:"strict"`
	Schema any    `json:"schema"`
}
