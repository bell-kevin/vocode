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
	v := newLocalVAD(
		16000,
		640,
		1280,
		2000,
		1.2,
		100,
		40,
		60,
		20,
		false,
	)
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

func TestLocalVAD_Flush_EmptySendBufWhileInSpeech_EmitsCommitOnly(t *testing.T) {
	v := newLocalVAD(
		16000,
		6400,
		16000,
		4000,
		1.65,
		100,
		60,
		750,
		320,
		false,
	)
	v.inSpeech = true
	v.sendBuf = nil
	out := v.flush()
	if len(out) != 1 {
		t.Fatalf("expected one chunk, got %d", len(out))
	}
	if !out[0].commit || len(out[0].pcm) != 0 {
		t.Fatalf("expected commit-only chunk, got commit=%v len(pcm)=%d", out[0].commit, len(out[0].pcm))
	}
	if v.inSpeech {
		t.Fatal("expected inSpeech cleared after flush")
	}
}

func TestLocalVAD_UtteranceMaxDisabled_NoPeriodicCommit(t *testing.T) {
	v := newLocalVAD(
		16000,
		640,
		1280,
		0,
		1.1,
		100,
		20,
		750,
		0,
		false,
	)
	speech := makeFrame(640, 2000)
	commits := 0
	for i := 0; i < 100; i++ {
		for _, c := range v.process(speech) {
			if c.commit {
				commits++
			}
		}
	}
	if commits != 0 {
		t.Fatalf("expected no periodic commits when max utterance off, got %d", commits)
	}
}

func TestLocalVAD_ForceCommitOnLongUtterance(t *testing.T) {
	v := newLocalVAD(
		16000,
		640,
		1280,
		100,
		1.1,
		100,
		20,
		750,
		0,
		false,
	)
	speech := makeFrame(640, 2000)
	commits := 0
	for i := 0; i < 20; i++ {
		for _, c := range v.process(speech) {
			if c.commit {
				commits++
			}
		}
	}
	if commits == 0 {
		t.Fatalf("expected forced commit on long utterance")
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
