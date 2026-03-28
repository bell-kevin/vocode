package app

import (
	"context"
	"fmt"
	"io"
	"os"
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
	// Heuristic scale for smoothed PCM16 frame RMS; lower ref = fuller bar at normal mic levels.
	const ref = 2200.0
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

	client, err := stt.NewElevenLabsStreamingClient(ctx, apiKey, modelID, 16000, sttLanguageCode())
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

	if vadDebugEnabled() {
		fmt.Fprintf(
			os.Stderr,
			"[vocode-vad] debug on (VOCODE_VOICE_VAD_DEBUG=%q)\n",
			strings.TrimSpace(os.Getenv("VOCODE_VOICE_VAD_DEBUG")),
		)
		stderrSync()
	}

	// ElevenLabs realtime STT emits partial_transcript until it receives input_audio_chunk with
	// commit: true, then it may emit committed_transcript for that segment (see sendSTTChunk).
	// Docs call that a "manual commit"; here we drive it from local VAD: commit=true at end of
	// utterance (silence), periodic long-utterance cuts, and mic flush — not a separate UI action.

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Drain STT events before blocking on the mic. Otherwise a closed WebSocket can sit in
		// client.Events() for seconds while Read() waits for audio, leaving the UI stuck on a partial.
		if a.drainSTTClientEvents(client, contextWindow) {
			return
		}

		n, readErr := rec.PCMReader().Read(buf)
		if n > 0 {
			chunk = append(chunk, buf[:n]...)
		}

		for len(chunk) >= vad.frameBytes {
			frame := chunk[:vad.frameBytes]
			chunk = chunk[vad.frameBytes:]
			for _, c := range vad.process(frame) {
				if vadDebugEnabled() && c.commit {
					fmt.Fprintf(os.Stderr, "[vocode-vad] sending commit pcm_bytes=%d\n", len(c.pcm))
					stderrSync()
				}
				if err := a.sendSTTChunk(client, contextWindow, c.pcm, c.commit); err != nil {
					_ = a.write(Event{Type: "error", Message: fmt.Sprintf("elevenlabs streaming send failed: %v", err)})
					return
				}
			}
		}

		if a.drainSTTClientEvents(client, contextWindow) {
			return
		}

		// One meter tick per outer iteration so the UI updates even when the mic buffer is < 1 VAD frame.
		if time.Since(lastAudioMeterEmit) >= audioMeterEmitInterval {
			lastAudioMeterEmit = time.Now()
			speaking, raw := vad.MeterSnapshot()
			norm := normalizeMeterRMS(raw)
			sp := speaking
			nr := norm
			_ = a.write(Event{Type: "audio_meter", Speaking: &sp, Rms: &nr})
		}

		if readErr != nil {
			if readErr == io.EOF {
				for _, c := range vad.flush() {
					if vadDebugEnabled() && c.commit {
						fmt.Fprintf(os.Stderr, "[vocode-vad] sending commit pcm_bytes=%d (flush)\n", len(c.pcm))
						stderrSync()
					}
					if err := a.sendSTTChunk(client, contextWindow, c.pcm, c.commit); err != nil {
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

// sendSTTChunk forwards VAD audio to ElevenLabs. For commits with PCM, we send audio with
// commit=false and then an empty commit=true frame — the official examples use this pattern and it
// avoids the model truncating the tail of the segment when commit shared the last audio packet.
func (a *App) sendSTTChunk(client *stt.ElevenLabsStreamingClient, contextWindow *utteranceWindow, pcm []byte, commit bool) error {
	if !commit {
		return client.SendInputAudioChunk(pcm, false, contextWindow.PreviousText())
	}
	if len(pcm) == 0 {
		return client.SendInputAudioChunk(nil, true, "")
	}
	if err := client.SendInputAudioChunk(pcm, false, contextWindow.PreviousText()); err != nil {
		return err
	}
	return client.SendInputAudioChunk(nil, true, "")
}

// drainSTTClientEvents pulls every event currently queued on the ElevenLabs client (non-blocking).
// Returns true if the transcribe loop should exit (error, channel closed, or write failure).
func (a *App) drainSTTClientEvents(client *stt.ElevenLabsStreamingClient, contextWindow *utteranceWindow) (stop bool) {
	for {
		select {
		case evt, ok := <-client.Events():
			if !ok {
				return true
			}
			if evt.Error != nil {
				_ = a.write(Event{Type: "error", Message: fmt.Sprintf("elevenlabs streaming stt failed: %v", evt.Error)})
				return true
			}
			if strings.TrimSpace(evt.Text) != "" {
				if err := a.writeTranscript(evt.Text, evt.IsFinal); err != nil {
					return true
				}
				if evt.IsFinal {
					contextWindow.AddUtterance(evt.Text)
				}
			}
		default:
			return false
		}
	}
}
