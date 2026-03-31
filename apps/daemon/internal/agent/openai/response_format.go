package openai

import (
	"encoding/json"
)

// chatResponseFormatScopedEdit picks OpenAI Chat Completions response_format for scoped edits.
func chatResponseFormatScopedEdit() *responseFormat {
	return &responseFormat{
		Type: "json_schema",
		JSONSchema: &namedJSONSchema{
			Name: "vocode_scoped_edit",
			// Strict=false: schema guides the model shape, but we rely on Go decode + validation for hard validation.
			Strict: false,
			Schema: scopedEditJSONSchema(),
		},
	}
}

// chatResponseFormatScopeIntent picks OpenAI Chat Completions response_format for scope intent classification.
func chatResponseFormatScopeIntent() *responseFormat {
	return &responseFormat{
		Type: "json_schema",
		JSONSchema: &namedJSONSchema{
			Name:   "vocode_scope_intent",
			Strict: false,
			Schema: scopeIntentJSONSchema(),
		},
	}
}

// chatResponseFormatTranscriptClassifier picks OpenAI Chat Completions response_format for first-pass routing.
func chatResponseFormatTranscriptClassifier() *responseFormat {
	return &responseFormat{
		Type: "json_schema",
		JSONSchema: &namedJSONSchema{
			Name:   "vocode_transcript_classifier",
			Strict: false,
			Schema: transcriptClassifierJSONSchema(),
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

func scopedEditJSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"replacementText": map[string]any{"type": "string"},
		},
		"required": []string{"replacementText"},
		// OpenAI structured outputs require additionalProperties=false at the top level.
		"additionalProperties": false,
	}
}

func scopeIntentJSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"scopeKind": map[string]any{
				"type": "string",
				"enum": []string{"current_function", "current_file", "named_symbol", "clarify"},
			},
			"symbolName":       map[string]any{"type": "string"},
			"clarifyQuestion":  map[string]any{"type": "string"},
		},
		"required":             []string{"scopeKind"},
		"additionalProperties": false,
	}
}

func transcriptClassifierJSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"kind": map[string]any{
				"type": "string",
				"enum": []string{"instruction", "search", "question", "irrelevant"},
			},
			"searchQuery": map[string]any{"type": "string"},
			"answerText":  map[string]any{"type": "string"},
		},
		"required":             []string{"kind"},
		"additionalProperties": false,
	}
}

// marshalChatResponseFormatJSON exists for tests (stable shape without building a full request).
func marshalChatResponseFormatJSON() ([]byte, error) {
	return json.Marshal(chatResponseFormatScopedEdit())
}
