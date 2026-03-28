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
	out io.Writer

	mu sync.Mutex

	running bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func New(in io.Reader, out io.Writer) *App {
	return &App{
		in:  in,
		out: out,
	}
}

func (a *App) Run() error {
	if err := a.write(Event{
		Type:    "ready",
		Version: "0.1.0",
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
	a.mu.Lock()
	defer a.mu.Unlock()
	enc := json.NewEncoder(a.out)
	return enc.Encode(evt)
}
