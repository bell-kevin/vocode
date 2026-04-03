package rpc

import (
	"encoding/json"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// ExtensionHost is the JSON-RPC surface the VS Code extension implements for vocode-cored
// (outbound requests from core on the stdio session).
type ExtensionHost interface {
	ApplyDirectives(protocol.HostApplyParams) (protocol.HostApplyResult, error)
	ReadHostFile(path string) (string, error)
	GetDocumentSymbols(protocol.HostGetDocumentSymbolsParams) (protocol.HostGetDocumentSymbolsResult, error)
	WorkspaceSymbolSearch(protocol.HostWorkspaceSymbolSearchParams) (protocol.HostWorkspaceSymbolSearchResult, error)
}

// ReadHostFile reads UTF-8 text via `host.readFile`.
func (s *Server) ReadHostFile(path string) (string, error) {
	raw, err := s.Request("host.readFile", protocol.HostReadFileParams{Path: path})
	if err != nil {
		return "", err
	}
	var out protocol.HostReadFileResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	return out.Text, nil
}

// GetDocumentSymbols runs `host.getDocumentSymbols`.
func (s *Server) GetDocumentSymbols(params protocol.HostGetDocumentSymbolsParams) (protocol.HostGetDocumentSymbolsResult, error) {
	raw, err := s.Request("host.getDocumentSymbols", params)
	if err != nil {
		return protocol.HostGetDocumentSymbolsResult{}, err
	}
	var out protocol.HostGetDocumentSymbolsResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return protocol.HostGetDocumentSymbolsResult{}, err
	}
	return out, nil
}

// WorkspaceSymbolSearch runs `host.workspaceSymbolSearch`.
func (s *Server) WorkspaceSymbolSearch(params protocol.HostWorkspaceSymbolSearchParams) (protocol.HostWorkspaceSymbolSearchResult, error) {
	raw, err := s.Request("host.workspaceSymbolSearch", params)
	if err != nil {
		return protocol.HostWorkspaceSymbolSearchResult{}, err
	}
	var out protocol.HostWorkspaceSymbolSearchResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return protocol.HostWorkspaceSymbolSearchResult{}, err
	}
	return out, nil
}
