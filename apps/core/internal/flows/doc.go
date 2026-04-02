// Package flows defines voice/transcript flows: flow IDs, composed route specs (global routes
// plus flow-specific routes), validation, and small flow helpers (global exit, selection list nav).
//
// Subpackage router classifies transcripts to a route id; per-route handlers interpret the transcript.
package flows
