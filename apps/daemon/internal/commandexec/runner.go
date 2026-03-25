package commandexec

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"time"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type Runner struct {
	outputLimitBytes int64
}

func NewRunner() *Runner {
	return &Runner{outputLimitBytes: 1024 * 1024} // 1MiB per stream
}

type RunOutput struct {
	exitCode int64
	stdout   string
	stderr   string
	timeout  bool
}

type capWriter struct {
	w         io.Writer
	remaining int64
}

func newCapWriter(w io.Writer, limitBytes int64) io.Writer {
	return &capWriter{w: w, remaining: limitBytes}
}

func (cw *capWriter) Write(p []byte) (int, error) {
	if cw.remaining <= 0 {
		// Discard without error once we hit the limit.
		return len(p), nil
	}

	origLen := int64(len(p))
	if origLen > cw.remaining {
		p = p[:cw.remaining]
	}

	n, err := cw.w.Write(p)
	cw.remaining -= int64(n)

	// Report the original length so callers don't treat this as a short write.
	return int(origLen), err
}

func (r *Runner) Run(params protocol.CommandRunParams) (RunOutput, error) {
	ctx := context.Background()

	var cancel context.CancelFunc
	if params.TimeoutMs != nil && *params.TimeoutMs > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*params.TimeoutMs)*time.Millisecond)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, params.Command, params.Args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = newCapWriter(&stdoutBuf, r.outputLimitBytes)
	cmd.Stderr = newCapWriter(&stderrBuf, r.outputLimitBytes)

	err := cmd.Run()
	if err == nil {
		return RunOutput{
			exitCode: int64(cmd.ProcessState.ExitCode()),
			stdout:   stdoutBuf.String(),
			stderr:   stderrBuf.String(),
			timeout:  false,
		}, nil
	}

	// Context deadline means we should classify as a timeout.
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return RunOutput{
			exitCode: -1,
			stdout:   stdoutBuf.String(),
			stderr:   stderrBuf.String(),
			timeout:  true,
		}, nil
	}

	// ExitError indicates the process ran; we treat that as success at the
	// transport level and propagate exit code + output.
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode := exitErr.ExitCode()
		return RunOutput{
			exitCode: int64(exitCode),
			stdout:   stdoutBuf.String(),
			stderr:   stderrBuf.String(),
			timeout:  false,
		}, nil
	}

	return RunOutput{}, err
}
