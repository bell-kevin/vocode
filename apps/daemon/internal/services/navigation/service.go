package navigation

import (
	"vocoding.net/vocode/v2/apps/daemon/internal/intent"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Service validates navigation intents and returns protocol navigation directives.
type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) DispatchIntent(nav intent.NavigationIntent) (protocol.NavigationDirective, error) {
	if err := intent.ValidateNavigationIntent(nav); err != nil {
		return protocol.NavigationDirective{}, err
	}
	action := protocol.NavigationAction{Kind: string(nav.Kind)}
	if nav.OpenFile != nil {
		action.OpenFile = &struct {
			Path string `json:"path"`
		}{Path: nav.OpenFile.Path}
	}
	if nav.RevealSymbol != nil {
		action.RevealSymbol = &struct {
			Path       string `json:"path,omitempty"`
			SymbolName string `json:"symbolName"`
			SymbolKind string `json:"symbolKind,omitempty"`
		}{Path: nav.RevealSymbol.Path, SymbolName: nav.RevealSymbol.SymbolName, SymbolKind: nav.RevealSymbol.SymbolKind}
	}
	if nav.MoveCursor != nil {
		action.MoveCursor = &struct {
			Target struct {
				Path string `json:"path,omitempty"`
				Line int64  `json:"line"`
				Char int64  `json:"char"`
			} `json:"target"`
		}{}
		action.MoveCursor.Target.Path = nav.MoveCursor.Target.Path
		action.MoveCursor.Target.Line = int64(nav.MoveCursor.Target.Line)
		action.MoveCursor.Target.Char = int64(nav.MoveCursor.Target.Char)
	}
	if nav.SelectRange != nil {
		action.SelectRange = &struct {
			Target struct {
				Path      string `json:"path,omitempty"`
				StartLine int64  `json:"startLine"`
				StartChar int64  `json:"startChar"`
				EndLine   int64  `json:"endLine"`
				EndChar   int64  `json:"endChar"`
			} `json:"target"`
		}{}
		action.SelectRange.Target.Path = nav.SelectRange.Target.Path
		action.SelectRange.Target.StartLine = int64(nav.SelectRange.Target.StartLine)
		action.SelectRange.Target.StartChar = int64(nav.SelectRange.Target.StartChar)
		action.SelectRange.Target.EndLine = int64(nav.SelectRange.Target.EndLine)
		action.SelectRange.Target.EndChar = int64(nav.SelectRange.Target.EndChar)
	}
	if nav.RevealEdit != nil {
		action.RevealEdit = &struct {
			EditId string `json:"editId"`
		}{EditId: nav.RevealEdit.EditID}
	}
	return protocol.NewNavigationDirectiveSuccess(action), nil
}
