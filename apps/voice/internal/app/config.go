package app

import "strings"

// SidecarConfig is the in-memory set of current non-secret tuning values.
// It starts with internal defaults, but can be updated
// at runtime without restarting the sidecar via a JSON event request.
type SidecarConfig struct {
	// STT tuning.
	SttModelId     string
	SttLanguage    string
	SttLanguageCode string

	// VAD tuning.
	VadDebugEnabled        bool
	VadThresholdMultiplier float64
	VadMinEnergyFloor      float64
	VadStartMs             int
	VadEndMs               int
	VadPrerollMs           int

	// STT streaming tuning.
	SttCommitResponseTimeoutMs int
	StreamMinChunkMs           int
	StreamMaxChunkMs           int
	StreamMaxUtteranceMs      int
}

func defaultSidecarConfig() SidecarConfig {
	lang := "en"
	return SidecarConfig{
		SttModelId:      "scribe_v2_realtime",
		SttLanguage:     lang,
		SttLanguageCode: lang,

		VadDebugEnabled:        false,
		VadThresholdMultiplier: 1.65,
		VadMinEnergyFloor:      100,
		VadStartMs:             60,
		VadEndMs:               750,
		VadPrerollMs:           320,

		SttCommitResponseTimeoutMs: 5000,
		StreamMinChunkMs:           200,
		StreamMaxChunkMs:           500,
		StreamMaxUtteranceMs:      0,
	}
}

func normalizeStreamMaxUtteranceMS(n int) int {
	// 0 = off.
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

// sttPipelineDebugEnabled logs ElevenLabs STT pipeline events: commit sent, outbound hold,
// committed_transcript received, timeout flush (stderr only; never stdout JSON).
func sttPipelineDebugEnabled() bool {
	// Backwards compat: for now keep this as a compile-time default.
	// If you want it live-configurable, add it to ConfigPatch.
	return false
}

func normalizeLanguageCode(raw string) (language string, languageCode string) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "en", "en"
	}
	if strings.EqualFold(v, "auto") {
		// Omit language_code entirely for server autodetect.
		return "auto", ""
	}
	return v, v
}
