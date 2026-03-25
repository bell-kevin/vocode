package app

import (
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type VoiceTranscriptService struct{}

func NewVoiceTranscriptService() *VoiceTranscriptService {
	return &VoiceTranscriptService{}
}

func (s *VoiceTranscriptService) AcceptTranscript(
	params protocol.VoiceTranscriptParams,
) (protocol.VoiceTranscriptResult, bool) {
	if strings.TrimSpace(params.Text) == "" {
		return protocol.VoiceTranscriptResult{}, false
	}

	// For now the daemon accepts non-empty transcripts unconditionally.
	return protocol.VoiceTranscriptResult{Accepted: true}, true
}

