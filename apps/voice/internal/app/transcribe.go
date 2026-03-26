package app

import (
	"context"
	"fmt"
	"io"
	"strings"

	"vocoding.net/vocode/v2/apps/voice/internal/mic"
	"vocoding.net/vocode/v2/apps/voice/internal/stt"
)

func (a *App) transcribeLoop(ctx context.Context, apiKey string, rec *mic.Recorder) {
	defer func() {
		_ = rec.Stop()
	}()

	if sttMode() == "stream" {
		a.transcribeLoopStream(ctx, apiKey, rec)
		return
	}
	a.transcribeLoopBatch(ctx, apiKey, rec)
}

func (a *App) transcribeLoopBatch(ctx context.Context, apiKey string, rec *mic.Recorder) {
	bytesPerSecond := int64(16000 * 1 * 2) // 16kHz * mono * int16
	targetBytes := bytesPerSecond * a.segmentSeconds
	if targetBytes <= 0 {
		targetBytes = bytesPerSecond * 5
	}

	buf := make([]byte, 32*1024)
	var segment []byte
	previousText := ""

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := rec.PCMReader().Read(buf)
		if n > 0 {
			segment = append(segment, buf[:n]...)
		}

		if int64(len(segment)) >= targetBytes {
			wav, werr := mic.EncodeWavPCM16LE(segment, 16000, 1)
			if werr != nil {
				_ = a.write(Event{Type: "error", Message: fmt.Sprintf("failed to encode wav: %v", werr)})
			} else {
				text, terr := stt.TranscribeElevenLabs(apiKey, "audio/wav", wav, previousText)
				if terr != nil {
					_ = a.write(Event{Type: "error", Message: fmt.Sprintf("elevenlabs stt failed: %v", terr)})
				} else if strings.TrimSpace(text) != "" {
					_ = a.write(Event{Type: "transcript", Text: text})
					previousText = appendRollingContext(previousText, text, 500)
				}
			}
			segment = nil
		}

		if err != nil {
			if err == io.EOF {
				return
			}
			_ = a.write(Event{Type: "error", Message: fmt.Sprintf("microphone read failed: %v", err)})
			return
		}
	}
}

func (a *App) transcribeLoopStream(ctx context.Context, apiKey string, rec *mic.Recorder) {
	client, err := stt.NewElevenLabsStreamingClient(ctx, apiKey, 16000)
	if err != nil {
		_ = a.write(Event{Type: "error", Message: fmt.Sprintf("failed to start elevenlabs streaming stt: %v", err)})
		return
	}
	defer func() {
		_ = client.Close()
	}()

	bytesPerSecond := int64(16000 * 1 * 2) // 16kHz * mono * int16
	chunkBytes := bytesPerSecond * 300 / 1000
	if chunkBytes <= 0 {
		chunkBytes = 9600
	}

	buf := make([]byte, 32*1024)
	var chunk []byte
	previousText := ""
	useVAD := vadEnabled()
	var vad *localVAD
	if useVAD {
		vad = newLocalVAD(16000, int(chunkBytes))
	}

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

		if useVAD {
			for len(chunk) >= vad.frameBytes {
				frame := chunk[:vad.frameBytes]
				chunk = chunk[vad.frameBytes:]
				for _, c := range vad.process(frame) {
					if err := client.SendInputAudioChunk(c.pcm, c.commit, previousText); err != nil {
						_ = a.write(Event{Type: "error", Message: fmt.Sprintf("elevenlabs streaming send failed: %v", err)})
						return
					}
				}
			}
		} else {
			for int64(len(chunk)) >= chunkBytes {
				toSend := chunk[:chunkBytes]
				chunk = chunk[chunkBytes:]
				if err := client.SendInputAudioChunk(toSend, true, previousText); err != nil {
					_ = a.write(Event{Type: "error", Message: fmt.Sprintf("elevenlabs streaming send failed: %v", err)})
					return
				}
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
				_ = a.write(Event{Type: "transcript", Text: evt.Text})
				previousText = appendRollingContext(previousText, evt.Text, 500)
			}
		default:
		}

		if readErr != nil {
			if readErr == io.EOF {
				if useVAD {
					for _, c := range vad.flush() {
						if err := client.SendInputAudioChunk(c.pcm, c.commit, previousText); err != nil {
							_ = a.write(Event{Type: "error", Message: fmt.Sprintf("elevenlabs streaming send failed: %v", err)})
							return
						}
					}
				}
				return
			}
			_ = a.write(Event{Type: "error", Message: fmt.Sprintf("microphone read failed: %v", readErr)})
			return
		}
	}
}

func appendRollingContext(existing string, next string, maxChars int) string {
	next = strings.TrimSpace(next)
	if next == "" {
		return existing
	}
	if maxChars <= 0 {
		maxChars = 500
	}

	combined := next
	if existing != "" {
		combined = existing + " " + next
	}
	if len(combined) <= maxChars {
		return combined
	}
	return strings.TrimSpace(combined[len(combined)-maxChars:])
}
