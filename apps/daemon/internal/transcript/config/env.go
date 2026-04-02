package config

import "time"

// Tuning is provided via `voice.transcript` params (`daemonConfig`) so it can
// change without restarting the daemon process.

const defaultSessionIdleReset = 30 * time.Minute

func DefaultSessionIdleReset() time.Duration {
	return defaultSessionIdleReset
}

const (
	DefaultGatheredMaxBytes    = 120_000
	DefaultGatheredMaxExcerpts = 12

	DefaultTranscriptQueueSize     = 10
	DefaultTranscriptCoalesceMs    = 750
	DefaultTranscriptMaxMergeJobs  = 5
	DefaultTranscriptMaxMergeChars = 6000
)
