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
// built-in heuristics (tests / offline dev).
type FlowRouter struct {
	Model agent.ModelClient
}

// NewFlowRouter returns a router. Pass model=nil for heuristic-only routing.
func NewFlowRouter(model agent.ModelClient) *FlowRouter {
	return &FlowRouter{Model: model}
}

// ClassifyFlow returns the route id for the given flow context.
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
		Route string `json:"route"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return Result{}, fmt.Errorf("router: decode classifier json: %w", err)
	}
	res := Result{
		Flow:  in.Flow,
		Route: strings.TrimSpace(raw.Route),
	}
	if err := res.Validate(); err != nil {
		return Result{}, err
	}
	return res, nil
}

func classifyWithStub(in Context) Result {
	t := strings.TrimSpace(strings.ToLower(in.Instruction))
	switch in.Flow {
	case flows.Select:
		return stubSelect(t)
	case flows.SelectFile:
		return stubSelectFile(t)
	default:
		return stubRoot(t)
	}
}

func stubRoot(t string) Result {
	if t == "" {
		return Result{Flow: flows.Root, Route: "irrelevant"}
	}
	if strings.HasPrefix(t, "find file ") || strings.HasPrefix(t, "find files ") ||
		strings.HasPrefix(t, "open file ") || strings.HasPrefix(t, "show file ") ||
		strings.HasPrefix(t, "file named ") || strings.HasPrefix(t, "locate file ") {
		return Result{Flow: flows.Root, Route: "select_file"}
	}
	if strings.HasPrefix(t, "find ") || strings.HasPrefix(t, "search ") || strings.HasPrefix(t, "where is ") || strings.HasPrefix(t, "locate ") {
		return Result{Flow: flows.Root, Route: "select"}
	}
	if strings.HasSuffix(t, "?") || strings.HasPrefix(t, "what ") || strings.HasPrefix(t, "why ") || strings.HasPrefix(t, "how ") {
		return Result{Flow: flows.Root, Route: "question"}
	}
	if globalExitLike(t) {
		return Result{Flow: flows.Root, Route: "control"}
	}
	return Result{Flow: flows.Root, Route: "irrelevant"}
}

func stubSelect(t string) Result {
	if t == "" {
		return Result{Flow: flows.Select, Route: "irrelevant"}
	}
	if globalExitLike(t) {
		return Result{Flow: flows.Select, Route: "control"}
	}
	if strings.Contains(t, "find file ") || strings.Contains(t, "open file ") || strings.Contains(t, "show file ") {
		return Result{Flow: flows.Select, Route: "select_file"}
	}
	if strings.Contains(t, "find ") || strings.Contains(t, "search ") {
		return Result{Flow: flows.Select, Route: "select"}
	}
	if strings.Contains(t, "next") || strings.Contains(t, "forward") ||
		strings.Contains(t, "back") || strings.Contains(t, "prev") {
		return Result{Flow: flows.Select, Route: "select_control"}
	}
	if strings.Contains(t, "delete") || strings.Contains(t, "remove") {
		return Result{Flow: flows.Select, Route: "delete"}
	}
	for _, w := range []string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "1", "2", "3", "4", "5", "6", "7", "8", "9"} {
		if strings.Contains(t, w) {
			return Result{Flow: flows.Select, Route: "select_control"}
		}
	}
	if strings.Contains(t, "edit") || strings.Contains(t, "change") {
		return Result{Flow: flows.Select, Route: "edit"}
	}
	return Result{Flow: flows.Select, Route: "irrelevant"}
}

func stubSelectFile(t string) Result {
	if t == "" {
		return Result{Flow: flows.SelectFile, Route: "irrelevant"}
	}
	if globalExitLike(t) {
		return Result{Flow: flows.SelectFile, Route: "control"}
	}
	if strings.Contains(t, "next") || strings.Contains(t, "forward") ||
		strings.Contains(t, "back") || strings.Contains(t, "prev") {
		return Result{Flow: flows.SelectFile, Route: "select_file_control"}
	}
	for _, w := range []string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "1", "2", "3", "4", "5", "6", "7", "8", "9"} {
		if strings.Contains(t, w) {
			return Result{Flow: flows.SelectFile, Route: "select_file_control"}
		}
	}
	if strings.Contains(t, "delete") || strings.Contains(t, "remove") || strings.Contains(t, "trash") {
		return Result{Flow: flows.SelectFile, Route: "delete"}
	}
	if strings.Contains(t, "open") || strings.Contains(t, "show") || strings.Contains(t, "reveal") {
		return Result{Flow: flows.SelectFile, Route: "open"}
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
