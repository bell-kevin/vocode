package router

import (
	"slices"
	"strings"
	"testing"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
)

func TestClassifierSystem_rootContainsTieBreaksAndRoutes(t *testing.T) {
	s := ClassifierSystem(flows.Root)
	for _, sub := range []string{
		"ROOT flow",
		"speech-to-text",
		"Tie-breaks (ROOT):",
		"Compound utterance",
		"Return exactly ONE JSON object:",
		`"control" vs "irrelevant"`,
	} {
		if !strings.Contains(s, sub) {
			t.Errorf("root ClassifierSystem missing %q\n---\n%s", sub, s)
		}
	}
}

func TestClassifierSystem_workspaceSelectAddendum(t *testing.T) {
	s := ClassifierSystem(flows.WorkspaceSelect)
	if !strings.Contains(s, "WORKSPACE SELECT flow") {
		t.Error("expected workspace select intro from spec")
	}
	if !strings.Contains(s, "hasNonemptySelection") {
		t.Error("expected workspace-select selection tie-break")
	}
}

func TestClassifierSystem_selectFileAddendum(t *testing.T) {
	s := ClassifierSystem(flows.SelectFile)
	if !strings.Contains(s, "SELECT FILE flow") {
		t.Error("expected select file intro from spec")
	}
	if !strings.Contains(s, "create_entry") || !strings.Contains(s, "search_query") {
		t.Error("expected select-file routes to include create_entry and Rules search_query")
	}
}

func TestClassifierResponseJSONSchema_routeEnumMatchesSpec(t *testing.T) {
	for _, fid := range []flows.ID{flows.Root, flows.WorkspaceSelect, flows.SelectFile} {
		schema := ClassifierResponseJSONSchema(fid)
		props, _ := schema["properties"].(map[string]any)
		routeProp, _ := props["route"].(map[string]any)
		enumRaw, _ := routeProp["enum"].([]string)
		want := flows.SpecFor(fid).RouteIDs()
		if len(enumRaw) != len(want) {
			t.Fatalf("flow %s: enum len %d, spec len %d", fid, len(enumRaw), len(want))
		}
		if !slices.Equal(enumRaw, want) {
			t.Fatalf("flow %s: enum mismatch\n got %v\nwant %v", fid, enumRaw, want)
		}
	}
}

func TestClassifierResponseJSONSchema_searchQueryDescriptionPerFlow(t *testing.T) {
	rootDesc := ClassifierResponseJSONSchema(flows.Root)["properties"].(map[string]any)["search_query"].(map[string]any)["description"].(string)
	if !strings.Contains(rootDesc, "question") {
		t.Errorf("root search_query description should mention question route: %q", rootDesc)
	}
	wsDesc := ClassifierResponseJSONSchema(flows.WorkspaceSelect)["properties"].(map[string]any)["search_query"].(map[string]any)["description"].(string)
	if strings.Contains(wsDesc, "question") {
		t.Errorf("workspace flow schema should not mention question: %q", wsDesc)
	}
}
