package agentcontext

import "vocoding.net/vocode/v2/apps/daemon/internal/intents"

// TurnContext is everything the agent model sees for one [agent.ModelClient.NextTurn] call.
type TurnContext struct {
	TranscriptText string
	// SucceededIntents lists intents the host already applied successfully for this voice session context
	// (plus intents dispatched earlier in the same Execute), for repair / partial-batch prompts.
	SucceededIntents   []intents.Intent
	FailedIntents      []FailedIntent
	SkippedIntents     []intents.Intent
	IntentApplyHistory []IntentApplyRecord
	Editor             EditorSnapshot
	Gathered           Gathered
	Limits             TurnLimits
}

// TurnLimits are per-transcript caps the planner should respect.
// These values come from the daemon's effective execution caps (env defaults overridden by voice.transcript daemonConfig).
type TurnLimits struct {
	MaxContextRounds int `json:"maxContextRounds"`
}
