package executor

import (
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch"
	"vocoding.net/vocode/v2/apps/daemon/internal/symbols"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Executor runs one voice.transcript through the agent: an iterative loop of [agent.Agent.NextTurn],
// optional request_context rounds, batched executable intents per turn, retries,
// and dispatch via [dispatch.Handler.Handle] (control vs executable).
type Executor struct {
	agent                    *agent.Agent
	intentHandler            *dispatch.Handler
	symbols                  symbols.Resolver
	maxAgentTurns            int
	maxIntentRetries         int
	maxContextRounds         int
	maxContextBytes          int
	maxConsecutiveContextReq int
	maxIntentsPerBatch       int
}

// Options configures caps and optional symbol resolution for [Executor].
type Options struct {
	MaxAgentTurns            int
	MaxIntentRetries         int
	MaxContextRounds         int
	MaxContextBytes          int
	MaxConsecutiveContextReq int
	// MaxIntentsPerBatch caps TurnIntents length; 0 or negative means no cap.
	MaxIntentsPerBatch int
	Symbols            symbols.Resolver
}

// New constructs an [Executor].
// MaxIntentsPerBatch: 0 means no cap; unset env defaults to 16 in [transcript.NewService].
func New(a *agent.Agent, h *dispatch.Handler, opts Options) *Executor {
	return &Executor{
		agent:                    a,
		intentHandler:            h,
		symbols:                  opts.Symbols,
		maxAgentTurns:            opts.MaxAgentTurns,
		maxIntentRetries:         opts.MaxIntentRetries,
		maxContextRounds:         opts.MaxContextRounds,
		maxContextBytes:          opts.MaxContextBytes,
		maxConsecutiveContextReq: opts.MaxConsecutiveContextReq,
		maxIntentsPerBatch:       opts.MaxIntentsPerBatch,
	}
}

// Execute runs the agent loop until done, caps, or failure.
func (e *Executor) Execute(
	params protocol.VoiceTranscriptParams,
	gatheredIn agentcontext.Gathered,
	intentApplyHistory []agentcontext.IntentApplyRecord,
	extSucceeded []intents.Intent,
	extFailed []agentcontext.FailedIntent,
	extSkipped []intents.Intent,
) (protocol.VoiceTranscriptResult, agentcontext.Gathered, *agentcontext.DirectiveApplyBatch, bool) {
	text := strings.TrimSpace(params.Text)
	if text == "" {
		return protocol.VoiceTranscriptResult{}, gatheredIn, nil, false
	}
	maxLoopIters := e.maxAgentTurns
	if maxLoopIters <= 0 {
		maxLoopIters = 8
	}
	maxRetries := e.maxIntentRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	hostCursor := resolveHostCursorSymbol(e.symbols, params)
	st := &agentLoopState{
		gathered:   agentcontext.SeedGatheredActiveFile(gatheredIn, params.ActiveFile),
		completed:  make([]intents.Intent, 0, maxLoopIters),
		directives: make([]protocol.VoiceTranscriptDirective, 0, maxLoopIters*4),
		maxRetries: maxRetries,
	}

	brokeOK := false
	for range maxLoopIters {
		adv, failRes, abort := e.runOneAgentLoopIteration(
			params, text, hostCursor, intentApplyHistory, extSucceeded, extFailed, extSkipped, st)
		if abort {
			return failRes, st.gathered, nil, true
		}
		if adv == advanceBreakLoop {
			brokeOK = true
			break
		}
	}

	if !brokeOK {
		return protocol.VoiceTranscriptResult{Success: false}, st.gathered, nil, true
	}
	return finalizeExecute(st)
}
