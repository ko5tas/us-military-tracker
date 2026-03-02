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

// Completer is the interface that AI providers must implement to generate text completions.
type Completer interface {
	Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	Name() string
}

// ChatMessage represents a single message in a chat conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the request body for the OpenAI-compatible chat completions API.
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

// Choice represents one completion choice in the response.
type Choice struct {
	Index   int         `json:"index"`
	Message ChatMessage `json:"message"`
}

// Usage contains token usage information.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResponse is the response body from the OpenAI-compatible chat completions API.
type ChatResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// OpenAIProvider implements Completer for any OpenAI-compatible API.
type OpenAIProvider struct {
	ProviderName string
	BaseURL      string
	APIKey       string
	ModelName    string
	MaxTokens    int
	HTTPClient   *http.Client
}

// Name returns the provider's display name.
func (p *OpenAIProvider) Name() string {
	return p.ProviderName
}

// Complete sends a chat completion request to an OpenAI-compatible endpoint.
func (p *OpenAIProvider) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	maxTokens := p.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}
	reqBody := ChatRequest{
		Model: p.ModelName,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
		MaxTokens:   maxTokens,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := p.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	client := p.HTTPClient
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

	var chatResp ChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// newOpenAI creates an OpenAIProvider with the given configuration.
func newOpenAI(name, baseURL, apiKey, model string, timeout time.Duration) *OpenAIProvider {
	return &OpenAIProvider{
		ProviderName: name,
		BaseURL:      baseURL,
		APIKey:       apiKey,
		ModelName:    model,
		HTTPClient:   &http.Client{Timeout: timeout},
	}
}

// NewGroq creates a Completer backed by the Groq API.
func NewGroq(apiKey string) Completer {
	return newOpenAI("groq", "https://api.groq.com/openai/v1", apiKey, "llama-3.3-70b-versatile", 30*time.Second)
}

// NewMistral creates a Completer backed by the Mistral API.
func NewMistral(apiKey string) Completer {
	return newOpenAI("mistral", "https://api.mistral.ai/v1", apiKey, "mistral-small-latest", 30*time.Second)
}

// NewDeepSeek creates a Completer backed by the DeepSeek API.
func NewDeepSeek(apiKey string) Completer {
	return newOpenAI("deepseek", "https://api.deepseek.com", apiKey, "deepseek-chat", 30*time.Second)
}

// NewOpenRouter creates a Completer backed by the OpenRouter API.
func NewOpenRouter(apiKey string) Completer {
	return newOpenAI("openrouter", "https://openrouter.ai/api/v1", apiKey, "meta-llama/llama-3.3-70b-instruct:free", 30*time.Second)
}

// NewOllama creates a Completer backed by a local Ollama instance.
func NewOllama() Completer {
	return newOpenAI("ollama", "http://localhost:11434/v1", "", "qwen2.5:1.5b", 120*time.Second)
}
