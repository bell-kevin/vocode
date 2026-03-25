package transcript

import (
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// TranscriptService owns daemon-side semantics for handling voice transcripts.
// Today it only validates non-empty transcripts and returns an explicit acceptance result.
type TranscriptService struct{}

func NewService() *TranscriptService {
	return &TranscriptService{}
}

func (s *TranscriptService) AcceptTranscript(
	params protocol.VoiceTranscriptParams,
) (protocol.VoiceTranscriptResult, bool) {
	if strings.TrimSpace(params.Text) == "" {
		return protocol.VoiceTranscriptResult{}, false
	}

	// For now the daemon accepts non-empty transcripts unconditionally.
	return protocol.VoiceTranscriptResult{Accepted: true}, true
}
