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

func (a *App) transcribeLoop(ctx context.Context, apiKey string, rec *mic.Recorder) {
	defer func() {
		_ = rec.Stop()
	}()

	cfg := a.getSidecarConfig()

	client, err := stt.NewElevenLabsStreamingClient(ctx, apiKey, cfg.SttModelId, 16000, cfg.SttLanguageCode)
	if err != nil {
		_ = a.write(Event{Type: "error", Message: fmt.Sprintf("failed to start elevenlabs streaming stt: %v", err)})
		return
	}
	defer func() {
		_ = client.Close()
	}()

	bytesPerSecond := int64(16000 * 1 * 2) // 16kHz * mono * int16
	minChunkBytes := bytesPerSecond * int64(cfg.StreamMinChunkMs) / 1000
	if minChunkBytes <= 0 {
		minChunkBytes = 6400
	}
	maxChunkBytes := bytesPerSecond * int64(cfg.StreamMaxChunkMs) / 1000
	if maxChunkBytes < minChunkBytes {
		maxChunkBytes = minChunkBytes
	}

	buf := make([]byte, 32*1024)
	var chunk []byte
	contextWindow := newUtteranceWindow(4, 1200)
	vad := newLocalVAD(
		16000,
		int(minChunkBytes),
		int(maxChunkBytes),
		cfg.StreamMaxUtteranceMs,
		cfg.VadThresholdMultiplier,
		cfg.VadMinEnergyFloor,
		cfg.VadStartMs,
		cfg.VadEndMs,
		cfg.VadPrerollMs,
		cfg.VadDebugEnabled,
	)
	var lastAudioMeterEmit time.Time
	lastCfg := cfg

	// After commit:true, defer further non-commit PCM until committed_transcript (IsFinal) is
	// drained, so the server can emit committed_transcript before new audio opens the next segment.
	// commitResponseTimeoutMS is only a safety ceiling if IsFinal never arrives (0 = no ceiling).
	commitResponseTimeoutMS := cfg.SttCommitResponseTimeoutMs
	var sttDeferred []vadChunk
	var awaitingOutboundCommitted bool
	var outboundCommitDeadline time.Time

	if cfg.VadDebugEnabled {
		fmt.Fprintf(os.Stderr, "[vocode-vad] verbose VAD logging on (VOCODE_VOICE_VAD_DEBUG=1)\n")
		stderrSync()
	}
	if sttPipelineDebugEnabled() {
		fmt.Fprintf(os.Stderr, "[vocode-stt] pipeline logging on — commit, hold, committed_transcript (VOCODE_VOICE_STT_DEBUG=1)\n")
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

		// Apply any pending live config updates before draining STT events.
		reload := false
		drainUpdates:
		for {
			select {
			case <-a.cfgUpdateCh:
				reload = true
			default:
				break drainUpdates
			}
		}
		if reload {
			newCfg := a.getSidecarConfig()
			prevCfg := lastCfg

			// Always refresh the commit timeout: it only matters for future commit holds.
			commitResponseTimeoutMS = newCfg.SttCommitResponseTimeoutMs

			sttChanged := newCfg.SttModelId != prevCfg.SttModelId || newCfg.SttLanguageCode != prevCfg.SttLanguageCode
			vadChanged :=
				newCfg.VadDebugEnabled != prevCfg.VadDebugEnabled ||
					newCfg.VadThresholdMultiplier != prevCfg.VadThresholdMultiplier ||
					newCfg.VadMinEnergyFloor != prevCfg.VadMinEnergyFloor ||
					newCfg.VadStartMs != prevCfg.VadStartMs ||
					newCfg.VadEndMs != prevCfg.VadEndMs ||
					newCfg.VadPrerollMs != prevCfg.VadPrerollMs ||
					newCfg.StreamMinChunkMs != prevCfg.StreamMinChunkMs ||
					newCfg.StreamMaxChunkMs != prevCfg.StreamMaxChunkMs ||
					newCfg.StreamMaxUtteranceMs != prevCfg.StreamMaxUtteranceMs

			if sttChanged {
				// Close the websocket so we don't keep draining a client that no longer matches tuning.
				_ = client.Close()
				client, err = stt.NewElevenLabsStreamingClient(ctx, apiKey, newCfg.SttModelId, 16000, newCfg.SttLanguageCode)
				if err != nil {
					_ = a.write(Event{Type: "error", Message: fmt.Sprintf("failed to reconnect elevenlabs streaming stt: %v", err)})
					return
				}
				// Reset hold state so segmentation & commit behavior is consistent with the new stream.
				sttDeferred = nil
				awaitingOutboundCommitted = false
				outboundCommitDeadline = time.Time{}
				contextWindow = newUtteranceWindow(4, 1200)
			}

			if vadChanged {
				minChunkBytes = bytesPerSecond * int64(newCfg.StreamMinChunkMs) / 1000
				if minChunkBytes <= 0 {
					minChunkBytes = 6400
				}
				maxChunkBytes = bytesPerSecond * int64(newCfg.StreamMaxChunkMs) / 1000
				if maxChunkBytes < minChunkBytes {
					maxChunkBytes = minChunkBytes
				}
				vad = newLocalVAD(
					16000,
					int(minChunkBytes),
					int(maxChunkBytes),
					newCfg.StreamMaxUtteranceMs,
					newCfg.VadThresholdMultiplier,
					newCfg.VadMinEnergyFloor,
					newCfg.VadStartMs,
					newCfg.VadEndMs,
					newCfg.VadPrerollMs,
					newCfg.VadDebugEnabled,
				)

				if newCfg.VadDebugEnabled && !prevCfg.VadDebugEnabled {
					fmt.Fprintf(os.Stderr, "[vocode-vad] verbose VAD logging enabled (live config)\n")
					stderrSync()
				}
			}

			lastCfg = newCfg
		}

		// Drain STT events before blocking on the mic. Otherwise a closed WebSocket can sit in
		// client.Events() for seconds while Read() waits for audio, leaving the UI stuck on a partial.
		stop, sawCommitted := a.drainSTTClientEvents(client, contextWindow)
		if stop {
			return
		}
		if err := a.considerReleasingOutboundHold(client, contextWindow, &awaitingOutboundCommitted, &outboundCommitDeadline, &sttDeferred, sawCommitted); err != nil {
			_ = a.write(Event{Type: "error", Message: fmt.Sprintf("elevenlabs streaming send failed: %v", err)})
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
				if c.commit {
					if err := a.endOutboundHold(client, contextWindow, &awaitingOutboundCommitted, &outboundCommitDeadline, &sttDeferred, "pre_commit"); err != nil {
						_ = a.write(Event{Type: "error", Message: fmt.Sprintf("elevenlabs streaming send failed: %v", err)})
						return
					}
					if err := a.sendSTTChunk(client, contextWindow, c.pcm, c.commit); err != nil {
						_ = a.write(Event{Type: "error", Message: fmt.Sprintf("elevenlabs streaming send failed: %v", err)})
						return
					}
					awaitingOutboundCommitted = true
					if commitResponseTimeoutMS > 0 {
						outboundCommitDeadline = time.Now().Add(time.Duration(commitResponseTimeoutMS) * time.Millisecond)
					} else {
						outboundCommitDeadline = time.Time{}
					}
					if sttPipelineDebugEnabled() {
						fmt.Fprintf(os.Stderr, "[vocode-stt] commit_sent pcm_bytes=%d commit_only=%v hold_max_ms=%d\n",
							len(c.pcm), len(c.pcm) == 0, commitResponseTimeoutMS)
						stderrSync()
					}
					continue
				}
				if awaitingOutboundCommitted {
					sttDeferred = append(sttDeferred, c)
					continue
				}
				if err := a.sendSTTChunk(client, contextWindow, c.pcm, false); err != nil {
					_ = a.write(Event{Type: "error", Message: fmt.Sprintf("elevenlabs streaming send failed: %v", err)})
					return
				}
			}
		}

		stop, sawCommitted = a.drainSTTClientEvents(client, contextWindow)
		if stop {
			return
		}
		if err := a.considerReleasingOutboundHold(client, contextWindow, &awaitingOutboundCommitted, &outboundCommitDeadline, &sttDeferred, sawCommitted); err != nil {
			_ = a.write(Event{Type: "error", Message: fmt.Sprintf("elevenlabs streaming send failed: %v", err)})
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
				if err := a.endOutboundHold(client, contextWindow, &awaitingOutboundCommitted, &outboundCommitDeadline, &sttDeferred, "mic_eof"); err != nil {
					_ = a.write(Event{Type: "error", Message: fmt.Sprintf("elevenlabs streaming send failed: %v", err)})
					return
				}
				for _, c := range vad.flush() {
					if err := a.sendSTTChunk(client, contextWindow, c.pcm, c.commit); err != nil {
						_ = a.write(Event{Type: "error", Message: fmt.Sprintf("elevenlabs streaming send failed: %v", err)})
						return
					}
					if sttPipelineDebugEnabled() && c.commit {
						fmt.Fprintf(os.Stderr, "[vocode-stt] commit_sent pcm_bytes=%d commit_only=%v (mic_eof_flush)\n", len(c.pcm), len(c.pcm) == 0)
						stderrSync()
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
		return client.SendInputAudioChunk(pcm, false, elevenLabsPreviousText(contextWindow))
	}
	if len(pcm) == 0 {
		return client.SendInputAudioChunk(nil, true, "")
	}
	if err := client.SendInputAudioChunk(pcm, false, elevenLabsPreviousText(contextWindow)); err != nil {
		return err
	}
	return client.SendInputAudioChunk(nil, true, "")
}

func (a *App) flushSTTDeferred(client *stt.ElevenLabsStreamingClient, contextWindow *utteranceWindow, deferred *[]vadChunk) error {
	for _, c := range *deferred {
		if err := a.sendSTTChunk(client, contextWindow, c.pcm, c.commit); err != nil {
			return err
		}
	}
	*deferred = (*deferred)[:0]
	return nil
}

func (a *App) endOutboundHold(client *stt.ElevenLabsStreamingClient, contextWindow *utteranceWindow, awaiting *bool, deadline *time.Time, deferred *[]vadChunk, reason string) error {
	if sttPipelineDebugEnabled() && *awaiting {
		n := len(*deferred)
		pcmBytes := 0
		for _, c := range *deferred {
			pcmBytes += len(c.pcm)
		}
		fmt.Fprintf(os.Stderr, "[vocode-stt] hold_end reason=%s deferred_chunks=%d deferred_pcm_bytes=%d\n", reason, n, pcmBytes)
		stderrSync()
	}
	*awaiting = false
	*deadline = time.Time{}
	return a.flushSTTDeferred(client, contextWindow, deferred)
}

func (a *App) considerReleasingOutboundHold(client *stt.ElevenLabsStreamingClient, contextWindow *utteranceWindow, awaiting *bool, deadline *time.Time, deferred *[]vadChunk, sawCommitted bool) error {
	if !*awaiting {
		return nil
	}
	if sawCommitted {
		return a.endOutboundHold(client, contextWindow, awaiting, deadline, deferred, "committed")
	}
	if !deadline.IsZero() && time.Now().After(*deadline) {
		return a.endOutboundHold(client, contextWindow, awaiting, deadline, deferred, "timeout")
	}
	return nil
}

// drainSTTClientEvents pulls every event currently queued on the ElevenLabs client (non-blocking).
// sawCommitted is true if any drained event had IsFinal (committed transcript).
// Returns stop=true if the transcribe loop should exit (error, channel closed, or write failure).
func (a *App) drainSTTClientEvents(client *stt.ElevenLabsStreamingClient, contextWindow *utteranceWindow) (stop bool, sawCommitted bool) {
	for {
		select {
		case evt, ok := <-client.Events():
			if !ok {
				return true, false
			}
			if evt.Error != nil {
				_ = a.write(Event{Type: "error", Message: fmt.Sprintf("elevenlabs streaming stt failed: %v", evt.Error)})
				return true, false
			}
			if strings.TrimSpace(evt.Text) != "" {
				if err := a.writeTranscript(evt.Text, evt.IsFinal); err != nil {
					return true, false
				}
				if evt.IsFinal {
					contextWindow.AddUtterance(evt.Text)
					sawCommitted = true
					if sttPipelineDebugEnabled() {
						t := strings.TrimSpace(evt.Text)
						runes := []rune(t)
						preview := t
						if len(runes) > 80 {
							preview = string(runes[:80]) + "…"
						}
						fmt.Fprintf(os.Stderr, "[vocode-stt] committed_transcript chars=%d preview=%q\n", len(runes), preview)
						stderrSync()
					}
				}
			}
		default:
			return false, sawCommitted
		}
	}
}
