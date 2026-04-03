package router

import (
	"testing"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
)

func TestResultValidate(t *testing.T) {
	t.Parallel()
	if err := (Result{Flow: flows.Root, Route: "select", SearchQuery: "foo"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Result{Flow: flows.Root, Route: "select"}).Validate(); err == nil {
		t.Fatal("expected error: select requires search_query")
	}
	if err := (Result{Flow: flows.Root, Route: "bogus"}).Validate(); err == nil {
		t.Fatal("expected error")
	}
	if err := (Result{Flow: flows.Select, Route: "select_control"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Result{Flow: flows.SelectFile, Route: "open"}).Validate(); err != nil {
		t.Fatal(err)
	}
}
