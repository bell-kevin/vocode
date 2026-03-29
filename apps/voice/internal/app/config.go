package app

import (
	"os"
	"strconv"
	"strings"
)

func sttModelID() string {
	v := strings.TrimSpace(os.Getenv("ELEVENLABS_STT_MODEL_ID"))
	if v == "" {
		return "scribe_v2_realtime"
	}
	return v
}

// sttLanguageCode is passed to ElevenLabs realtime as language_code (ISO 639-1) to avoid random
// language flips. Set ELEVENLABS_STT_LANGUAGE=auto to omit and allow server auto-detection.
func sttLanguageCode() string {
	v := strings.TrimSpace(os.Getenv("ELEVENLABS_STT_LANGUAGE"))
	if strings.EqualFold(v, "auto") {
		return ""
	}
	if v == "" {
		return "en"
	}
	return v
}

func vadThresholdMultiplier() float64 {
	// Lower defaults help real mics that are gain-limited but still sound “loud” in the room.
	return envFloat("VOCODE_VOICE_VAD_THRESHOLD_MULTIPLIER", 1.65, 1.0, 10.0)
}

// vadMinEnergyFloor is the minimum RMS (PCM16, per 20ms frame) used for speech gating and noise floor.
func vadMinEnergyFloor() float64 {
	return envFloat("VOCODE_VOICE_VAD_MIN_ENERGY_FLOOR", 100, 30, 800)
}

func vadStartMS() int {
	return envInt("VOCODE_VOICE_VAD_START_MS", 60, 20, 2000)
}

func vadEndMS() int {
	return envInt("VOCODE_VOICE_VAD_END_MS", 750, 60, 5000)
}

func vadPrerollMS() int {
	// Extra audio kept before speech is declared — helps soft word onsets (first syllables).
	return envInt("VOCODE_VOICE_VAD_PREROLL_MS", 320, 0, 1000)
}

// sttCommitResponseTimeoutMS: after commit:true we wait for committed_transcript before sending more
// non-commit PCM; if it never arrives, flush deferred audio after this many ms. 0 = wait indefinitely
// (can grow memory if the stream is broken).
func sttCommitResponseTimeoutMS() int {
	return envInt("VOCODE_VOICE_STT_COMMIT_RESPONSE_TIMEOUT_MS", 5000, 0, 120000)
}

func streamMinChunkMS() int {
	return envInt("VOCODE_VOICE_STREAM_MIN_CHUNK_MS", 200, 50, 2000)
}

func streamMaxChunkMS() int {
	return envInt("VOCODE_VOICE_STREAM_MAX_CHUNK_MS", 500, 50, 3000)
}

// streamMaxUtteranceMS caps how long one spoken segment can grow before VAD forces a commit (ms).
// 0 means off (default): commits only on trailing silence, mic EOF, or Stop Voice — better for long explanations.
// Non-zero values are clamped to [500, 120000].
func streamMaxUtteranceMS() int {
	v := strings.TrimSpace(os.Getenv("VOCODE_VOICE_STREAM_MAX_UTTERANCE_MS"))
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	if n == 0 {
		return 0
	}
	if n < 500 {
		return 500
	}
	const maxMS = 120_000
	if n > maxMS {
		return maxMS
	}
	return n
}

// vadDebugEnabled logs per-frame VAD decisions (speech_start, commit reasons) to stderr — noisy.
func vadDebugEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("VOCODE_VOICE_VAD_DEBUG")))
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

// sttPipelineDebugEnabled logs ElevenLabs STT pipeline events: commit sent, outbound hold,
// committed_transcript received, timeout flush (stderr only; never stdout JSON).
func sttPipelineDebugEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("VOCODE_VOICE_STT_DEBUG")))
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func envInt(name string, defaultValue int, minValue int, maxValue int) int {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultValue
	}
	if n < minValue {
		return minValue
	}
	if n > maxValue {
		return maxValue
	}
	return n
}

func envFloat(name string, defaultValue float64, minValue float64, maxValue float64) float64 {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return defaultValue
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return defaultValue
	}
	if n < minValue {
		return minValue
	}
	if n > maxValue {
		return maxValue
	}
	return n
}
