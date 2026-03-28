package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
)

type App struct {
	in io.Reader
	// Buffered so each JSON line is flushed promptly to the extension (line-oriented protocol).
	outBuf *bufio.Writer

	mu sync.Mutex

	running bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func New(in io.Reader, out io.Writer) *App {
	return &App{
		in:     in,
		outBuf: bufio.NewWriter(out),
	}
}

func (a *App) Run() error {
	if err := a.write(Event{
		Type:    "ready",
		Version: "0.2.0",
		Features: map[string]bool{
			"transcript_committed_field": true,
			"audio_meter":                true,
		},
	}); err != nil {
		return err
	}

	scanner := bufio.NewScanner(a.in)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			if werr := a.write(Event{
				Type:    "error",
				Message: fmt.Sprintf("invalid request json: %v", err),
			}); werr != nil {
				return werr
			}
			continue
		}

		switch req.Type {
		case "start":
			if err := a.handleStart(); err != nil {
				return err
			}
		case "stop":
			if err := a.handleStop(); err != nil {
				return err
			}
		case "shutdown":
			shouldExit, err := a.handleShutdown()
			if err != nil {
				return err
			}
			if shouldExit {
				return nil
			}
		default:
			if err := a.write(Event{
				Type:    "error",
				Message: fmt.Sprintf("unknown request type %q", req.Type),
			}); err != nil {
				return err
			}
		}
	}

	return scanner.Err()
}

func (a *App) write(evt Event) error {
	return a.writeJSON(evt)
}

func (a *App) writeJSON(v any) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if _, err := a.outBuf.Write(b); err != nil {
		return err
	}
	return a.outBuf.Flush()
}

// writeTranscript emits transcript JSON with committed always present (extension treats missing as partial-only).
func (a *App) writeTranscript(text string, committed bool) error {
	return a.writeJSON(struct {
		Type      string `json:"type"`
		Text      string `json:"text"`
		Committed bool   `json:"committed"`
	}{Type: "transcript", Text: text, Committed: committed})
}
