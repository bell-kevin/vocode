package executor

import (
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// loopAdvance controls whether Execute continues after one NextIntent + dispatch cycle.
type loopAdvance byte

const (
	advanceContinue loopAdvance = iota
	advanceBreakLoop
)

// agentLoopState is mutable state for one Execute() run (one voice.transcript).
type agentLoopState struct {
	gathered              agentcontext.Gathered
	completed             []intents.Intent
	failedIntents         []agentcontext.FailedIntent
	contextRounds         int
	consecutiveContextReq int
	editCounter           int
	directives            []protocol.VoiceTranscriptDirective
	batchSourceIntents    []intents.Intent
	transcriptSummary     string
	maxRetries            int
}
