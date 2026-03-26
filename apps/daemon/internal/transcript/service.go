package transcript

import (
	"context"
	"fmt"
	"os"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/actionplan"
	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/actionplan/dispatch"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// TranscriptService adapts the voice.transcript RPC to the agent runtime and
// action-plan execution (edits via structured results for the extension;
// commands on the daemon).
type TranscriptService struct {
	agent    *agent.Agent
	dispatch *dispatch.Dispatcher
}

func NewService(
	agentRuntime *agent.Agent,
	dispatch *dispatch.Dispatcher,
) *TranscriptService {
	return &TranscriptService{agent: agentRuntime, dispatch: dispatch}
}

func (s *TranscriptService) AcceptTranscript(
	params protocol.VoiceTranscriptParams,
) (protocol.VoiceTranscriptResult, bool) {
	r := s.agent.HandleTranscript(context.Background(), params)
	if !r.Valid {
		return protocol.VoiceTranscriptResult{}, false
	}
	if r.Err != nil {
		return protocol.VoiceTranscriptResult{
			Accepted:  true,
			PlanError: r.Err.Error(),
		}, true
	}
	if r.Plan == nil {
		return protocol.VoiceTranscriptResult{
			Accepted:  true,
			PlanError: "no plan returned",
		}, true
	}

	editParams, planErr := buildEditApplyParams(params, r.Plan)
	if planErr != "" {
		return protocol.VoiceTranscriptResult{
			Accepted:  true,
			PlanError: planErr,
		}, true
	}

	execResult, err := s.dispatch.Execute(*r.Plan, editParams)
	if err != nil {
		return protocol.VoiceTranscriptResult{
			Accepted:  true,
			PlanError: err.Error(),
		}, true
	}

	steps := make([]protocol.VoiceTranscriptStepResult, 0, len(execResult.Steps))
	for _, st := range execResult.Steps {
		switch {
		case st.EditResult != nil:
			steps = append(steps, protocol.VoiceTranscriptStepResult{
				Kind:       "edit",
				EditResult: st.EditResult,
			})
		case st.CommandResult != nil:
			steps = append(steps, protocol.VoiceTranscriptStepResult{
				Kind:          "run_command",
				CommandResult: st.CommandResult,
			})
		}
	}

	result := protocol.VoiceTranscriptResult{
		Accepted: true,
		Steps:    steps,
	}
	if err := result.Validate(); err != nil {
		return protocol.VoiceTranscriptResult{
			Accepted:  true,
			PlanError: err.Error(),
		}, true
	}
	return result, true
}

func planHasEditStep(p *actionplan.ActionPlan) bool {
	for _, s := range p.Steps {
		if s.Kind == actionplan.StepKindEdit {
			return true
		}
	}
	return false
}

// buildEditApplyParams loads file text on the daemon when activeFile is set.
// Unsaved editor buffers are not visible until workspace indexing supplies them.
func buildEditApplyParams(params protocol.VoiceTranscriptParams, plan *actionplan.ActionPlan) (protocol.EditApplyParams, string) {
	active := strings.TrimSpace(params.ActiveFile)
	if planHasEditStep(plan) && active == "" {
		return protocol.EditApplyParams{}, "activeFile is required when the plan includes edit steps"
	}
	fileText := ""
	if active != "" {
		b, err := os.ReadFile(active)
		if err != nil {
			return protocol.EditApplyParams{}, fmt.Sprintf("read active file: %v", err)
		}
		fileText = string(b)
	}
	return protocol.EditApplyParams{
		Instruction: params.Text,
		ActiveFile:  params.ActiveFile,
		FileText:    fileText,
	}, ""
}
