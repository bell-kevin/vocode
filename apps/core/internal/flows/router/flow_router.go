package router

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/agent"
	"vocoding.net/vocode/v2/apps/core/internal/flows"
)

// FlowRouter maps a transcript to a route for the active flow. When Model is nil, it uses
// built-in stubs for tests / offline dev (still returns a structured search_query for rg routes).
type FlowRouter struct {
	Model agent.ModelClient
}

// NewFlowRouter returns a router. Pass model=nil for stub routing.
func NewFlowRouter(model agent.ModelClient) *FlowRouter {
	return &FlowRouter{Model: model}
}

// ClassifyFlow returns route classification and structured fields (e.g. search_query for rg).
func (r *FlowRouter) ClassifyFlow(ctx context.Context, in Context) (Result, error) {
	if r == nil {
		return Result{}, fmt.Errorf("router: nil FlowRouter")
	}
	if r.Model == nil {
		return classifyWithStub(in), nil
	}
	return classifyWithModel(ctx, r.Model, in)
}

func classifyWithModel(ctx context.Context, m agent.ModelClient, in Context) (Result, error) {
	userBytes, err := ClassifierUserJSON(in)
	if err != nil {
		return Result{}, err
	}
	schema := ClassifierResponseJSONSchema(in.Flow)
	content, err := m.Call(ctx, agent.CompletionRequest{
		System:     ClassifierSystem(in.Flow),
		User:       string(userBytes),
		JSONSchema: schema,
	})
	if err != nil {
		return Result{}, err
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return Result{}, ErrEmptyModelContent
	}
	var raw struct {
		Route            string `json:"route"`
		SearchQuery      string `json:"search_query"`
		SearchSymbolKind string `json:"search_symbol_kind"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return Result{}, fmt.Errorf("router: decode classifier json: %w", err)
	}
	res := Result{
		Flow:             in.Flow,
		Route:            strings.TrimSpace(raw.Route),
		SearchQuery:      strings.TrimSpace(raw.SearchQuery),
		SearchSymbolKind: strings.TrimSpace(strings.ToLower(raw.SearchSymbolKind)),
	}
	if err := res.Validate(); err != nil {
		return Result{}, err
	}
	return res, nil
}

func classifyWithStub(in Context) Result {
	t := strings.TrimSpace(strings.ToLower(in.Instruction))
	raw := strings.TrimSpace(in.Instruction)
	var res Result
	switch in.Flow {
	case flows.WorkspaceSelect:
		res = stubWorkspaceSelect(in, t, raw)
	case flows.SelectFile:
		res = stubSelectFile(t, raw)
	default:
		res = stubRoot(t, raw)
	}
	if err := res.Validate(); err != nil {
		return Result{Flow: in.Flow, Route: "irrelevant"}
	}
	return res
}

func stubRoot(t, raw string) Result {
	if t == "" {
		return Result{Flow: flows.Root, Route: "irrelevant"}
	}
	if strings.HasPrefix(t, "find file ") || strings.HasPrefix(t, "find files ") ||
		strings.HasPrefix(t, "open file ") || strings.HasPrefix(t, "show file ") ||
		strings.HasPrefix(t, "file named ") || strings.HasPrefix(t, "locate file ") {
		return Result{Flow: flows.Root, Route: "select_file", SearchQuery: raw}
	}
	if strings.HasPrefix(t, "find ") || strings.HasPrefix(t, "search ") || strings.HasPrefix(t, "where is ") || strings.HasPrefix(t, "locate ") {
		return Result{Flow: flows.Root, Route: "workspace_select", SearchQuery: raw}
	}
	if strings.HasSuffix(t, "?") || strings.HasPrefix(t, "what ") || strings.HasPrefix(t, "why ") || strings.HasPrefix(t, "how ") {
		return Result{Flow: flows.Root, Route: "question"}
	}
	if globalExitLike(t) {
		return Result{Flow: flows.Root, Route: "control"}
	}
	return Result{Flow: flows.Root, Route: "irrelevant"}
}

// stubImperativeEditLike matches common spoken edit intents (lowercased instruction).
func stubImperativeEditLike(t string) bool {
	for _, kw := range []string{
		"make ", "pass ", "change ", "add ", "remove ", "fix ", "rename ", "refactor",
		"update ", "edit ", "insert ", "delete ", "replace ",
	} {
		if strings.Contains(t, kw) {
			return true
		}
	}
	return false
}

func stubWorkspaceSelect(in Context, t, raw string) Result {
	if t == "" {
		return Result{Flow: flows.WorkspaceSelect, Route: "irrelevant"}
	}
	if globalExitLike(t) {
		return Result{Flow: flows.WorkspaceSelect, Route: "control"}
	}
	if strings.Contains(t, "find file ") || strings.Contains(t, "open file ") || strings.Contains(t, "show file ") {
		return Result{Flow: flows.WorkspaceSelect, Route: "select_file", SearchQuery: raw}
	}
	if strings.Contains(t, "rename") && strings.Contains(t, " to ") {
		return Result{Flow: flows.WorkspaceSelect, Route: "rename"}
	}
	if in.HasNonemptySelection && stubImperativeEditLike(t) {
		return Result{Flow: flows.WorkspaceSelect, Route: "edit"}
	}
	if strings.Contains(t, "find ") || strings.Contains(t, "search ") {
		return Result{Flow: flows.WorkspaceSelect, Route: "workspace_select", SearchQuery: raw}
	}
	if strings.Contains(t, "next") || strings.Contains(t, "forward") ||
		strings.Contains(t, "back") || strings.Contains(t, "prev") {
		return Result{Flow: flows.WorkspaceSelect, Route: "workspace_select_control"}
	}
	if strings.Contains(t, "delete") || strings.Contains(t, "remove") {
		return Result{Flow: flows.WorkspaceSelect, Route: "delete"}
	}
	for _, w := range []string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "1", "2", "3", "4", "5", "6", "7", "8", "9"} {
		if strings.Contains(t, w) {
			return Result{Flow: flows.WorkspaceSelect, Route: "workspace_select_control"}
		}
	}
	if strings.Contains(t, "edit") || strings.Contains(t, "change") {
		return Result{Flow: flows.WorkspaceSelect, Route: "edit"}
	}
	return Result{Flow: flows.WorkspaceSelect, Route: "irrelevant"}
}

func stubSelectFile(t, raw string) Result {
	if t == "" {
		return Result{Flow: flows.SelectFile, Route: "irrelevant"}
	}
	if globalExitLike(t) {
		return Result{Flow: flows.SelectFile, Route: "control"}
	}
	if strings.Contains(t, "next") || strings.Contains(t, "forward") ||
		strings.Contains(t, "back") || strings.Contains(t, "prev") {
		return Result{Flow: flows.SelectFile, Route: "file_select_control"}
	}
	for _, w := range []string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "1", "2", "3", "4", "5", "6", "7", "8", "9"} {
		if strings.Contains(t, w) {
			return Result{Flow: flows.SelectFile, Route: "file_select_control"}
		}
	}
	if strings.Contains(t, "delete") || strings.Contains(t, "remove") || strings.Contains(t, "trash") {
		return Result{Flow: flows.SelectFile, Route: "delete"}
	}
	if strings.Contains(t, "open") || strings.Contains(t, "show") || strings.Contains(t, "reveal") {
		return Result{Flow: flows.SelectFile, Route: "irrelevant"}
	}
	if strings.Contains(t, "rename") {
		return Result{Flow: flows.SelectFile, Route: "rename"}
	}
	if strings.Contains(t, "move") {
		return Result{Flow: flows.SelectFile, Route: "move"}
	}
	if strings.Contains(t, "create") || strings.Contains(t, "new file") || strings.Contains(t, "new folder") {
		return Result{Flow: flows.SelectFile, Route: "create"}
	}
	return Result{Flow: flows.SelectFile, Route: "irrelevant"}
}

func globalExitLike(t string) bool {
	t = strings.TrimSpace(strings.ToLower(t))
	for _, w := range []string{"cancel", "exit", "close", "stop", "done", "quit", "leave", "abort"} {
		if strings.Contains(t, w) {
			return true
		}
	}
	return false
}
