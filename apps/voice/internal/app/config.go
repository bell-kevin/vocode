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
