package router

import (
	"testing"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
)

func TestResultValidate(t *testing.T) {
	t.Parallel()
	if err := (Result{Flow: flows.Root, Route: "workspace_select", SearchQuery: "foo"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Result{Flow: flows.Root, Route: "workspace_select"}).Validate(); err == nil {
		t.Fatal("expected error: workspace_select requires search_query")
	}
	if err := (Result{Flow: flows.Root, Route: "bogus"}).Validate(); err == nil {
		t.Fatal("expected error")
	}
	if err := (Result{Flow: flows.WorkspaceSelect, Route: "workspace_select_control"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Result{Flow: flows.SelectFile, Route: "rename"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Result{Flow: flows.WorkspaceSelect, Route: "rename"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Result{Flow: flows.WorkspaceSelect, Route: "create"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Result{Flow: flows.Root, Route: "create"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Result{Flow: flows.SelectFile, Route: "create"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Result{Flow: flows.SelectFile, Route: "create_entry"}).Validate(); err != nil {
		t.Fatal(err)
	}
}
