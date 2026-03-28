package app

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"vocoding.net/vocode/v2/apps/voice/internal/mic"
	"vocoding.net/vocode/v2/apps/voice/internal/stt"
)

const audioMeterEmitInterval = 40 * time.Millisecond

func normalizeMeterRMS(rms float64) float64 {
	if rms <= 0 {
		return 0
	}
	const ref = 12000.0
	x := rms / ref
	if x > 1 {
		return 1
	}
	return x
}

func (a *App) transcribeLoop(ctx context.Context, apiKey string, modelID string, rec *mic.Recorder) {
	defer func() {
		_ = rec.Stop()
	}()

	client, err := stt.NewElevenLabsStreamingClient(ctx, apiKey, modelID, 16000)
	if err != nil {
		_ = a.write(Event{Type: "error", Message: fmt.Sprintf("failed to start elevenlabs streaming stt: %v", err)})
		return
	}
	defer func() {
		_ = client.Close()
	}()

	bytesPerSecond := int64(16000 * 1 * 2) // 16kHz * mono * int16
	minChunkBytes := bytesPerSecond * int64(streamMinChunkMS()) / 1000
	if minChunkBytes <= 0 {
		minChunkBytes = 6400
	}
	maxChunkBytes := bytesPerSecond * int64(streamMaxChunkMS()) / 1000
	if maxChunkBytes < minChunkBytes {
		maxChunkBytes = minChunkBytes
	}

	buf := make([]byte, 32*1024)
	var chunk []byte
	contextWindow := newUtteranceWindow(4, 1200)
	vad := newLocalVAD(16000, int(minChunkBytes), int(maxChunkBytes), streamMaxUtteranceMS())
	var lastAudioMeterEmit time.Time

	// ElevenLabs realtime STT emits partial_transcript until it receives input_audio_chunk with
	// commit: true, then it may emit committed_transcript for that segment (see stt.SendInputAudioChunk).
	// Docs call that a "manual commit"; here we drive it from local VAD: commit=true at end of
	// utterance (silence), periodic long-utterance cuts, and mic flush — not a separate UI action.

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, readErr := rec.PCMReader().Read(buf)
		if n > 0 {
			chunk = append(chunk, buf[:n]...)
		}

		for len(chunk) >= vad.frameBytes {
			frame := chunk[:vad.frameBytes]
			chunk = chunk[vad.frameBytes:]
			for _, c := range vad.process(frame) {
				if err := client.SendInputAudioChunk(c.pcm, c.commit, contextWindow.PreviousText()); err != nil {
					_ = a.write(Event{Type: "error", Message: fmt.Sprintf("elevenlabs streaming send failed: %v", err)})
					return
				}
			}
			if time.Since(lastAudioMeterEmit) >= audioMeterEmitInterval {
				lastAudioMeterEmit = time.Now()
				speaking, raw := vad.MeterSnapshot()
				norm := normalizeMeterRMS(raw)
				sp := speaking
				nr := norm
				_ = a.write(Event{Type: "audio_meter", Speaking: &sp, Rms: &nr})
			}
		}

		select {
		case evt, ok := <-client.Events():
			if !ok {
				return
			}
			if evt.Error != nil {
				_ = a.write(Event{Type: "error", Message: fmt.Sprintf("elevenlabs streaming stt failed: %v", evt.Error)})
				return
			}
			if strings.TrimSpace(evt.Text) != "" {
				committed := evt.IsFinal
				_ = a.write(Event{Type: "transcript", Text: evt.Text, Committed: &committed})
				if evt.IsFinal {
					contextWindow.AddUtterance(evt.Text)
				}
			}
		default:
		}

		if readErr != nil {
			if readErr == io.EOF {
				for _, c := range vad.flush() {
					if err := client.SendInputAudioChunk(c.pcm, c.commit, contextWindow.PreviousText()); err != nil {
						_ = a.write(Event{Type: "error", Message: fmt.Sprintf("elevenlabs streaming send failed: %v", err)})
						return
					}
				}
				return
			}
			_ = a.write(Event{Type: "error", Message: fmt.Sprintf("microphone read failed: %v", readErr)})
			return
		}
	}
}
