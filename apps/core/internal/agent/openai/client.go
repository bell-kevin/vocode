// Package openai implements [agent.ModelClient] via the OpenAI Chat Completions API.
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

	"vocoding.net/vocode/v2/apps/core/internal/agent"
)

type Client struct {
	HTTPClient *http.Client
	APIKey     string
	BaseURL    string
	Model      string
}

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

func (c *Client) Complete(ctx context.Context, req agent.CompletionRequest) (string, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return "", fmt.Errorf("openai: missing API key")
	}
	temp := 0.0
	body := chatCompletionsRequest{
		Model:       c.Model,
		Temperature: &temp,
		Messages: []chatMessage{
			{Role: "system", Content: req.System},
			{Role: "user", Content: req.User},
		},
	}
	if len(req.JSONSchema) > 0 {
		body.ResponseFormat = chatResponseFormatFromSchema(req.JSONSchema)
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	url := c.BaseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("openai: request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("openai: HTTP %s: %s", resp.Status, truncateForErr(respBody, 512))
	}
	var parsed chatCompletionsResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("openai: decode response: %w", err)
	}
	if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
		return "", fmt.Errorf("openai: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("openai: empty choices")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

type chatCompletionsRequest struct {
	Model          string          `json:"model"`
	Temperature    *float64        `json:"temperature,omitempty"`
	Messages       []chatMessage   `json:"messages"`
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
	} `json:"error,omitempty"`
}

func truncateForErr(b []byte, max int) string {
	s := strings.TrimSpace(string(b))
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
