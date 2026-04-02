package transcript

import (
	protocol "vocoding.net/vocode/v2/packages/protocol/go"

	"vocoding.net/vocode/v2/apps/core/internal/flows/router"
	transcriptservice "vocoding.net/vocode/v2/apps/core/internal/transcript/service"
)

// Service adapts `internal/transcript/service` to the JSON-RPC handler interface.
type Service struct {
	inner *transcriptservice.Service
}

func NewService(flowRouter *router.FlowRouter) *Service {
	return &Service{inner: transcriptservice.NewService(flowRouter)}
}

func (s *Service) AcceptTranscript(params protocol.VoiceTranscriptParams) (protocol.VoiceTranscriptCompletion, bool, string) {
	if s == nil || s.inner == nil {
		// Fail-safe: accept but return a non-breaking completion.
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           "core daemon not initialized yet",
			TranscriptOutcome: "completed",
			UiDisposition:     "hidden",
		}, true, ""
	}
	return s.inner.AcceptTranscript(params)
}

// SetHostApplyClient wires the extension callback for `host.applyDirectives`.
func (s *Service) SetHostApplyClient(
	client interface {
		ApplyDirectives(protocol.HostApplyParams) (protocol.HostApplyResult, error)
	},
) {
	if s == nil || s.inner == nil {
		return
	}
	s.inner.SetHostApplyClient(client)
}
