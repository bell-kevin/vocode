package actionplan

import (
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// CommandIntent is the daemon-plannable, protocol-shaped payload for a
// `run_command` step.
//
// The dispatcher maps this intent into protocol.CommandRunParams for the
// extension to execute, and uses commandexec.Policy to validate command
// shape/allowlist membership.
type CommandIntent struct {
	Command   string  `json:"command"`
	Args      []string `json:"args,omitempty"`
	TimeoutMs *int64   `json:"timeoutMs,omitempty"`
}

// CommandParams maps to protocol.CommandRunParams for command execution.
func (i CommandIntent) CommandParams() protocol.CommandRunParams {
	return protocol.CommandRunParams{
		Command:   strings.TrimSpace(i.Command),
		Args:      i.Args,
		TimeoutMs: i.TimeoutMs,
	}
}

