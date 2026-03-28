package command

import (
	"fmt"

	"vocoding.net/vocode/v2/apps/daemon/internal/intent"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Service validates command run parameters against daemon-side policy.
//
// The extension is responsible for performing actual execution.
type Service struct {
	policy *Policy
}

func NewService() *Service {
	return &Service{policy: NewPolicy()}
}

func (s *Service) Validate(params protocol.CommandDirective) error {
	if err := s.policy.Validate(params); err != nil {
		return fmt.Errorf("%s", err.Error())
	}
	return nil
}

func (s *Service) DispatchIntent(cmd intent.CommandIntent) (protocol.CommandDirective, error) {
	params := protocol.NewCommandDirective(cmd.Command, cmd.Args, cmd.TimeoutMs)
	if err := s.Validate(params); err != nil {
		return protocol.CommandDirective{}, err
	}
	return params, nil
}
