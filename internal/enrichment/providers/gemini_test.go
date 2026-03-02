package providers

import (
	"context"
	"testing"
)

func TestGeminiProvider_Name(t *testing.T) {
	p := &GeminiProvider{
		ProviderName: "test-gemini",
		ModelName:    "gemini-2.0-flash",
	}

	if got := p.Name(); got != "test-gemini" {
		t.Errorf("Name() = %q, want %q", got, "test-gemini")
	}
}

func TestNewGemini_EmptyAPIKey(t *testing.T) {
	_, err := NewGemini(context.Background(), "", "gemini", "gemini-2.0-flash")
	if err == nil {
		t.Fatal("NewGemini with empty API key should return error")
	}
}

func TestGeminiProvider_Complete_NilClient(t *testing.T) {
	p := &GeminiProvider{
		ProviderName: "test-gemini",
		ModelName:    "gemini-2.0-flash",
		Client:       nil,
	}

	_, err := p.Complete(context.Background(), "system", "user")
	if err == nil {
		t.Fatal("Complete with nil client should return error")
	}
}
