package run

import "vocoding.net/vocode/v2/apps/core/internal/flows"

// ExecuteOpts optionally supplies a route already computed by FlowRouter (classify-then-queue fast path).
type ExecuteOpts struct {
	HasPreclassified         bool
	PreclassifiedFlow        flows.ID
	PreclassifiedRoute       string
	PreclassifiedSearchQuery string
}
