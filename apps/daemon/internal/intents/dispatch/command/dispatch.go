package command

import (
	"fmt"

	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// defaultPolicy is process-wide command allowlist validation (extension performs execution).
var defaultPolicy = NewPolicy()

// DispatchCommand validates command run parameters against daemon-side policy and returns
// the wire directive for the extension.
func DispatchCommand(cmd intents.CommandIntent) (protocol.CommandDirective, error) {
	params := protocol.NewCommandDirective(cmd.Command, cmd.Args, cmd.TimeoutMs)
	if err := defaultPolicy.Validate(params); err != nil {
		return protocol.CommandDirective{}, fmt.Errorf("%s", err.Error())
	}
	return params, nil
}
