package app

type Request struct {
	Type string `json:"type"`
	// Config carries non-secret tuning overrides to apply without restarting.
	// Api keys must remain in the process environment.
	Config *ConfigPatch `json:"config,omitempty"`
}

type Event struct {
	Type    string `json:"type"`
	State   string `json:"state,omitempty"`
	Message string `json:"message,omitempty"`
	Version string `json:"version,omitempty"`
	// Features is set on "ready" so the extension can detect a stale sidecar binary.
	Features map[string]bool `json:"features,omitempty"`
	Text     string          `json:"text,omitempty"`
	// Committed is used by generic write(); transcript lines use writeTranscript() so committed is always present.
	Committed *bool `json:"committed,omitempty"`
	// Speaking and Rms are set for type "audio_meter" (mic level + VAD in-speech for extension UI).
	Speaking *bool    `json:"speaking,omitempty"`
	Rms      *float64 `json:"rms,omitempty"`
}

// ConfigPatch is a set of optional overrides for the voice sidecar.
// All values are validated/clamped by the sidecar.
type ConfigPatch struct {
	// STT tuning (non-secret).
	SttModelId   *string `json:"sttModelId,omitempty"`
	SttLanguage  *string `json:"sttLanguage,omitempty"`
	VadDebug      *bool   `json:"vadDebug,omitempty"`

	// VAD tuning.
	VadThresholdMultiplier *float64 `json:"vadThresholdMultiplier,omitempty"`
	VadMinEnergyFloor      *float64 `json:"vadMinEnergyFloor,omitempty"`
	VadStartMs             *int     `json:"vadStartMs,omitempty"`
	VadEndMs               *int     `json:"vadEndMs,omitempty"`
	VadPrerollMs           *int     `json:"vadPrerollMs,omitempty"`

	// STT streaming tuning.
	SttCommitResponseTimeoutMs *int `json:"sttCommitResponseTimeoutMs,omitempty"`

	StreamMinChunkMs     *int `json:"streamMinChunkMs,omitempty"`
	StreamMaxChunkMs     *int `json:"streamMaxChunkMs,omitempty"`
	StreamMaxUtteranceMs *int `json:"streamMaxUtteranceMs,omitempty"`
}
