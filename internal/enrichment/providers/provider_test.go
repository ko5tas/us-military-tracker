package providers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenAIProviderComplete(t *testing.T) {
	// Set up a mock OpenAI-compatible server.
	var receivedReq ChatRequest
	var receivedAuthHeader string
	var receivedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("Authorization")
		receivedContentType = r.Header.Get("Content-Type")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		if err := json.Unmarshal(body, &receivedReq); err != nil {
			t.Errorf("unmarshalling request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		resp := ChatResponse{
			ID: "chatcmpl-test123",
			Choices: []Choice{
				{
					Index: 0,
					Message: ChatMessage{
						Role:    "assistant",
						Content: "This is a test response.",
					},
				},
			},
			Usage: Usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := &OpenAIProvider{
		ProviderName: "test-provider",
		BaseURL:      server.URL,
		APIKey:       "test-api-key-123",
		ModelName:    "test-model",
		HTTPClient:   server.Client(),
	}

	// Verify it implements Completer.
	var _ Completer = provider

	result, err := provider.Complete(context.Background(), "You are a helpful assistant.", "What is 2+2?")
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	// Verify response content.
	if result != "This is a test response." {
		t.Errorf("result: got %q, want %q", result, "This is a test response.")
	}

	// Verify Name().
	if provider.Name() != "test-provider" {
		t.Errorf("Name: got %q, want %q", provider.Name(), "test-provider")
	}

	// Verify auth header was sent.
	if receivedAuthHeader != "Bearer test-api-key-123" {
		t.Errorf("Authorization header: got %q, want %q", receivedAuthHeader, "Bearer test-api-key-123")
	}

	// Verify content type.
	if receivedContentType != "application/json" {
		t.Errorf("Content-Type header: got %q, want %q", receivedContentType, "application/json")
	}

	// Verify request body.
	if receivedReq.Model != "test-model" {
		t.Errorf("request Model: got %q, want %q", receivedReq.Model, "test-model")
	}
	if receivedReq.Temperature != 0.3 {
		t.Errorf("request Temperature: got %v, want %v", receivedReq.Temperature, 0.3)
	}
	if len(receivedReq.Messages) != 2 {
		t.Fatalf("request Messages count: got %d, want 2", len(receivedReq.Messages))
	}
	if receivedReq.Messages[0].Role != "system" {
		t.Errorf("Messages[0].Role: got %q, want %q", receivedReq.Messages[0].Role, "system")
	}
	if receivedReq.Messages[0].Content != "You are a helpful assistant." {
		t.Errorf("Messages[0].Content: got %q, want %q", receivedReq.Messages[0].Content, "You are a helpful assistant.")
	}
	if receivedReq.Messages[1].Role != "user" {
		t.Errorf("Messages[1].Role: got %q, want %q", receivedReq.Messages[1].Role, "user")
	}
	if receivedReq.Messages[1].Content != "What is 2+2?" {
		t.Errorf("Messages[1].Content: got %q, want %q", receivedReq.Messages[1].Content, "What is 2+2?")
	}
}

func TestOpenAIProviderNoAPIKey(t *testing.T) {
	var receivedAuthHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("Authorization")

		resp := ChatResponse{
			ID: "chatcmpl-nokey",
			Choices: []Choice{
				{
					Index: 0,
					Message: ChatMessage{
						Role:    "assistant",
						Content: "Response without auth.",
					},
				},
			},
			Usage: Usage{
				PromptTokens:     5,
				CompletionTokens: 3,
				TotalTokens:      8,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := &OpenAIProvider{
		ProviderName: "ollama-local",
		BaseURL:      server.URL,
		APIKey:       "",
		ModelName:    "qwen2.5:1.5b",
		HTTPClient:   server.Client(),
	}

	result, err := provider.Complete(context.Background(), "system", "user")
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	if result != "Response without auth." {
		t.Errorf("result: got %q, want %q", result, "Response without auth.")
	}

	// When APIKey is empty, no Authorization header should be set.
	if receivedAuthHeader != "" {
		t.Errorf("Authorization header should be empty when APIKey is empty, got %q", receivedAuthHeader)
	}
}

func TestOpenAIProviderErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"rate limit exceeded","type":"rate_limit_error"}}`))
	}))
	defer server.Close()

	provider := &OpenAIProvider{
		ProviderName: "test-provider",
		BaseURL:      server.URL,
		APIKey:       "test-key",
		ModelName:    "test-model",
		HTTPClient:   server.Client(),
	}

	_, err := provider.Complete(context.Background(), "system", "user")
	if err == nil {
		t.Fatal("expected error for 429 status, got nil")
	}

	// Error should contain status code information.
	errMsg := err.Error()
	if !contains(errMsg, "429") {
		t.Errorf("error should mention status code 429, got: %s", errMsg)
	}

	// Error should contain the response body.
	if !contains(errMsg, "rate limit exceeded") {
		t.Errorf("error should contain response body, got: %s", errMsg)
	}
}

func TestOpenAIProviderNoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChatResponse{
			ID:      "chatcmpl-empty",
			Choices: []Choice{},
			Usage: Usage{
				PromptTokens:     10,
				CompletionTokens: 0,
				TotalTokens:      10,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := &OpenAIProvider{
		ProviderName: "test-provider",
		BaseURL:      server.URL,
		APIKey:       "test-key",
		ModelName:    "test-model",
		HTTPClient:   server.Client(),
	}

	_, err := provider.Complete(context.Background(), "system", "user")
	if err == nil {
		t.Fatal("expected error for empty choices, got nil")
	}

	if !contains(err.Error(), "no choices") {
		t.Errorf("error should mention 'no choices', got: %s", err.Error())
	}
}

func TestFactoryFunctions(t *testing.T) {
	tests := []struct {
		name        string
		factory     func() Completer
		wantName    string
		wantBaseURL string
		wantModel   string
		wantHasKey  bool
		wantTimeout time.Duration
	}{
		{
			name:        "NewGroq",
			factory:     func() Completer { return NewGroq("groq-key") },
			wantName:    "groq",
			wantBaseURL: "https://api.groq.com/openai/v1",
			wantModel:   "llama-3.3-70b-versatile",
			wantHasKey:  true,
			wantTimeout: 30 * time.Second,
		},
		{
			name:        "NewMistral",
			factory:     func() Completer { return NewMistral("mistral-key") },
			wantName:    "mistral",
			wantBaseURL: "https://api.mistral.ai/v1",
			wantModel:   "mistral-small-latest",
			wantHasKey:  true,
			wantTimeout: 30 * time.Second,
		},
		{
			name:        "NewDeepSeek",
			factory:     func() Completer { return NewDeepSeek("deepseek-key") },
			wantName:    "deepseek",
			wantBaseURL: "https://api.deepseek.com",
			wantModel:   "deepseek-chat",
			wantHasKey:  true,
			wantTimeout: 30 * time.Second,
		},
		{
			name:        "NewOpenRouter",
			factory:     func() Completer { return NewOpenRouter("openrouter-key") },
			wantName:    "openrouter",
			wantBaseURL: "https://openrouter.ai/api/v1",
			wantModel:   "meta-llama/llama-3.3-70b-instruct:free",
			wantHasKey:  true,
			wantTimeout: 30 * time.Second,
		},
		{
			name:        "NewOllama",
			factory:     func() Completer { return NewOllama() },
			wantName:    "ollama",
			wantBaseURL: "http://localhost:11434/v1",
			wantModel:   "qwen2.5:1.5b",
			wantHasKey:  false,
			wantTimeout: 120 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.factory()

			if c.Name() != tt.wantName {
				t.Errorf("Name: got %q, want %q", c.Name(), tt.wantName)
			}

			// Type-assert to check internal fields.
			p, ok := c.(*OpenAIProvider)
			if !ok {
				t.Fatalf("factory did not return *OpenAIProvider")
			}

			if p.BaseURL != tt.wantBaseURL {
				t.Errorf("BaseURL: got %q, want %q", p.BaseURL, tt.wantBaseURL)
			}
			if p.ModelName != tt.wantModel {
				t.Errorf("ModelName: got %q, want %q", p.ModelName, tt.wantModel)
			}

			if tt.wantHasKey && p.APIKey == "" {
				t.Error("expected APIKey to be set")
			}
			if !tt.wantHasKey && p.APIKey != "" {
				t.Errorf("expected no APIKey, got %q", p.APIKey)
			}

			if p.HTTPClient == nil {
				t.Fatal("HTTPClient should not be nil")
			}

			if p.HTTPClient.Timeout != tt.wantTimeout {
				t.Errorf("Timeout: got %v, want %v", p.HTTPClient.Timeout, tt.wantTimeout)
			}
		})
	}
}

// contains checks whether s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
