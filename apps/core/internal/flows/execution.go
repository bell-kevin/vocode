package flows

// Execution is host/runtime metadata for transcript handler ordering (immediate vs serialized queue).
// It MUST NOT be included in classifier system prompts, user JSON, or response JSON schema — see router/prompt.go.
type Execution int

const (
	// ExecutionImmediate handlers are deterministic or cheap; they may bypass the transcript job queue
	// while still holding the per-session execute mutex (see transcript.Service).
	ExecutionImmediate Execution = iota
	// ExecutionSerialized covers model calls, ripgrep, host apply, and other work that must not overlap
	// with other serialized handlers for the same session.
	ExecutionSerialized
)

// RouteExecution returns the execution policy for a classified route id within a flow.
// Unknown routes default to ExecutionSerialized (safest).
func RouteExecution(flow ID, route string) Execution {
	for _, r := range SpecFor(flow).Routes {
		if r.ID == route {
			return r.Execution
		}
	}
	return ExecutionSerialized
}
