package commandexec

import (
	"fmt"

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

func (s *Service) Validate(params protocol.CommandRunParams) error {
	failure, ok := s.policy.Validate(params)
	if ok {
		return nil
	}
	return fmt.Errorf("%s", failure.Message)
}
