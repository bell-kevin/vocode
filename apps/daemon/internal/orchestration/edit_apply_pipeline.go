package orchestration

import (
	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/edits"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// EditApplyPipeline maps a concrete [agent.EditIntent] plus file snapshot params
// to protocol edit actions (the agent is the planner; there is no separate
// planner stage here).
type EditApplyPipeline struct {
	edits editActionBuilder
}

type editActionBuilder interface {
	BuildActions(params protocol.EditApplyParams, intent agent.EditIntent) ([]protocol.EditAction, *protocol.EditFailure)
}

func NewEditApplyPipeline(editService *edits.Service) *EditApplyPipeline {
	return &EditApplyPipeline{edits: editService}
}

// ApplyWithIntent builds actions from an [agent.EditIntent] (e.g. model output).
func (p *EditApplyPipeline) ApplyWithIntent(params protocol.EditApplyParams, intent agent.EditIntent) (protocol.EditApplyResult, error) {
	if err := agent.ValidateEditIntent(intent); err != nil {
		f := protocol.EditFailure{Code: "unsupported_instruction", Message: err.Error()}
		result := protocol.NewEditApplyFailure(f)
		return result, result.Validate()
	}
	return p.applyWithIntent(params, intent)
}

func (p *EditApplyPipeline) applyWithIntent(params protocol.EditApplyParams, intent agent.EditIntent) (protocol.EditApplyResult, error) {
	actions, failure := p.edits.BuildActions(params, intent)
	if failure != nil {
		if failure.Code == "no_change_needed" {
			result := protocol.NewEditApplyNoop(failure.Message)
			return result, result.Validate()
		}
		result := protocol.NewEditApplyFailure(*failure)
		return result, result.Validate()
	}

	result := protocol.NewEditApplySuccess(actions)
	return result, result.Validate()
}
