package openai

import (
	"encoding/json"
	"testing"
)

func TestChatResponseFormatStrictAlways(t *testing.T) {
	t.Helper()
	b, err := marshalChatResponseFormatJSON()
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got["type"] != "json_schema" {
		t.Fatalf("got %v", got)
	}
	innerAny, ok := got["json_schema"]
	if !ok {
		t.Fatalf("missing json_schema: %v", got)
	}
	innerBytes, err := json.Marshal(innerAny)
	if err != nil {
		t.Fatal(err)
	}
	var inner struct {
		Name   string         `json:"name"`
		Strict bool           `json:"strict"`
		Schema map[string]any `json:"schema"`
	}
	if err := json.Unmarshal(innerBytes, &inner); err != nil {
		t.Fatal(err)
	}
	if inner.Name != "vocode_turn" || inner.Strict != false {
		t.Fatalf("%+v", inner)
	}
	if inner.Schema["type"] != "object" {
		t.Fatalf("schema.type: %v", inner.Schema["type"])
	}
}
