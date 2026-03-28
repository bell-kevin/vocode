package app

type Request struct {
	Type string `json:"type"`
}

type Event struct {
	Type    string `json:"type"`
	State   string `json:"state,omitempty"`
	Message string `json:"message,omitempty"`
	Version string `json:"version,omitempty"`
	Text    string `json:"text,omitempty"`
	// Committed indicates whether this transcript is a final/committed hypothesis.
	// When omitted, the event is considered backwards-compatible.
	Committed *bool `json:"committed,omitempty"`
	// Speaking and Rms are set for type "audio_meter" (mic level + VAD in-speech for extension UI).
	Speaking *bool    `json:"speaking,omitempty"`
	Rms      *float64 `json:"rms,omitempty"`
}
