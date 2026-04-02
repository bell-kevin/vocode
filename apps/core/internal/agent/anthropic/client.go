// Package anthropic implements [agent.ModelClient] via the Anthropic Messages API.
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

	"vocoding.net/vocode/v2/apps/core/internal/agent"
)

const apiVersion = "2023-06-01"

// Client calls the Anthropic HTTP API.
type Client struct {
	HTTPClient *http.Client
	APIKey     string
	BaseURL    string
	Model      string
}

// NewFromEnv requires ANTHROPIC_API_KEY. Optional VOCODE_ANTHROPIC_BASE_URL (default https://api.anthropic.com/v1),
// VOCODE_ANTHROPIC_MODEL (default claude-3-5-haiku-20241022).
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
		model = "claude-3-5-haiku-20241022"
	}
	return &Client{
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
		APIKey:     key,
		BaseURL:    base,
		Model:      model,
	}, nil
}

func (c *Client) Call(ctx context.Context, req agent.CompletionRequest) (string, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return "", fmt.Errorf("anthropic: missing API key")
	}
	system := req.System
	if len(req.JSONSchema) > 0 {
		schemaJSON, err := json.MarshalIndent(req.JSONSchema, "", "  ")
		if err != nil {
			return "", fmt.Errorf("anthropic: schema: %w", err)
		}
		system += "\n\nRespond with a single JSON object only (no markdown code fences). It must satisfy this JSON Schema:\n" + string(schemaJSON)
	}
	body := messagesRequest{
		Model:     c.Model,
		MaxTokens: 256,
		System:    system,
		Messages: []messageBlock{
			{Role: "user", Content: []contentPart{{Type: "text", Text: req.User}}},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	url := c.BaseURL + "/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("x-api-key", c.APIKey)
	httpReq.Header.Set("anthropic-version", apiVersion)
	httpReq.Header.Set("Content-Type", "application/json")

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("anthropic: request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("anthropic: HTTP %s: %s", resp.Status, truncateForErr(respBody, 512))
	}
	var parsed messagesResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("anthropic: decode response: %w", err)
	}
	if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
		return "", fmt.Errorf("anthropic: %s", parsed.Error.Message)
	}
	var text string
	for _, b := range parsed.Content {
		if b.Type == "text" {
			text += b.Text
		}
	}
	return strings.TrimSpace(text), nil
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
		Message string `json:"message"`
	} `json:"error"`
}

func truncateForErr(b []byte, max int) string {
	s := strings.TrimSpace(string(b))
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
