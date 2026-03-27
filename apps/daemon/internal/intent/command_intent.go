package intent

import (
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type CommandIntent struct {
	Command   string  `json:"command"`
	Args      []string `json:"args,omitempty"`
	TimeoutMs *int64   `json:"timeoutMs,omitempty"`
}

func (i CommandIntent) CommandParams() protocol.CommandRunParams {
	return protocol.CommandRunParams{
		Command:   strings.TrimSpace(i.Command),
		Args:      i.Args,
		TimeoutMs: i.TimeoutMs,
	}
}
