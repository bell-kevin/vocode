// Package run implements one voice.transcript utterance: load session, handle control / clarify /
// selection / file-selection preludes, invoke the executor, optional host.applyDirectives, persist.
//
// Boundaries: depends on agentcontext, sibling packages config / executor / voicesession, and
// protocol types. This package must not import the parent transcript package (no cycles); the
// service constructs run.Env and calls [Execute].
package run
