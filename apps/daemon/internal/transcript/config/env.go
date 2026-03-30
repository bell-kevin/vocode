package config

import "time"

// This package previously read daemon transcript tuning from environment variables.
// Tuning is now provided via `voice.transcript` params (`daemonConfig`) so it can
// change without restarting the daemon process.

const defaultSessionIdleReset = 30 * time.Minute

func DefaultSessionIdleReset() time.Duration {
	return defaultSessionIdleReset
}

const (
	DefaultGatheredMaxBytes    = 120_000
	DefaultGatheredMaxExcerpts = 12
	DefaultMaxRepairSteps      = 8

	DefaultTranscriptQueueSize   = 10
	DefaultTranscriptCoalesceMs  = 750
	DefaultTranscriptMaxMergeJobs  = 5
	DefaultTranscriptMaxMergeChars = 6000

	DefaultMaxAgentTurns              = 8
	DefaultMaxIntentRetries           = 2
	DefaultMaxContextRounds           = 2
	DefaultMaxContextBytes            = 12000
	DefaultMaxConsecutiveContextReq   = 3
	DefaultMaxIntentsPerBatch         = 16
)
