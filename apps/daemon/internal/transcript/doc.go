// Package transcript wires voice.transcript on the daemon: RPC service, optional coalescing queue,
// session / directive-batch correlation, and delegation to [executor.Executor] plus [run.Execute]
// for each utterance.
//
// Subpackages:
//   - [vocoding.net/vocode/v2/apps/daemon/internal/transcript/run] — single-utterance execute path (flows, host apply)
//   - [vocoding.net/vocode/v2/apps/daemon/internal/transcript/executor] — classifier / scope / scoped edit → directives
//   - [vocoding.net/vocode/v2/apps/daemon/internal/transcript/voicesession] — load/save voice session + apply report intake
//   - [vocoding.net/vocode/v2/apps/daemon/internal/transcript/config] — env for transcript path
//
// Root package files: service.go, service_worker.go, e2e tests, and small test helpers.
package transcript
