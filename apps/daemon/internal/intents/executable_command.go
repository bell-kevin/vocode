package intents

type CommandIntent struct {
	Command   string   `json:"command"`
	Args      []string `json:"args,omitempty"`
	TimeoutMs *int64   `json:"timeoutMs,omitempty"`
}
