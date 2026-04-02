package router

import (
	"fmt"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
)

// Result is which route the transcript belongs to for the given flow.
type Result struct {
	Flow  flows.ID
	Route string
}

func (r Result) Validate() error {
	if r.Flow != flows.Root && r.Flow != flows.Select && r.Flow != flows.SelectFile {
		return fmt.Errorf("flow router: unknown flow %q", r.Flow)
	}
	return flows.ValidateRoute(r.Flow, r.Route)
}
