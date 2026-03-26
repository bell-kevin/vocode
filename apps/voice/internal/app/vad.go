package app

import (
	"encoding/binary"
	"math"
)

type vadChunk struct {
	pcm    []byte
	commit bool
}

type localVAD struct {
	frameBytes       int
	streamChunkBytes int
	startFrames      int
	endFrames        int
	threshold        float64
	minEnergyFloor   float64

	inSpeech     bool
	speechFrames int
	silenceFrames int
	noiseFloor   float64

	prerollFrames [][]byte
	prerollCap    int
	sendBuf       []byte
}

func newLocalVAD(sampleRate int, streamChunkBytes int) *localVAD {
	const frameMS = 20
	frameBytes := sampleRate * 2 * frameMS / 1000
	if frameBytes <= 0 {
		frameBytes = 640
	}

	startFrames := maxInt(1, vadStartMS()/frameMS)
	endFrames := maxInt(1, vadEndMS()/frameMS)
	prerollCap := maxInt(0, vadPrerollMS()/frameMS)

	return &localVAD{
		frameBytes:       frameBytes,
		streamChunkBytes: streamChunkBytes,
		startFrames:      startFrames,
		endFrames:        endFrames,
		threshold:        vadThresholdMultiplier(),
		minEnergyFloor:   150.0,
		noiseFloor:       150.0,
		prerollCap:       prerollCap,
		prerollFrames:    make([][]byte, 0, prerollCap),
	}
}

func (v *localVAD) process(frame []byte) []vadChunk {
	if len(frame) == 0 {
		return nil
	}
	energy := pcm16RMS(frame)
	isSpeech := energy > math.Max(v.noiseFloor*v.threshold, v.minEnergyFloor)
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
			v.inSpeech = true
			v.silenceFrames = 0
			v.speechFrames = 0
			for _, f := range v.prerollFrames {
				v.sendBuf = append(v.sendBuf, f...)
			}
			v.prerollFrames = v.prerollFrames[:0]
			chunks = append(chunks, v.drainNonCommitChunks()...)
		}
		return chunks
	}

	v.sendBuf = append(v.sendBuf, frame...)
	chunks = append(chunks, v.drainNonCommitChunks()...)

	if isSpeech {
		v.silenceFrames = 0
	} else {
		v.silenceFrames++
		v.updateNoiseFloor(energy)
	}

	if v.silenceFrames >= v.endFrames {
		if len(v.sendBuf) > 0 {
			chunks = append(chunks, vadChunk{pcm: append([]byte(nil), v.sendBuf...), commit: true})
		}
		v.sendBuf = nil
		v.inSpeech = false
		v.silenceFrames = 0
		v.speechFrames = 0
	}
	return chunks
}

func (v *localVAD) flush() []vadChunk {
	if len(v.sendBuf) == 0 {
		return nil
	}
	ch := vadChunk{pcm: append([]byte(nil), v.sendBuf...), commit: true}
	v.sendBuf = nil
	return []vadChunk{ch}
}

func (v *localVAD) drainNonCommitChunks() []vadChunk {
	if v.streamChunkBytes <= 0 {
		return nil
	}
	out := make([]vadChunk, 0, 2)
	// Keep a small tail buffered so utterance end can emit commit=true.
	for len(v.sendBuf) > v.streamChunkBytes {
		out = append(out, vadChunk{
			pcm:    append([]byte(nil), v.sendBuf[:v.streamChunkBytes]...),
			commit: false,
		})
		v.sendBuf = v.sendBuf[v.streamChunkBytes:]
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
