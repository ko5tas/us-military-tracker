package providers

import (
	"context"
	"errors"

	"google.golang.org/genai"
)

// GeminiProvider wraps the Google Gemini generative AI SDK.
type GeminiProvider struct {
	ProviderName string
	ModelName    string
	Client       *genai.Client
}

// Name returns the provider's display name.
func (g *GeminiProvider) Name() string {
	return g.ProviderName
}

// Complete sends a prompt to the Gemini model and returns the generated text.
func (g *GeminiProvider) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if g.Client == nil {
		return "", errors.New("gemini client is nil")
	}

	temp := float32(0.3)
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, "user"),
		Temperature:       &temp,
	}

	resp, err := g.Client.Models.GenerateContent(ctx, g.ModelName, genai.Text(userPrompt), config)
	if err != nil {
		return "", err
	}

	return resp.Text(), nil
}

// NewGemini creates a new GeminiProvider backed by the Gemini API.
func NewGemini(ctx context.Context, apiKey, name, model string) (Completer, error) {
	if apiKey == "" {
		return nil, errors.New("gemini API key must not be empty")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}

	return &GeminiProvider{
		ProviderName: name,
		ModelName:    model,
		Client:       client,
	}, nil
}
