package app

import (
	"encoding/binary"
	"testing"
)

func TestPCM16RMS(t *testing.T) {
	pcm := make([]byte, 8)
	putInt16(pcm[0:], 1000)
	putInt16(pcm[2:], -1000)
	putInt16(pcm[4:], 1000)
	putInt16(pcm[6:], -1000)

	got := pcm16RMS(pcm)
	if got < 999 || got > 1001 {
		t.Fatalf("expected RMS near 1000, got %f", got)
	}
}

func TestLocalVAD_SpeechThenSilenceProducesCommit(t *testing.T) {
	t.Setenv("VOCODE_VOICE_VAD_START_MS", "40")
	t.Setenv("VOCODE_VOICE_VAD_END_MS", "60")
	t.Setenv("VOCODE_VOICE_VAD_PREROLL_MS", "20")
	t.Setenv("VOCODE_VOICE_VAD_THRESHOLD_MULTIPLIER", "1.2")

	v := newLocalVAD(16000, 640)
	speech := makeFrame(640, 1600)
	silence := makeFrame(640, 0)

	var commits int
	for i := 0; i < 6; i++ {
		for _, c := range v.process(speech) {
			if c.commit {
				commits++
			}
		}
	}
	for i := 0; i < 8; i++ {
		for _, c := range v.process(silence) {
			if c.commit {
				commits++
			}
		}
	}
	if commits == 0 {
		t.Fatalf("expected at least one commit chunk")
	}
}

func makeFrame(size int, amp int16) []byte {
	b := make([]byte, size)
	for i := 0; i+1 < size; i += 2 {
		putInt16(b[i:], amp)
	}
	return b
}

func putInt16(dst []byte, v int16) {
	binary.LittleEndian.PutUint16(dst, uint16(v))
}
