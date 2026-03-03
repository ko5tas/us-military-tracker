package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ClaudeProvider implements Completer for the Anthropic Messages API.
type ClaudeProvider struct {
	ProviderName string
	APIKey       string
	ModelName    string
	HTTPClient   *http.Client
}

// Name returns the provider's display name.
func (c *ClaudeProvider) Name() string {
	return c.ProviderName
}

// claudeRequest is the request body for the Anthropic Messages API.
type claudeRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	System    string        `json:"system,omitempty"`
	Messages  []ChatMessage `json:"messages"`
}

// claudeContent represents a content block in the Anthropic response.
type claudeContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// claudeResponse is the response body from the Anthropic Messages API.
type claudeResponse struct {
	Content []claudeContent `json:"content"`
}

// Complete sends a prompt to the Anthropic Messages API and returns the generated text.
func (c *ClaudeProvider) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody := claudeRequest{
		Model:     c.ModelName,
		MaxTokens: 4096,
		System:    systemPrompt,
		Messages: []ChatMessage{
			{Role: "user", Content: userPrompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(respBytes, &claudeResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	// Concatenate all text blocks
	var text string
	for _, block := range claudeResp.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}

	return text, nil
}

// NewClaude creates a Completer backed by the Anthropic Claude API.
func NewClaude(apiKey string) Completer {
	return &ClaudeProvider{
		ProviderName: "claude",
		APIKey:       apiKey,
		ModelName:    "claude-haiku-4-5-20251001",
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
	}
}
