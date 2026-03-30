package app

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
)

type vadChunk struct {
	pcm    []byte
	commit bool
}

type localVAD struct {
	frameBytes          int
	minChunkBytes       int
	maxChunkBytes       int
	startFrames         int
	endFrames           int
	utteranceMaxEnabled bool
	maxUtterFrames      int
	threshold           float64
	minEnergyFloor      float64

	inSpeech       bool
	speechFrames   int
	inSpeechFrames int
	silenceFrames  int
	noiseFloor     float64

	prerollFrames [][]byte
	prerollCap    int
	sendBuf       []byte

	// lastFrameRMS is the RMS energy of the most recently processed 20ms frame (for extension UI).
	lastFrameRMS float64
	// lastFrameSpeechClass is true when the last frame cleared the energy gate (same as isSpeech).
	lastFrameSpeechClass bool
	// meterSmoothed is a fast-attack / slow-decay envelope of RMS for a steadier level meter.
	meterSmoothed float64

	debugEnabled bool
}

func newLocalVAD(
	sampleRate int,
	minChunkBytes int,
	maxChunkBytes int,
	maxUtteranceMS int,
	threshold float64,
	minEnergyFloor float64,
	vadStartMS int,
	vadEndMS int,
	vadPrerollMS int,
	debugEnabled bool,
) *localVAD {
	const frameMS = 20
	frameBytes := sampleRate * 2 * frameMS / 1000
	if frameBytes <= 0 {
		frameBytes = 640
	}
	if minChunkBytes <= 0 {
		minChunkBytes = frameBytes * 10
	}
	if maxChunkBytes < minChunkBytes {
		maxChunkBytes = minChunkBytes
	}

	startFrames := maxInt(1, vadStartMS/frameMS)
	endFrames := maxInt(1, vadEndMS/frameMS)
	prerollCap := maxInt(0, vadPrerollMS/frameMS)
	utteranceMaxEnabled := maxUtteranceMS > 0
	maxUtterFrames := 0
	if utteranceMaxEnabled {
		maxUtterFrames = maxInt(1, maxUtteranceMS/frameMS)
	}

	return &localVAD{
		frameBytes:          frameBytes,
		minChunkBytes:       minChunkBytes,
		maxChunkBytes:       maxChunkBytes,
		startFrames:         startFrames,
		endFrames:           endFrames,
		utteranceMaxEnabled: utteranceMaxEnabled,
		maxUtterFrames:      maxUtterFrames,
		threshold:           threshold,
		minEnergyFloor:      minEnergyFloor,
		noiseFloor:          minEnergyFloor,
		prerollCap:          prerollCap,
		prerollFrames:       make([][]byte, 0, prerollCap),
		debugEnabled:        debugEnabled,
	}
}

func stderrSync() {
	_ = os.Stderr.Sync()
}

func (v *localVAD) dbg(format string, args ...any) {
	if !v.debugEnabled {
		return
	}
	fmt.Fprintf(os.Stderr, "[vocode-vad] "+format+"\n", args...)
	stderrSync()
}

// MeterSnapshot reports whether the UI should show “speaking” and a smoothed RMS for the level bar.
// Speaking is true during an utterance (inSpeech) or when the current frame’s energy passes the gate,
// so short dips between phonemes don’t flash “Quiet” while you’re still talking.
func (v *localVAD) MeterSnapshot() (speaking bool, rms float64) {
	return v.inSpeech || v.lastFrameSpeechClass, v.meterSmoothed
}

func (v *localVAD) process(frame []byte) []vadChunk {
	if len(frame) == 0 {
		return nil
	}
	energy := pcm16RMS(frame)
	v.lastFrameRMS = energy
	v.smoothMeterLevel(energy)
	isSpeech := energy > math.Max(v.noiseFloor*v.threshold, v.minEnergyFloor)
	v.lastFrameSpeechClass = isSpeech
	chunks := make([]vadChunk, 0, 2)

	if !v.inSpeech {
		v.pushPreroll(frame)
		if isSpeech {
			v.speechFrames++
		} else {
			v.speechFrames = 0
			v.updateNoiseFloor(energy)
		}
		if v.speechFrames >= v.startFrames {
			v.dbg(
				"speech_start energy=%.1f noise_floor=%.1f speech_threshold=%.1f (start_frames=%d)",
				energy,
				v.noiseFloor,
				math.Max(v.noiseFloor*v.threshold, v.minEnergyFloor),
				v.startFrames,
			)
			v.inSpeech = true
			v.inSpeechFrames = 0
			v.silenceFrames = 0
			v.speechFrames = 0
			v.prerollFrames = v.prerollFrames[:0]
			// Preroll was already streamed below as non-commit PCM; only buffer this frame for
			// commit segmentation (same audio was not duplicated in sendBuf).
			v.sendBuf = append(v.sendBuf, frame...)
			chunks = append(chunks, v.drainNonCommitChunks()...)
			return chunks
		}
		// Stream every frame upstream while waiting for speech_start so STT doesn’t miss the
		// start of the next phrase after a commit (local VAD silence).
		return []vadChunk{{pcm: append([]byte(nil), frame...), commit: false}}
	}

	v.sendBuf = append(v.sendBuf, frame...)
	v.inSpeechFrames++
	chunks = append(chunks, v.drainNonCommitChunks()...)

	if isSpeech {
		v.silenceFrames = 0
	} else {
		v.silenceFrames++
		v.updateNoiseFloor(energy)
	}

	if v.silenceFrames >= v.endFrames {
		if len(v.sendBuf) > 0 {
			v.dbg("commit utterance_end (silence) silence_frames=%d end_frames=%d pcm_bytes=%d", v.silenceFrames, v.endFrames, len(v.sendBuf))
			chunks = append(chunks, vadChunk{pcm: append([]byte(nil), v.sendBuf...), commit: true})
		} else {
			// Audio for this utterance was already sent as non-commit chunks (sendBuf fully drained
			// while trailing silence used large chunk sizes). STT still needs a finalize commit or
			// the segment stays open and the client shows a stuck partial.
			v.dbg("commit utterance_end (silence) pcm_bytes=0 commit_only (drained tail)")
			chunks = append(chunks, vadChunk{pcm: nil, commit: true})
		}
		v.sendBuf = nil
		v.inSpeech = false
		v.inSpeechFrames = 0
		v.silenceFrames = 0
		v.speechFrames = 0
		return chunks
	}

	// Optional periodic commits for long continuous speech (VOCODE_VOICE_STREAM_MAX_UTTERANCE_MS > 0).
	if v.utteranceMaxEnabled && v.inSpeechFrames >= v.maxUtterFrames && len(v.sendBuf) > 0 {
		v.dbg("commit utterance_max in_speech_frames=%d max_frames=%d pcm_bytes=%d", v.inSpeechFrames, v.maxUtterFrames, len(v.sendBuf))
		chunks = append(chunks, vadChunk{pcm: append([]byte(nil), v.sendBuf...), commit: true})
		v.sendBuf = nil
		v.inSpeechFrames = 0
	}
	return chunks
}

func (v *localVAD) flush() []vadChunk {
	if len(v.sendBuf) > 0 {
		v.dbg("commit flush pcm_bytes=%d", len(v.sendBuf))
		ch := vadChunk{pcm: append([]byte(nil), v.sendBuf...), commit: true}
		v.sendBuf = nil
		v.inSpeech = false
		return []vadChunk{ch}
	}
	if v.inSpeech {
		v.dbg("commit flush pcm_bytes=0 commit_only (in speech, drained)")
		v.inSpeech = false
		return []vadChunk{{pcm: nil, commit: true}}
	}
	return nil
}

func (v *localVAD) currentChunkBytes() int {
	// During active speech bursts keep chunks small; when trailing toward silence,
	// increase chunk size to reduce churn.
	if v.silenceFrames > 0 {
		return v.maxChunkBytes
	}
	return v.minChunkBytes
}

func (v *localVAD) drainNonCommitChunks() []vadChunk {
	chunkBytes := v.currentChunkBytes()
	if chunkBytes <= 0 {
		return nil
	}
	out := make([]vadChunk, 0, 2)
	// Keep a small tail buffered so utterance end can emit commit=true.
	for len(v.sendBuf) > chunkBytes {
		out = append(out, vadChunk{
			pcm:    append([]byte(nil), v.sendBuf[:chunkBytes]...),
			commit: false,
		})
		v.sendBuf = v.sendBuf[chunkBytes:]
	}
	return out
}

func (v *localVAD) pushPreroll(frame []byte) {
	if v.prerollCap == 0 {
		return
	}
	cp := append([]byte(nil), frame...)
	if len(v.prerollFrames) == v.prerollCap {
		copy(v.prerollFrames, v.prerollFrames[1:])
		v.prerollFrames[len(v.prerollFrames)-1] = cp
		return
	}
	v.prerollFrames = append(v.prerollFrames, cp)
}

func (v *localVAD) smoothMeterLevel(energy float64) {
	// Attack follows peaks quickly; decay slowly so the bar matches perceived loudness.
	const attackMix = 0.42
	const decayMix = 0.14
	if v.meterSmoothed <= 0 {
		v.meterSmoothed = energy
		return
	}
	if energy >= v.meterSmoothed {
		v.meterSmoothed = (1-attackMix)*v.meterSmoothed + attackMix*energy
	} else {
		v.meterSmoothed = (1-decayMix)*v.meterSmoothed + decayMix*energy
	}
}

func (v *localVAD) updateNoiseFloor(energy float64) {
	// Slow adaptation to background noise while preserving speech contrast.
	v.noiseFloor = (0.95 * v.noiseFloor) + (0.05 * energy)
	if v.noiseFloor < v.minEnergyFloor {
		v.noiseFloor = v.minEnergyFloor
	}
}

func pcm16RMS(pcm []byte) float64 {
	if len(pcm) < 2 {
		return 0
	}
	sampleCount := len(pcm) / 2
	if sampleCount == 0 {
		return 0
	}
	var sumSquares float64
	for i := 0; i+1 < len(pcm); i += 2 {
		s := int16(binary.LittleEndian.Uint16(pcm[i:]))
		f := float64(s)
		sumSquares += f * f
	}
	return math.Sqrt(sumSquares / float64(sampleCount))
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
