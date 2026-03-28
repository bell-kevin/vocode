package app

type Request struct {
	Type string `json:"type"`
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
