package commandexec

import (
	"fmt"
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Service runs shell commands under daemon-side policy.
type Service struct {
	policy *Policy
	runner *Runner
}

func NewService() *Service {
	return &Service{
		policy: NewPolicy(),
		runner: NewRunner(),
	}
}

func (s *Service) Run(params protocol.CommandRunParams) protocol.CommandRunResult {
	// Normalize command input.
	params.Command = strings.TrimSpace(params.Command)
	if len(params.Args) == 0 {
		// Keep args as nil/empty; downstream exec.Command handles both.
	}

	failure, ok := s.policy.Validate(params)
	if !ok {
		return protocol.CommandRunResult{
			Kind:    "failure",
			Failure: &failure,
			Stdout:  "",
			Stderr:  "",
		}
	}

	out, err := s.runner.Run(params)
	if err != nil {
		return protocol.CommandRunResult{
			Kind: "failure",
			Failure: &protocol.CommandFailure{
				Code:    "execution_failed",
				Message: fmt.Sprintf("command failed to execute: %v", err),
			},
			Stdout: "",
			Stderr: "",
		}
	}

	if out.timeout {
		return protocol.CommandRunResult{
			Kind: "failure",
			Failure: &protocol.CommandFailure{
				Code:    "timeout",
				Message: "command timed out",
			},
			Stdout: out.stdout,
			Stderr: out.stderr,
		}
	}

	// Even if the exit code is non-zero, we treat the command as successfully
	// executed at the transport layer.
	exitCode := out.exitCode
	return protocol.CommandRunResult{
		Kind:     "success",
		ExitCode: &exitCode,
		Stdout:   out.stdout,
		Stderr:   out.stderr,
	}
}
