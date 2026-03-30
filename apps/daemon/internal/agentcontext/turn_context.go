package agentcontext

import "vocoding.net/vocode/v2/apps/daemon/internal/intents"

// TurnContext is everything the agent model sees for one [agent.ModelClient.NextTurn] call.
type TurnContext struct {
	TranscriptText     string
	SucceededIntents   []intents.Intent
	FailedIntents      []FailedIntent
	SkippedIntents     []intents.Intent
	IntentApplyHistory []IntentApplyRecord
	Editor             EditorSnapshot
	Gathered           Gathered
}
