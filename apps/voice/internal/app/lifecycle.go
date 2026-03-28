package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"vocoding.net/vocode/v2/apps/voice/internal/mic"
)

func (a *App) handleStart() error {
	if err := a.write(Event{Type: "state", State: "starting"}); err != nil {
		return err
	}
	if a.running {
		// Already running; treat as idempotent.
		return nil
	}

	if !sttEnabled() {
		// STT disabled: allow devs to manually send transcripts from the extension.
		// We still report a listening state so the UX is consistent.
		return a.write(Event{Type: "state", State: "listening"})
	}

	apiKey := strings.TrimSpace(os.Getenv("ELEVENLABS_API_KEY"))
	if apiKey == "" {
		return a.write(Event{Type: "error", Message: "ELEVENLABS_API_KEY is not set"})
	}

	ctx, cancel := context.WithCancel(context.Background())
	rec, err := mic.Start(ctx, mic.StartParams{SampleRateHz: 16000, Channels: 1})
	if err != nil {
		cancel()
		return a.write(Event{Type: "error", Message: fmt.Sprintf("failed to start microphone recorder: %v", err)})
	}

	a.running = true
	a.cancel = cancel
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		defer a.markTranscribeFinished()
		a.transcribeLoop(ctx, apiKey, sttModelID(), rec)
	}()

	return a.write(Event{Type: "state", State: "listening"})
}

func (a *App) handleStop() error {
	a.stopIfRunning()
	return a.write(Event{Type: "state", State: "stopped"})
}

func (a *App) handleShutdown() (bool, error) {
	a.stopIfRunning()
	if err := a.write(Event{Type: "state", State: "shutdown"}); err != nil {
		return false, err
	}
	return true, nil
}

func (a *App) stopIfRunning() {
	if a.running {
		a.running = false
		if a.cancel != nil {
			a.cancel()
			a.cancel = nil
		}
		a.wg.Wait()
	}
}

// markTranscribeFinished runs when the transcribe goroutine exits. If the session was still marked
// running (e.g. STT WebSocket closed without handleStop), clear flags and emit stopped so clients
// can restart and the extension is not stuck thinking the session is active.
func (a *App) markTranscribeFinished() {
	if !a.running {
		return
	}
	a.running = false
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	_ = a.write(Event{Type: "state", State: "stopped"})
}
