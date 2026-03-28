package edits

import (
	"vocoding.net/vocode/v2/apps/daemon/internal/intent"
	"vocoding.net/vocode/v2/apps/daemon/internal/symbols"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Engine holds edit action building state (symbol resolution, etc.).
type Engine struct {
	actionBuilder *ActionBuilder
}

func NewEngine() *Engine {
	return &Engine{actionBuilder: NewActionBuilder()}
}

func NewEngineWithResolver(resolver symbols.Resolver) *Engine {
	return &Engine{actionBuilder: NewActionBuilderWithResolver(resolver)}
}

func (e *Engine) BuildActions(ctx EditExecutionContext, editIntent intent.EditIntent) ([]protocol.EditAction, *EditBuildFailure) {
	return e.actionBuilder.BuildActions(ctx, editIntent)
}
