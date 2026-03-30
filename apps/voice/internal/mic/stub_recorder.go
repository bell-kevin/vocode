//go:build !cgo

package mic

import (
	"context"
	"errors"
	"io"
	"strings"
)

// Recorder is a no-op stub used when cgo is disabled.
//
// The voice sidecar cannot capture microphone audio without the cgo/portaudio
// implementation, but we still want the rest of the codebase to compile and
// unit test on non-cgo platforms.
type Recorder struct{}

func Start(_ context.Context, _ StartParams) (*Recorder, error) {
	return nil, errors.New("mic capture requires cgo/portaudio (not available in this build)")
}

func (r *Recorder) PCMReader() io.Reader {
	// If Start failed, this shouldn't be used; still return an empty reader
	// to keep it safe if called inadvertently.
	return strings.NewReader("")
}

func (r *Recorder) Stop() error {
	return nil
}

