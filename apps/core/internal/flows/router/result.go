package router

import (
	"fmt"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
)

// Result is the classifier output for the given flow: route id plus optional structured fields.
type Result struct {
	Flow  flows.ID
	Route string
	// SearchQuery is the literal string to pass to ripgrep for routes "workspace_select" (content) and
	// "select_file" (path/name fragment). Populated by the routing model (or stub); must be non-empty when those routes are chosen.
	SearchQuery string
}

func (r Result) Validate() error {
	if r.Flow != flows.Root && r.Flow != flows.WorkspaceSelect && r.Flow != flows.SelectFile {
		return fmt.Errorf("flow router: unknown flow %q", r.Flow)
	}
	if err := flows.ValidateRoute(r.Flow, r.Route); err != nil {
		return err
	}
	switch r.Route {
	case "workspace_select", "select_file":
		if strings.TrimSpace(r.SearchQuery) == "" {
			return fmt.Errorf("flow router: route %q requires non-empty search_query", r.Route)
		}
	}
	return nil
}
