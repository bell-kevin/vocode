//go:build cgo

package mic

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"sync"

	portaudio "github.com/gordonklaus/portaudio"
)

type Recorder struct {
	stream *portaudio.Stream
	in     []int16

	// guarded because Read() + Stop() can happen concurrently.
	stopOnce sync.Once
	stopErr  error

	// Tracks whether a Read() call is currently blocked inside PortAudio.
	// We use this to call portaudio.Terminate() only after Pa_ReadStream
	// is no longer running (prevents access violations on shutdown).
	mu       sync.Mutex
	reading  bool
	readCond *sync.Cond

	pending []byte
	tmpBuf  []byte
}

func Start(ctx context.Context, params StartParams) (*Recorder, error) {
	if params.SampleRateHz <= 0 {
		params.SampleRateHz = 16000
	}
	if params.Channels <= 0 {
		params.Channels = 1
	}

	// The WAV encoder and ElevenLabs STT path currently assume mono.
	if params.Channels != 1 {
		return nil, errors.New("portaudio recorder currently supports mono only (channels=1)")
	}

	// Keep this small-ish for responsive segmentation downstream.
	const framesPerBuffer = 1024

	if err := portaudio.Initialize(); err != nil {
		return nil, err
	}

	in := make([]int16, params.Channels*framesPerBuffer)
	stream, err := portaudio.OpenDefaultStream(
		params.Channels, 0,
		float64(params.SampleRateHz),
		framesPerBuffer,
		in,
	)
	if err != nil {
		portaudio.Terminate()
		return nil, err
	}

	if err := stream.Start(); err != nil {
		_ = stream.Close()
		portaudio.Terminate()
		return nil, err
	}

	rec := &Recorder{
		stream: stream,
		in:     in,
		tmpBuf: make([]byte, len(in)*2), // PCM16LE bytes
	}
	rec.readCond = sync.NewCond(&rec.mu)

	// On cancel, stop the PortAudio stream so Read unblocks quickly.
	go func() {
		<-ctx.Done()
		_ = rec.Stop()
	}()

	return rec, nil
}

func (r *Recorder) PCMReader() io.Reader {
	return &pcm16leReader{rec: r}
}

func (r *Recorder) Stop() error {
	if r == nil || r.stream == nil {
		return nil
	}

	r.stopOnce.Do(func() {
		r.mu.Lock()
		st := r.stream
		r.mu.Unlock()
		if st == nil {
			return
		}

		// Unblock Pa_ReadStream first; only Close after Read() has exited (see pcm16leReader.Read).
		if err := st.Stop(); err != nil {
			r.stopErr = err
		}

		r.mu.Lock()
		for r.reading {
			r.readCond.Wait()
		}
		r.stream = nil
		r.mu.Unlock()

		_ = st.Close()
		_ = portaudio.Terminate()
	})

	return r.stopErr
}

type pcm16leReader struct {
	rec *Recorder
}

func (r *pcm16leReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.rec == nil {
		return 0, io.EOF
	}

	r.rec.mu.Lock()
	if r.rec.stream == nil {
		r.rec.mu.Unlock()
		return 0, io.EOF
	}
	r.rec.reading = true
	st := r.rec.stream
	r.rec.mu.Unlock()
	defer func() {
		r.rec.mu.Lock()
		r.rec.reading = false
		r.rec.readCond.Broadcast()
		r.rec.mu.Unlock()
	}()

	// If we have pending bytes from a previous larger frame, flush those first.
	if len(r.rec.pending) > 0 {
		n := copy(p, r.rec.pending)
		r.rec.pending = r.rec.pending[n:]
		return n, nil
	}

	// Read fills r.rec.in (int16 samples).
	if err := st.Read(); err != nil {
		return 0, err
	}

	// Convert int16 samples to PCM16LE bytes into tmpBuf.
	for i, s := range r.rec.in {
		binary.LittleEndian.PutUint16(r.rec.tmpBuf[i*2:], uint16(s))
	}
	r.rec.pending = r.rec.tmpBuf

	n := copy(p, r.rec.pending)
	r.rec.pending = r.rec.pending[n:]
	return n, nil
}
