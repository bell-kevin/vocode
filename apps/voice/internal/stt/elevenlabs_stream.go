package stt

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

const elevenLabsSpeechToTextStreamURL = "wss://api.elevenlabs.io/v1/speech-to-text/stream"

type StreamingEvent struct {
	Text    string
	IsFinal bool
	Error   error
}

type ElevenLabsStreamingClient struct {
	conn         *websocket.Conn
	events       chan StreamingEvent
	done         chan struct{}
	mu           sync.Mutex
	sampleRate   int
	sentAnyChunk bool
}

func NewElevenLabsStreamingClient(ctx context.Context, apiKey string, modelID string, sampleRate int) (*ElevenLabsStreamingClient, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("ELEVENLABS_API_KEY is empty")
	}
	if strings.TrimSpace(modelID) == "" {
		modelID = "scribe_v2"
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
	wsURL.RawQuery = query.Encode()

	header := http.Header{}
	header.Set("xi-api-key", apiKey)

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL.String(), header)
	if err != nil {
		return nil, err
	}

	c := &ElevenLabsStreamingClient{
		conn:       conn,
		events:     make(chan StreamingEvent, 16),
		done:       make(chan struct{}),
		sampleRate: sampleRate,
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

	err := c.conn.Close()
	return err
}

// SendInputAudioChunk sends PCM audio; commit asks ElevenLabs to finalize that segment (leading to
// committed_transcript on the websocket). The app sets commit from local VAD boundaries.
func (c *ElevenLabsStreamingClient) SendInputAudioChunk(pcm []byte, commit bool, previousText string) error {
	if len(pcm) == 0 {
		return nil
	}

	payload := inputAudioChunkMessage{
		MessageType: messageTypeInputAudioChunk,
		AudioBase64: base64.StdEncoding.EncodeToString(pcm),
		Commit:      commit,
		SampleRate:  c.sampleRate,
	}
	if !c.sentAnyChunk && strings.TrimSpace(previousText) != "" {
		payload.PreviousText = previousText
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.conn.WriteJSON(payload); err != nil {
		return err
	}
	c.sentAnyChunk = true
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
			select {
			case c.events <- StreamingEvent{Error: err}:
			default:
			}
			return
		}

		evt := parseStreamingEventPayload(data)
		if evt.Error != nil {
			select {
			case c.events <- evt:
			default:
			}
			continue
		}
		if strings.TrimSpace(evt.Text) == "" {
			continue
		}
		select {
		case c.events <- evt:
		default:
		}
	}
}
