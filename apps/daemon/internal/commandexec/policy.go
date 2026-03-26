package commandexec

import (
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Policy owns the daemon-side safety rules for command execution.
//
// This is intentionally conservative for now: it only allows execution of a
// small set of known shell entry points, and it requires the command name to
// match exactly (case-insensitive).
type Policy struct {
	allowed map[string]struct{}
}

func NewPolicy() *Policy {
	return &Policy{
		allowed: map[string]struct{}{
			"cmd.exe":        {},
			"powershell.exe": {},
			"powershell":     {},
			"pwsh": {},
			// Unix stub smoke test; on Windows the stub uses cmd.exe instead.
			"echo": {},
		},
	}
}

func (p *Policy) Validate(params protocol.CommandRunParams) (protocol.CommandFailure, bool) {
	cmd := strings.TrimSpace(params.Command)
	if cmd == "" {
		return protocol.CommandFailure{
			Code:    "command_rejected",
			Message: "command cannot be empty",
		}, false
	}

	// This API is structured as (command + args), so reject anything that
	// looks like it already contains spacing. (Shell parsing belongs in the
	// target shell, not in the daemon policy layer.)
	if strings.ContainsAny(cmd, " \t\r\n") {
		return protocol.CommandFailure{
			Code:    "command_rejected",
			Message: "command must be a single executable name",
		}, false
	}

	normalized := strings.ToLower(cmd)
	if _, ok := p.allowed[normalized]; !ok {
		return protocol.CommandFailure{
			Code:    "command_rejected",
			Message: "command is not allowed",
		}, false
	}

	for _, arg := range params.Args {
		// Null bytes can corrupt exec boundaries.
		if strings.ContainsRune(arg, '\x00') {
			return protocol.CommandFailure{
				Code:    "command_rejected",
				Message: "command args contain invalid characters",
			}, false
		}
	}

	return protocol.CommandFailure{}, true
}
