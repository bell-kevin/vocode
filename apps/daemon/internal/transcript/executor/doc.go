// Package executor runs one voice.transcript agent loop: iterative [agent.Agent.NextTurn], context rounds,
// batched intents per turn, retries, and dispatch into intents/dispatch.
//
// Files: executor.go (entry), agent_loop_state.go, execute_iteration.go, apply_outcome.go, execute_finalize.go, helpers.go.
package executor
