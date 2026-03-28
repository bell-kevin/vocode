package command

import (
	"fmt"
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Policy owns the daemon-side safety rules for command "execution requests".
//
// Even though the extension performs actual execution, we still validate
// command shape against an allowlist so the daemon never emits unsafe
// instructions.
type Policy struct {
	allowed map[string]struct{}
}

func NewPolicy() *Policy {
	return &Policy{
		allowed: map[string]struct{}{
			"cmd.exe":        {},
			"powershell.exe": {},
			"powershell":     {},
			"pwsh":           {},
			// Unix stub smoke test; on Windows the stub uses cmd.exe instead.
			"echo": {},
		},
	}
}

func (p *Policy) Validate(params protocol.CommandDirective) error {
	cmd := strings.TrimSpace(params.Command)
	if cmd == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// This API is structured as (command + args), so reject anything that
	// looks like it already contains spacing. (Shell parsing belongs in the
	// target shell, not in the daemon policy layer.)
	if strings.ContainsAny(cmd, " \t\r\n") {
		return fmt.Errorf("command must be a single executable name")
	}

	normalized := strings.ToLower(cmd)
	if _, ok := p.allowed[normalized]; !ok {
		return fmt.Errorf("command is not allowed")
	}

	for _, arg := range params.Args {
		// Null bytes can corrupt exec boundaries.
		if strings.ContainsRune(arg, '\x00') {
			return fmt.Errorf("command args contain invalid characters")
		}
	}

	return nil
}
