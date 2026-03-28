package intentloop

import (
	"context"
	"fmt"
	"os"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/edits"
	"vocoding.net/vocode/v2/apps/daemon/internal/intent"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type ContextProvider interface {
	Fulfill(params protocol.VoiceTranscriptParams, in agent.PlanningContext, req *intent.RequestContextIntent) (agent.PlanningContext, error)
}

type Runner struct {
	agent                    *agent.Agent
	intents                  *intents.Handler
	contextProvider          ContextProvider
	maxPlannerTurns          int
	maxIntentRetries         int
	maxContextRounds         int
	maxContextBytes          int
	maxConsecutiveContextReq int
}

type Options struct {
	MaxPlannerTurns          int
	MaxIntentRetries         int
	MaxContextRounds         int
	MaxContextBytes          int
	MaxConsecutiveContextReq int
}

func NewRunner(a *agent.Agent, h *intents.Handler, cp ContextProvider, opts Options) *Runner {
	return &Runner{
		agent:                    a,
		intents:                  h,
		contextProvider:          cp,
		maxPlannerTurns:          opts.MaxPlannerTurns,
		maxIntentRetries:         opts.MaxIntentRetries,
		maxContextRounds:         opts.MaxContextRounds,
		maxContextBytes:          opts.MaxContextBytes,
		maxConsecutiveContextReq: opts.MaxConsecutiveContextReq,
	}
}

func (r *Runner) Execute(params protocol.VoiceTranscriptParams) (protocol.VoiceTranscriptResult, bool) {
	text := strings.TrimSpace(params.Text)
	if text == "" {
		return protocol.VoiceTranscriptResult{}, false
	}
	maxTurns := r.maxPlannerTurns
	if maxTurns <= 0 {
		maxTurns = 8
	}
	maxRetries := r.maxIntentRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	turnCtx := agent.PlanningContext{}
	completed := make([]intent.NextIntent, 0, maxTurns)
	contextRounds := 0
	consecutiveContextReq := 0
	editCounter := 0
	directives := make([]protocol.VoiceTranscriptDirective, 0, maxTurns)
	trace := make([]string, 0, maxTurns*2)
	stopPlanning := false

	for i := 0; i < maxTurns; i++ {
		turn := i + 1
		next, err := r.agent.NextIntent(context.Background(), agent.ModelInput{
			Transcript:       text,
			CompletedActions: append([]intent.NextIntent(nil), completed...),
			Context:          turnCtx,
		})
		if err != nil {
			trace = appendTurnTrace(trace, turn, "model_error")
			return protocol.VoiceTranscriptResult{Accepted: false}, true
		}
		if err := intent.ValidateNextIntent(next); err != nil {
			trace = appendTurnTrace(trace, turn, "invalid_next_intent")
			return protocol.VoiceTranscriptResult{Accepted: false}, true
		}
		trace = appendTurnTrace(trace, turn, "intent:"+string(next.Kind))
		if next.Kind == intent.NextIntentKindDone {
			break
		}
		if next.Kind == intent.NextIntentKindRequestContext {
			contextRounds++
			consecutiveContextReq++
			if r.maxContextRounds > 0 && contextRounds > r.maxContextRounds {
				trace = appendTurnTrace(trace, turn, "context:cap:max_rounds")
				return protocol.VoiceTranscriptResult{Accepted: false}, true
			}
			if r.maxConsecutiveContextReq > 0 && consecutiveContextReq > r.maxConsecutiveContextReq {
				trace = appendTurnTrace(trace, turn, "context:cap:consecutive_requests")
				return protocol.VoiceTranscriptResult{Accepted: false}, true
			}
			if r.contextProvider == nil {
				return protocol.VoiceTranscriptResult{Accepted: false}, true
			}
			updated, err := r.contextProvider.Fulfill(params, turnCtx, next.RequestContext)
			if err != nil {
				trace = appendTurnTrace(trace, turn, "context:fulfill_error")
				return protocol.VoiceTranscriptResult{Accepted: false}, true
			}
			if r.maxContextBytes > 0 && estimatePlanningContextBytes(updated) > r.maxContextBytes {
				trace = appendTurnTrace(trace, turn, "context:cap:byte_budget")
				return protocol.VoiceTranscriptResult{Accepted: false}, true
			}
			trace = appendTurnTrace(trace, turn, "context:fulfilled")
			turnCtx = updated
			completed = append(completed, next)
			continue
		}
		consecutiveContextReq = 0
		editCtx, planErr := buildEditExecutionContext(params, next)
		if planErr != "" {
			trace = appendTurnTrace(trace, turn, "pre_execute_error")
			if maxRetries > 0 {
				trace = appendTurnTrace(trace, turn, "retry:pre_execute")
				turnCtx = appendPlanningNote(turnCtx, fmt.Sprintf("daemon rejected %q intent before execution: %s; retry with corrected intent", next.Kind, planErr))
				maxRetries--
				continue
			}
			return protocol.VoiceTranscriptResult{Accepted: false}, true
		}
		st, err := r.intents.DispatchIntent(next, editCtx)
		if err != nil {
			trace = appendTurnTrace(trace, turn, "execute_error")
			if maxRetries > 0 {
				trace = appendTurnTrace(trace, turn, "retry:execute")
				turnCtx = appendPlanningNote(turnCtx, fmt.Sprintf("daemon execution failed for %q intent: %v; retry with corrected intent", next.Kind, err))
				maxRetries--
				continue
			}
			return protocol.VoiceTranscriptResult{Accepted: false}, true
		}
		maxRetries = r.maxIntentRetries
		if maxRetries < 0 {
			maxRetries = 0
		}
		switch {
		case st.EditDirective != nil:
			if st.EditDirective.Kind == "success" {
				for i := range st.EditDirective.Actions {
					if st.EditDirective.Actions[i].EditId == "" {
						st.EditDirective.Actions[i].EditId = fmt.Sprintf("edit-%d", editCounter)
						editCounter++
					}
				}
			}
			directives = append(directives, protocol.VoiceTranscriptDirective{Kind: "edit", EditDirective: st.EditDirective})
			trace = appendTurnTrace(trace, turn, "result:edit:"+st.EditDirective.Kind)
			completed = append(completed, next)
		case st.CommandDirective != nil:
			directives = append(directives, protocol.VoiceTranscriptDirective{Kind: "command", CommandDirective: st.CommandDirective})
			trace = appendTurnTrace(trace, turn, "result:command")
			completed = append(completed, next)
		case st.NavigationDirective != nil:
			directives = append(directives, protocol.VoiceTranscriptDirective{Kind: "navigate", NavigationDirective: st.NavigationDirective})
			trace = appendTurnTrace(trace, turn, "result:navigate")
			completed = append(completed, next)
		case st.UndoDirective != nil:
			directives = append(directives, protocol.VoiceTranscriptDirective{Kind: "undo", UndoDirective: st.UndoDirective})
			trace = appendTurnTrace(trace, turn, "result:undo:"+st.UndoDirective.Scope)
			completed = append(completed, next)
		}
		if stopPlanning {
			break
		}
	}
	if len(completed) >= maxTurns {
		trace = appendTrace(trace, "cap:max_turns")
		return protocol.VoiceTranscriptResult{Accepted: false}, true
	}
	result := protocol.VoiceTranscriptResult{Accepted: true, Directives: directives}
	if err := result.Validate(); err != nil {
		return protocol.VoiceTranscriptResult{Accepted: false}, true
	}
	return result, true
}

func buildEditExecutionContext(params protocol.VoiceTranscriptParams, next intent.NextIntent) (edits.EditExecutionContext, string) {
	if next.Kind == intent.NextIntentKindUndo {
		return edits.EditExecutionContext{}, ""
	}
	active := strings.TrimSpace(params.ActiveFile)
	workspaceRoot := strings.TrimSpace(params.WorkspaceRoot)
	if next.Kind == intent.NextIntentKindEdit && active == "" {
		return edits.EditExecutionContext{}, "activeFile is required when the next intent is an edit"
	}
	if next.Kind == intent.NextIntentKindEdit && workspaceRoot == "" {
		return edits.EditExecutionContext{}, "workspaceRoot is required when the next intent is an edit"
	}
	fileText := ""
	if active != "" {
		b, err := os.ReadFile(active)
		if err != nil {
			return edits.EditExecutionContext{}, fmt.Sprintf("read active file: %v", err)
		}
		fileText = string(b)
	}
	return edits.EditExecutionContext{
		Instruction:   params.Text,
		ActiveFile:    params.ActiveFile,
		FileText:      fileText,
		WorkspaceRoot: workspaceRoot,
	}, ""
}

func estimatePlanningContextBytes(c agent.PlanningContext) int {
	total := 0
	for _, sref := range c.Symbols {
		total += len(sref.ID) + len(sref.Name) + len(sref.Path) + len(sref.Kind) + 16
	}
	for _, ex := range c.Excerpts {
		total += len(ex.Path) + len(ex.Content)
	}
	for _, note := range c.Notes {
		total += len(note)
	}
	return total
}

func appendTurnTrace(trace []string, turn int, msg string) []string {
	return appendTrace(trace, fmt.Sprintf("t%d:%s", turn, msg))
}

func appendTrace(trace []string, msg string) []string {
	const maxTraceEntries = 16
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return trace
	}
	trace = append(trace, msg)
	if len(trace) > maxTraceEntries {
		trace = trace[len(trace)-maxTraceEntries:]
	}
	return trace
}

func appendPlanningNote(c agent.PlanningContext, note string) agent.PlanningContext {
	const maxNotes = 8
	note = strings.TrimSpace(note)
	if note == "" {
		return c
	}
	c.Notes = append(c.Notes, note)
	if len(c.Notes) > maxNotes {
		c.Notes = c.Notes[len(c.Notes)-maxNotes:]
	}
	return c
}
