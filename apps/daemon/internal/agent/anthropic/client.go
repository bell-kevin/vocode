// Package anthropic implements [agent.ModelClient] via the Anthropic Messages API (JSON in assistant text).
package anthropic

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

const anthropicAPIVersion = "2023-06-01"

// Client calls the Anthropic HTTP API.
type Client struct {
	HTTPClient *http.Client
	APIKey     string
	BaseURL    string
	Model      string
}

// NewFromEnv requires ANTHROPIC_API_KEY; optional VOCODE_ANTHROPIC_MODEL (default claude-3-5-haiku-20241022),
// optional VOCODE_ANTHROPIC_BASE_URL (default https://api.anthropic.com/v1).
func NewFromEnv() (*Client, error) {
	key := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	if key == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is not set")
	}
	base := strings.TrimSpace(os.Getenv("VOCODE_ANTHROPIC_BASE_URL"))
	base = strings.TrimSuffix(base, "/")
	if base == "" {
		base = "https://api.anthropic.com/v1"
	}
	model := strings.TrimSpace(os.Getenv("VOCODE_ANTHROPIC_MODEL"))
	if model == "" {
		return nil, fmt.Errorf("VOCODE_ANTHROPIC_MODEL is not set")
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
		return agent.TranscriptClassifierResult{}, fmt.Errorf("anthropic: missing API key")
	}
	userBytes, err := prompt.TranscriptClassifierUserJSON(in)
	if err != nil {
		return agent.TranscriptClassifierResult{}, fmt.Errorf("anthropic: prompt: %w", err)
	}
	body := messagesRequest{
		Model:     c.Model,
		MaxTokens: 768,
		System:    prompt.TranscriptClassifierSystem(),
		Messages: []messageBlock{
			{Role: "user", Content: []contentPart{{Type: "text", Text: string(userBytes)}}},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return agent.TranscriptClassifierResult{}, err
	}
	url := c.BaseURL + "/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return agent.TranscriptClassifierResult{}, err
	}
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)
	req.Header.Set("Content-Type", "application/json")

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return agent.TranscriptClassifierResult{}, fmt.Errorf("anthropic: request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return agent.TranscriptClassifierResult{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return agent.TranscriptClassifierResult{}, fmt.Errorf("anthropic: HTTP %s: %s", resp.Status, truncateForErr(respBody, 512))
	}
	var parsed messagesResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return agent.TranscriptClassifierResult{}, fmt.Errorf("anthropic: decode: %w", err)
	}
	if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
		return agent.TranscriptClassifierResult{}, fmt.Errorf("anthropic: %s", parsed.Error.Message)
	}
	var text string
	for _, b := range parsed.Content {
		if b.Type == "text" {
			text += b.Text
		}
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return agent.TranscriptClassifierResult{}, fmt.Errorf("anthropic: empty assistant content")
	}
	var out struct {
		Kind       string `json:"kind"`
		SearchQuery string `json:"searchQuery"`
		AnswerText string `json:"answerText"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return agent.TranscriptClassifierResult{}, fmt.Errorf("anthropic: decode classifier: %w", err)
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
		return agent.ScopedEditResult{}, fmt.Errorf("anthropic: missing API key")
	}
	userBytes, err := prompt.ScopedEditUserJSON(in)
	if err != nil {
		return agent.ScopedEditResult{}, fmt.Errorf("anthropic: prompt: %w", err)
	}
	body := messagesRequest{
		Model:     c.Model,
		MaxTokens: 4096,
		System:    prompt.ScopedEditSystem(),
		Messages: []messageBlock{
			{Role: "user", Content: []contentPart{{Type: "text", Text: string(userBytes)}}},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return agent.ScopedEditResult{}, err
	}
	url := c.BaseURL + "/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return agent.ScopedEditResult{}, err
	}
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)
	req.Header.Set("Content-Type", "application/json")

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return agent.ScopedEditResult{}, fmt.Errorf("anthropic: request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return agent.ScopedEditResult{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return agent.ScopedEditResult{}, fmt.Errorf("anthropic: HTTP %s: %s", resp.Status, truncateForErr(respBody, 512))
	}
	var parsed messagesResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return agent.ScopedEditResult{}, fmt.Errorf("anthropic: decode: %w", err)
	}
	if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
		return agent.ScopedEditResult{}, fmt.Errorf("anthropic: %s", parsed.Error.Message)
	}
	var text string
	for _, b := range parsed.Content {
		if b.Type == "text" {
			text += b.Text
		}
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return agent.ScopedEditResult{}, fmt.Errorf("anthropic: empty assistant content")
	}
	var out struct {
		ReplacementText string `json:"replacementText"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return agent.ScopedEditResult{}, fmt.Errorf("anthropic: decode scoped edit: %w", err)
	}
	res := agent.ScopedEditResult{ReplacementText: out.ReplacementText}
	if err := res.Validate(); err != nil {
		return agent.ScopedEditResult{}, err
	}
	return res, nil
}

func (c *Client) ScopeIntent(ctx context.Context, in agentcontext.ScopeIntentContext) (agent.ScopeIntentResult, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return agent.ScopeIntentResult{}, fmt.Errorf("anthropic: missing API key")
	}
	userBytes, err := prompt.ScopeIntentUserJSON(in)
	if err != nil {
		return agent.ScopeIntentResult{}, fmt.Errorf("anthropic: prompt: %w", err)
	}
	body := messagesRequest{
		Model:     c.Model,
		MaxTokens: 512,
		System:    prompt.ScopeIntentSystem(),
		Messages: []messageBlock{
			{Role: "user", Content: []contentPart{{Type: "text", Text: string(userBytes)}}},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return agent.ScopeIntentResult{}, err
	}
	url := c.BaseURL + "/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return agent.ScopeIntentResult{}, err
	}
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)
	req.Header.Set("Content-Type", "application/json")

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return agent.ScopeIntentResult{}, fmt.Errorf("anthropic: request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return agent.ScopeIntentResult{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return agent.ScopeIntentResult{}, fmt.Errorf("anthropic: HTTP %s: %s", resp.Status, truncateForErr(respBody, 512))
	}
	var parsed messagesResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return agent.ScopeIntentResult{}, fmt.Errorf("anthropic: decode: %w", err)
	}
	if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
		return agent.ScopeIntentResult{}, fmt.Errorf("anthropic: %s", parsed.Error.Message)
	}
	var text string
	for _, b := range parsed.Content {
		if b.Type == "text" {
			text += b.Text
		}
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return agent.ScopeIntentResult{}, fmt.Errorf("anthropic: empty assistant content")
	}
	var out struct {
		ScopeKind       string `json:"scopeKind"`
		SymbolName      string `json:"symbolName"`
		ClarifyQuestion string `json:"clarifyQuestion"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return agent.ScopeIntentResult{}, fmt.Errorf("anthropic: decode scope intent: %w", err)
	}
	res := agent.ScopeIntentResult{
		ScopeKind:       agent.ScopeKind(strings.TrimSpace(out.ScopeKind)),
		SymbolName:      out.SymbolName,
		ClarifyQuestion: out.ClarifyQuestion,
	}
	if err := res.Validate(); err != nil {
		return agent.ScopeIntentResult{}, err
	}
	return res, nil
}

type messagesRequest struct {
	Model     string         `json:"model"`
	MaxTokens int            `json:"max_tokens"`
	System    string         `json:"system"`
	Messages  []messageBlock `json:"messages"`
}

type messageBlock struct {
	Role    string        `json:"role"`
	Content []contentPart `json:"content"`
}

type contentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type messagesResponse struct {
	Content []contentPart `json:"content"`
	Error   *struct {
		Type    string `json:"type"`
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
