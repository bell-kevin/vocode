package stt

// https://elevenlabs.io/docs/api-reference/speech-to-text/v-1-speech-to-text-realtime

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	// messageTypeInputAudioChunk is the client->server audio publish event.
	messageTypeInputAudioChunk = "input_audio_chunk"
)

const (
	// messageTypeSessionStarted indicates the websocket session is ready.
	messageTypeSessionStarted = "session_started"
	// messageTypePartialTranscript is an uncommitted hypothesis.
	messageTypePartialTranscript = "partial_transcript"
	// messageTypeCommittedTranscript is a committed text segment.
	messageTypeCommittedTranscript = "committed_transcript"
	// messageTypeCommittedTranscriptWithTS is committed text with word timestamps.
	messageTypeCommittedTranscriptWithTS = "committed_transcript_with_timestamps"
)

const (
	// Error payload message types documented by ElevenLabs Scribe websocket API.
	messageTypeError                     = "error"
	messageTypeAuthError                 = "auth_error"
	messageTypeQuotaExceeded             = "quota_exceeded"
	messageTypeCommitThrottled           = "commit_throttled"
	messageTypeUnacceptedTerms           = "unaccepted_terms"
	messageTypeRateLimited               = "rate_limited"
	messageTypeQueueOverflow             = "queue_overflow"
	messageTypeResourceExhausted         = "resource_exhausted"
	messageTypeSessionTimeLimitExceeded  = "session_time_limit_exceeded"
	messageTypeInputError                = "input_error"
	messageTypeChunkSizeExceeded         = "chunk_size_exceeded"
	messageTypeInsufficientAudioActivity = "insufficient_audio_activity"
	messageTypeTranscriberError          = "transcriber_error"
)

type inputAudioChunkMessage struct {
	MessageType  string `json:"message_type"`
	AudioBase64  string `json:"audio_base_64"`
	Commit       bool   `json:"commit"`
	SampleRate   int    `json:"sample_rate"`
	// API: only allowed on the first non-empty audio chunk of the WebSocket session (input_error otherwise).
	PreviousText string `json:"previous_text,omitempty"`
}

type inboundEnvelope struct {
	MessageType string `json:"message_type"`
	// Some payloads use "type" instead of "message_type".
	Type       string `json:"type"`
	Text       string `json:"text"`
	Transcript string `json:"transcript"`
	Error      string `json:"error"`
	IsFinal    *bool  `json:"is_final"`
	Final      *bool  `json:"final"`
}

func primaryText(msg inboundEnvelope) string {
	t := strings.TrimSpace(msg.Text)
	if t != "" {
		return t
	}
	return strings.TrimSpace(msg.Transcript)
}

func parseStreamingEventPayload(data []byte) StreamingEvent {
	var msg inboundEnvelope
	if err := json.Unmarshal(data, &msg); err != nil {
		return StreamingEvent{}
	}

	msgType := strings.ToLower(strings.TrimSpace(msg.MessageType))
	if msgType == "" {
		msgType = strings.ToLower(strings.TrimSpace(msg.Type))
	}
	switch msgType {
	case messageTypePartialTranscript:
		return StreamingEvent{Text: primaryText(msg), IsFinal: false}
	case messageTypeCommittedTranscript, messageTypeCommittedTranscriptWithTS:
		return StreamingEvent{Text: primaryText(msg), IsFinal: true}
	case messageTypeSessionStarted:
		return StreamingEvent{SessionStarted: true}
	case messageTypeError, messageTypeAuthError, messageTypeQuotaExceeded, messageTypeCommitThrottled,
		messageTypeUnacceptedTerms, messageTypeRateLimited, messageTypeQueueOverflow,
		messageTypeResourceExhausted, messageTypeSessionTimeLimitExceeded, messageTypeInputError,
		messageTypeChunkSizeExceeded, messageTypeInsufficientAudioActivity, messageTypeTranscriberError:
		errText := strings.TrimSpace(msg.Error)
		if errText == "" {
			errText = msgType
		}
		return StreamingEvent{Error: fmt.Errorf("elevenlabs stream %s: %s", msgType, errText)}
	default:
		// Compatibility fallback for undocumented/legacy payloads.
		if t := primaryText(msg); t != "" {
			out := StreamingEvent{Text: t}
			if msg.IsFinal != nil {
				out.IsFinal = *msg.IsFinal
			} else if msg.Final != nil {
				out.IsFinal = *msg.Final
			}
			return out
		}
		return StreamingEvent{}
	}
}
