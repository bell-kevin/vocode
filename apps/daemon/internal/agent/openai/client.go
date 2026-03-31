// Package openai implements [agent.ModelClient] via OpenAI Chat Completions (JSON object mode).
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agent/prompt"
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
)

// Client calls the OpenAI HTTP API.
type Client struct {
	HTTPClient *http.Client
	APIKey     string
	BaseURL    string
	Model      string
}

// NewFromEnv builds a client from OPENAI_API_KEY (required), VOCODE_OPENAI_BASE_URL (optional),
// and VOCODE_OPENAI_MODEL (required, supplied by the VS Code extension).
func NewFromEnv() (*Client, error) {
	key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if key == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set")
	}
	base := strings.TrimSpace(os.Getenv("VOCODE_OPENAI_BASE_URL"))
	base = strings.TrimSuffix(base, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	model := strings.TrimSpace(os.Getenv("VOCODE_OPENAI_MODEL"))
	if model == "" {
		return nil, fmt.Errorf("VOCODE_OPENAI_MODEL is not set")
	}
	return &Client{
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
		APIKey:     key,
		BaseURL:    base,
		Model:      model,
	}, nil
}

func (c *Client) ClassifyTranscript(ctx context.Context, in agentcontext.TranscriptClassifierContext) (agent.TranscriptClassifierResult, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return agent.TranscriptClassifierResult{}, fmt.Errorf("openai: missing API key")
	}
	userBytes, err := prompt.TranscriptClassifierUserJSON(in)
	if err != nil {
		return agent.TranscriptClassifierResult{}, fmt.Errorf("openai: prompt: %w", err)
	}
	temp := 0.0
	body := chatCompletionsRequest{
		Model:       c.Model,
		Temperature: &temp,
		Messages: []chatMessage{
			{Role: "system", Content: prompt.TranscriptClassifierSystem()},
			{Role: "user", Content: string(userBytes)},
		},
		ResponseFormat: chatResponseFormatTranscriptClassifier(),
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return agent.TranscriptClassifierResult{}, err
	}
	url := c.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return agent.TranscriptClassifierResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return agent.TranscriptClassifierResult{}, fmt.Errorf("openai: request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return agent.TranscriptClassifierResult{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return agent.TranscriptClassifierResult{}, fmt.Errorf("openai: HTTP %s: %s", resp.Status, truncateForErr(respBody, 512))
	}
	var parsed chatCompletionsResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return agent.TranscriptClassifierResult{}, fmt.Errorf("openai: decode response: %w", err)
	}
	if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
		return agent.TranscriptClassifierResult{}, fmt.Errorf("openai: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return agent.TranscriptClassifierResult{}, fmt.Errorf("openai: empty choices")
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return agent.TranscriptClassifierResult{}, fmt.Errorf("openai: empty message content")
	}
	var out struct {
		Kind       string `json:"kind"`
		SearchQuery string `json:"searchQuery"`
		AnswerText string `json:"answerText"`
	}
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return agent.TranscriptClassifierResult{}, fmt.Errorf("openai: decode classifier: %w", err)
	}
	res := agent.TranscriptClassifierResult{
		Kind:       agent.TranscriptKind(strings.TrimSpace(out.Kind)),
		SearchQuery: out.SearchQuery,
		AnswerText: out.AnswerText,
	}
	if err := res.Validate(); err != nil {
		return agent.TranscriptClassifierResult{}, err
	}
	return res, nil
}

// ScopedEdit implements [agent.ModelClient].
func (c *Client) ScopedEdit(ctx context.Context, in agentcontext.ScopedEditContext) (agent.ScopedEditResult, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return agent.ScopedEditResult{}, fmt.Errorf("openai: missing API key")
	}
	userBytes, err := prompt.ScopedEditUserJSON(in)
	if err != nil {
		return agent.ScopedEditResult{}, fmt.Errorf("openai: prompt: %w", err)
	}
	temp := 0.2
	body := chatCompletionsRequest{
		Model:       c.Model,
		Temperature: &temp,
		Messages: []chatMessage{
			{Role: "system", Content: prompt.ScopedEditSystem()},
			{Role: "user", Content: string(userBytes)},
		},
		ResponseFormat: chatResponseFormatScopedEdit(),
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return agent.ScopedEditResult{}, err
	}
	url := c.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return agent.ScopedEditResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return agent.ScopedEditResult{}, fmt.Errorf("openai: request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return agent.ScopedEditResult{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return agent.ScopedEditResult{}, fmt.Errorf("openai: HTTP %s: %s", resp.Status, truncateForErr(respBody, 512))
	}
	var parsed chatCompletionsResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return agent.ScopedEditResult{}, fmt.Errorf("openai: decode response: %w", err)
	}
	if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
		return agent.ScopedEditResult{}, fmt.Errorf("openai: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return agent.ScopedEditResult{}, fmt.Errorf("openai: empty choices")
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return agent.ScopedEditResult{}, fmt.Errorf("openai: empty message content")
	}
	var out struct {
		ReplacementText string `json:"replacementText"`
	}
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return agent.ScopedEditResult{}, fmt.Errorf("openai: decode scoped edit: %w", err)
	}
	res := agent.ScopedEditResult{ReplacementText: out.ReplacementText}
	if err := res.Validate(); err != nil {
		return agent.ScopedEditResult{}, err
	}
	return res, nil
}

func (c *Client) ScopeIntent(ctx context.Context, in agentcontext.ScopeIntentContext) (agent.ScopeIntentResult, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return agent.ScopeIntentResult{}, fmt.Errorf("openai: missing API key")
	}
	userBytes, err := prompt.ScopeIntentUserJSON(in)
	if err != nil {
		return agent.ScopeIntentResult{}, fmt.Errorf("openai: prompt: %w", err)
	}
	temp := 0.0
	body := chatCompletionsRequest{
		Model:       c.Model,
		Temperature: &temp,
		Messages: []chatMessage{
			{Role: "system", Content: prompt.ScopeIntentSystem()},
			{Role: "user", Content: string(userBytes)},
		},
		ResponseFormat: chatResponseFormatScopeIntent(),
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return agent.ScopeIntentResult{}, err
	}
	url := c.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return agent.ScopeIntentResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return agent.ScopeIntentResult{}, fmt.Errorf("openai: request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return agent.ScopeIntentResult{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return agent.ScopeIntentResult{}, fmt.Errorf("openai: HTTP %s: %s", resp.Status, truncateForErr(respBody, 512))
	}
	var parsed chatCompletionsResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return agent.ScopeIntentResult{}, fmt.Errorf("openai: decode response: %w", err)
	}
	if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
		return agent.ScopeIntentResult{}, fmt.Errorf("openai: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return agent.ScopeIntentResult{}, fmt.Errorf("openai: empty choices")
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return agent.ScopeIntentResult{}, fmt.Errorf("openai: empty message content")
	}
	var out struct {
		ScopeKind        string `json:"scopeKind"`
		SymbolName       string `json:"symbolName"`
		ClarifyQuestion  string `json:"clarifyQuestion"`
	}
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return agent.ScopeIntentResult{}, fmt.Errorf("openai: decode scope intent: %w", err)
	}
	res := agent.ScopeIntentResult{
		ScopeKind:        agent.ScopeKind(strings.TrimSpace(out.ScopeKind)),
		SymbolName:       out.SymbolName,
		ClarifyQuestion:  out.ClarifyQuestion,
	}
	if err := res.Validate(); err != nil {
		return agent.ScopeIntentResult{}, err
	}
	return res, nil
}

type chatCompletionsRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	Temperature    *float64        `json:"temperature,omitempty"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionsResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func truncateForErr(b []byte, max int) string {
	s := string(b)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
