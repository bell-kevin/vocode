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
	"vocoding.net/vocode/v2/apps/daemon/internal/agent/turnjson"
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

// NextTurn implements [agent.ModelClient].
func (c *Client) NextTurn(ctx context.Context, in agentcontext.TurnContext) (agent.TurnResult, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return agent.TurnResult{}, fmt.Errorf("openai: missing API key")
	}
	userBytes, err := prompt.UserJSON(in)
	if err != nil {
		return agent.TurnResult{}, fmt.Errorf("openai: prompt: %w", err)
	}
	temp := 0.2
	body := chatCompletionsRequest{
		Model:       c.Model,
		Temperature: &temp,
		Messages: []chatMessage{
			{Role: "system", Content: prompt.System(prompt.SystemConfig{MaxContextRounds: in.Limits.MaxContextRounds})},
			{Role: "user", Content: string(userBytes)},
		},
		ResponseFormat: chatResponseFormat(),
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return agent.TurnResult{}, err
	}
	url := c.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return agent.TurnResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return agent.TurnResult{}, fmt.Errorf("openai: request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return agent.TurnResult{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return agent.TurnResult{}, fmt.Errorf("openai: HTTP %s: %s", resp.Status, truncateForErr(respBody, 512))
	}
	var parsed chatCompletionsResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return agent.TurnResult{}, fmt.Errorf("openai: decode response: %w", err)
	}
	if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
		return agent.TurnResult{}, fmt.Errorf("openai: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return agent.TurnResult{}, fmt.Errorf("openai: empty choices")
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return agent.TurnResult{}, fmt.Errorf("openai: empty message content")
	}
	return turnjson.ParseTurn([]byte(content))
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
