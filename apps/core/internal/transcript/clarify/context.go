package clarify

// ClarificationContext is structured clarification information threaded into the
// agent/executor so resume behavior does not rely on string rewriting heuristics.
type ClarificationContext struct {
	OriginalTranscript string
	ClarifyQuestion    string
	AnswerText         string

	// ClarifyTargetResolution is a protocol string such as "workspace_select", "edit", "select_file", etc.
	ClarifyTargetResolution string
}
