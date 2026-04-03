package stt

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// Official WebSocket path per https://elevenlabs.io/docs/api-reference/speech-to-text/v-1-speech-to-text-realtime
// (The older "/speech-to-text/stream" path can return 403 on handshake.)
const elevenLabsSpeechToTextStreamURL = "wss://api.elevenlabs.io/v1/speech-to-text/realtime"

type StreamingEvent struct {
	Text    string
	IsFinal bool
	Error   error
	// SessionStarted is true for the server session_started handshake (no transcript text).
	SessionStarted bool
}

type ElevenLabsStreamingClient struct {
	conn         *websocket.Conn
	events       chan StreamingEvent
	done         chan struct{}
	sessionReady chan struct{}
	sessionOnce  sync.Once
	mu           sync.Mutex
	sampleRate   int
	// ElevenLabs returns input_error if previous_text appears after the first non-empty audio chunk
	// of the session (not per VAD segment).
	sentAudioChunk bool
	// Bytes sent with commit:false since the last successful commit (server-side uncommitted buffer).
	pendingUncommittedBytes int
}

func NewElevenLabsStreamingClient(
	ctx context.Context,
	apiKey string,
	modelID string,
	sampleRate int,
	languageCode string,
	inactivityTimeoutSeconds int,
) (*ElevenLabsStreamingClient, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("ELEVENLABS_API_KEY is empty")
	}
	if strings.TrimSpace(modelID) == "" {
		// Realtime WebSocket requires the realtime model id (see AsyncAPI + docs examples).
		// Using batch ids like "scribe_v2" yields close 1008 invalid_request.
		modelID = "scribe_v2_realtime"
	}
	if sampleRate <= 0 {
		sampleRate = 16000
	}

	wsURL, err := url.Parse(elevenLabsSpeechToTextStreamURL)
	if err != nil {
		return nil, err
	}
	query := wsURL.Query()
	query.Set("model_id", modelID)
	// Required / expected by realtime AsyncAPI so PCM16 @ sample_rate matches negotiated format.
	query.Set("audio_format", "pcm_16000")
	if lc := strings.TrimSpace(languageCode); lc != "" {
		query.Set("language_code", lc)
	}
	if inactivityTimeoutSeconds > 0 {
		if inactivityTimeoutSeconds > 180 {
			inactivityTimeoutSeconds = 180
		}
		query.Set("inactivity_timeout", strconv.Itoa(inactivityTimeoutSeconds))
	}
	for _, kt := range AllRealtimeSTTKeyterms() {
		kt = strings.TrimSpace(kt)
		if kt != "" {
			query.Add("keyterms", kt)
		}
	}
	// Do not set commit_strategy here: some API revisions reject the upgrade when unknown or
	// mismatched query values are present (websocket: bad handshake). Manual commit is the
	// documented default; we still send commit on input_audio_chunk from local VAD.
	wsURL.RawQuery = query.Encode()

	header := http.Header{}
	header.Set("xi-api-key", apiKey)

	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, wsURL.String(), header)
	if err != nil {
		if resp != nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			return nil, fmt.Errorf("%w (http %d: %s)", err, resp.StatusCode, strings.TrimSpace(string(body)))
		}
		return nil, err
	}
	if resp != nil {
		_ = resp.Body.Close()
	}

	c := &ElevenLabsStreamingClient{
		conn:         conn,
		events:       make(chan StreamingEvent, 256),
		done:         make(chan struct{}),
		sessionReady: make(chan struct{}),
		sampleRate:   sampleRate,
	}

	go c.readLoop()
	return c, nil
}

func (c *ElevenLabsStreamingClient) Events() <-chan StreamingEvent {
	return c.events
}

func (c *ElevenLabsStreamingClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case <-c.done:
		return nil
	default:
		close(c.done)
	}

	c.unblockSessionWait()
	err := c.conn.Close()
	return err
}

func (c *ElevenLabsStreamingClient) unblockSessionWait() {
	c.sessionOnce.Do(func() { close(c.sessionReady) })
}

// messageTypeIsSessionStarted handles minor API / JSON shape differences so we unblock audio
// only after the server session is ready.
func messageTypeIsSessionStarted(data []byte) bool {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return false
	}
	for _, key := range []string{"message_type", "type", "event", "event_type"} {
		if v, ok := m[key].(string); ok && strings.EqualFold(strings.TrimSpace(v), messageTypeSessionStarted) {
			return true
		}
	}
	return false
}

func formatWebSocketErr(err error) error {
	var ce *websocket.CloseError
	if errors.As(err, &ce) {
		detail := strings.TrimSpace(ce.Text)
		if ce.Code == websocket.CloseNormalClosure {
			if detail != "" {
				return fmt.Errorf("websocket closed: code=1000 (normal closure) reason=%q", detail)
			}
			return fmt.Errorf("websocket closed: code=1000 (normal closure; session ended by server)")
		}
		if detail == "" {
			return fmt.Errorf("websocket closed: code=%d (empty close reason from server)", ce.Code)
		}
		// Do not wrap the original err — its Error() repeats the same code/text as ce.Text.
		return fmt.Errorf("websocket closed: code=%d reason=%q", ce.Code, detail)
	}
	return err
}

// minUncommittedBytesBeforeCommit is how much commit:false audio ElevenLabs expects before commit:true
// (see commit_throttled: "at least 0.3s"); use a small margin above 300ms.
func (c *ElevenLabsStreamingClient) minUncommittedBytesBeforeCommit() int {
	const ms = 310
	return c.sampleRate * 2 * ms / 1000
}

// SendInputAudioChunk sends PCM audio; commit asks ElevenLabs to finalize that segment (leading to
// committed_transcript on the websocket). The app sets commit from local VAD boundaries.
func (c *ElevenLabsStreamingClient) SendInputAudioChunk(pcm []byte, commit bool, previousText string) error {
	// Docs allow commit-only frames: empty audio_base_64 + commit:true to finalize a segment.
	if len(pcm) == 0 && !commit {
		return nil
	}

	// Realtime API rejects early audio with 1008 invalid_request until session_started is received.
	<-c.sessionReady

	c.mu.Lock()
	defer c.mu.Unlock()

	// Finalize: pad trailing silence if the segment is shorter than the API minimum, then send
	// empty commit (matches sendSTTChunk: pcm commit false, then nil commit true).
	if commit && len(pcm) == 0 {
		minB := c.minUncommittedBytesBeforeCommit()
		// VAD sometimes emits commit after all PCM was already sent (drained tail). Server still
		// needs ≥ min duration before accept; pad silence so committed_transcript arrives instead of
		// commit_throttled or an open segment with stuck partials.
		if c.pendingUncommittedBytes == 0 {
			pad := make([]byte, minB)
			if err := c.writeInputAudioChunkLocked(pad, false, ""); err != nil {
				return err
			}
		}
		if c.pendingUncommittedBytes < minB {
			need := minB - c.pendingUncommittedBytes
			if need%2 != 0 {
				need++
			}
			pad := make([]byte, need)
			if err := c.writeInputAudioChunkLocked(pad, false, ""); err != nil {
				return err
			}
		}
		if err := c.writeInputAudioChunkLocked(nil, true, ""); err != nil {
			return err
		}
		return nil
	}

	return c.writeInputAudioChunkLocked(pcm, commit, previousText)
}

func (c *ElevenLabsStreamingClient) writeInputAudioChunkLocked(pcm []byte, commit bool, previousText string) error {
	payload := inputAudioChunkMessage{
		MessageType: messageTypeInputAudioChunk,
		AudioBase64: base64.StdEncoding.EncodeToString(pcm),
		Commit:      commit,
		SampleRate:  c.sampleRate,
	}
	if len(pcm) > 0 && !c.sentAudioChunk {
		if pt := strings.TrimSpace(previousText); pt != "" {
			payload.PreviousText = pt
		}
	}
	if err := c.conn.WriteJSON(payload); err != nil {
		return err
	}
	if commit {
		c.pendingUncommittedBytes = 0
		return nil
	}
	if len(pcm) > 0 {
		c.sentAudioChunk = true
		c.pendingUncommittedBytes += len(pcm)
	}
	return nil
}

func (c *ElevenLabsStreamingClient) readLoop() {
	defer close(c.events)

	for {
		select {
		case <-c.done:
			return
		default:
		}

		_, data, err := c.conn.ReadMessage()
		if err != nil {
			c.unblockSessionWait()
			c.events <- StreamingEvent{Error: formatWebSocketErr(err)}
			return
		}

		if messageTypeIsSessionStarted(data) {
			c.unblockSessionWait()
			continue
		}

		evt := parseStreamingEventPayload(data)
		if evt.Error != nil {
			c.unblockSessionWait()
			c.events <- evt
			continue
		}
		if evt.SessionStarted {
			c.unblockSessionWait()
			continue
		}
		if strings.TrimSpace(evt.Text) == "" {
			continue
		}
		// Never drop transcript events — a full channel would lose committed_transcript.
		c.events <- evt
	}
}
