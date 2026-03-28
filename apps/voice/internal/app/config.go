package app

import (
	"os"
	"strconv"
	"strings"
)

func sttEnabled() bool {
	// Default enabled to preserve existing behavior.
	v := strings.TrimSpace(os.Getenv("VOCODE_VOICE_STT_ENABLED"))
	if v == "" {
		return true
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "y", "on", "enabled":
		return true
	case "0", "false", "no", "n", "off", "disabled":
		return false
	default:
		// Fail open to avoid confusing "no transcripts" because of a typo.
		return true
	}
}

func sttModelID() string {
	v := strings.TrimSpace(os.Getenv("ELEVENLABS_STT_MODEL_ID"))
	if v == "" {
		return "scribe_v2"
	}
	return v
}

func vadThresholdMultiplier() float64 {
	return envFloat("VOCODE_VOICE_VAD_THRESHOLD_MULTIPLIER", 2.0, 1.0, 10.0)
}

func vadStartMS() int {
	return envInt("VOCODE_VOICE_VAD_START_MS", 60, 20, 2000)
}

func vadEndMS() int {
	return envInt("VOCODE_VOICE_VAD_END_MS", 500, 60, 5000)
}

func vadPrerollMS() int {
	return envInt("VOCODE_VOICE_VAD_PREROLL_MS", 200, 0, 1000)
}

func streamMinChunkMS() int {
	return envInt("VOCODE_VOICE_STREAM_MIN_CHUNK_MS", 200, 50, 2000)
}

func streamMaxChunkMS() int {
	return envInt("VOCODE_VOICE_STREAM_MAX_CHUNK_MS", 500, 50, 3000)
}

func streamMaxUtteranceMS() int {
	return envInt("VOCODE_VOICE_STREAM_MAX_UTTERANCE_MS", 4000, 500, 20000)
}

// vadDebugEnabled logs VAD decisions to stderr (never stdout — stdout is JSON for the extension).
func vadDebugEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("VOCODE_VOICE_VAD_DEBUG")))
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
