package provider

import (
	"context"
	"testing"

	"github.com/aideveloper1976-arch/ollarouter/internal/config"
	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

// mockProvider implements types.Provider for testing.
type mockProvider struct{}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) Type() string { return "openai" }
func (m *mockProvider) Chat(ctx context.Context, req *types.ChatRequest) (*types.ChatResponse, error) {
	return &types.ChatResponse{Model: req.Model, Message: types.Message{Role: "assistant", Content: "ok"}}, nil
}
func (m *mockProvider) ChatStream(ctx context.Context, req *types.ChatRequest, handler types.StreamHandler) error {
	return handler(&types.StreamChunk{Content: "ok", Done: true})
}
func (m *mockProvider) Generate(ctx context.Context, req *types.GenerateRequest) (*types.GenerateResponse, error) {
	return &types.GenerateResponse{Model: req.Model, Response: "ok"}, nil
}
func (m *mockProvider) GenerateStream(ctx context.Context, req *types.GenerateRequest, handler types.StreamHandler) error {
	return handler(&types.StreamChunk{Content: "ok", Done: true})
}
func (m *mockProvider) ListModels(ctx context.Context) ([]types.ModelInfo, error) {
	return []types.ModelInfo{{Name: "mock/gpt-4", ProviderName: "mock"}}, nil
}
func (m *mockProvider) CheckHealth(ctx context.Context) types.ProviderHealth {
	return types.ProviderHealth{Status: "online", LatencyMS: 1}
}

func TestRegistryUpdatePreservesOptionalFields(t *testing.T) {
	reg := NewRegistry()

	// Create a provider with all optional fields set
	cfg := config.ProviderConfig{
		Name:             "test-provider",
		Type:             "chatgpt-plus",
		BaseURL:          "https://chatgpt.com",
		APIKey:           "test-api-key-example",
		OrganizationID:   "org-123",
		AnthropicVersion: "2023-06-01",
		AutoDiscover:     true,
		PollingInterval:  60,
		MaxRetries:       5,
		TimeoutSec:       120,
		SessionCookie:    "original-session-cookie",
		UserAgent:        "custom-user-agent",
		Auth: &config.AuthSettings{
			Mode:            "oauth",
			ClientID:        "client-123",
			ClientSecret:    "secret-456",
			RedirectURI:     "https://example.com/callback",
			CallbackPort:    11437,
			KeychainService: "ollarouter-test",
			TokenEndpoint:   "https://example.com/token",
			AuthEndpoint:    "https://example.com/auth",
			Scopes:          "openid profile",
		},
		Pricing: &config.PricingConfig{
			InputPer1K:  5.0,
			OutputPer1K: 15.0,
		},
	}

	if err := reg.Add(cfg, &mockProvider{}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Update only the BaseURL
	updateCfg := config.ProviderConfig{
		Name:    "test-provider",
		Type:    "chatgpt-plus",
		BaseURL: "https://updated.chatgpt.com",
	}

	if err := reg.Update("test-provider", updateCfg, &mockProvider{}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	ps, ok := reg.Get("test-provider")
	if !ok {
		t.Fatal("Get() after update returned not found")
	}

	// Verify Auth settings preserved
	if ps.Config.Auth == nil {
		t.Fatal("Auth settings were dropped by Update()")
	}
	if ps.Config.Auth.Mode != "oauth" {
		t.Errorf("Auth.Mode = %q, want %q", ps.Config.Auth.Mode, "oauth")
	}
	if ps.Config.Auth.ClientSecret != "secret-456" {
		t.Errorf("Auth.ClientSecret = %q, want %q", ps.Config.Auth.ClientSecret, "secret-456")
	}

	// Verify SessionCookie preserved
	if ps.Config.SessionCookie != "original-session-cookie" {
		t.Errorf("SessionCookie = %q, want %q", ps.Config.SessionCookie, "original-session-cookie")
	}

	// Verify UserAgent preserved
	if ps.Config.UserAgent != "custom-user-agent" {
		t.Errorf("UserAgent = %q, want %q", ps.Config.UserAgent, "custom-user-agent")
	}

	// Verify OrganizationID preserved
	if ps.Config.OrganizationID != "org-123" {
		t.Errorf("OrganizationID = %q, want %q", ps.Config.OrganizationID, "org-123")
	}

	// Verify Pricing preserved
	if ps.Config.Pricing == nil {
		t.Fatal("Pricing was dropped by Update()")
	}
	if ps.Config.Pricing.InputPer1K != 5.0 {
		t.Errorf("Pricing.InputPer1K = %f, want 5.0", ps.Config.Pricing.InputPer1K)
	}

	// Verify new BaseURL took effect
	if ps.Config.BaseURL != "https://updated.chatgpt.com" {
		t.Errorf("BaseURL = %q, want %q", ps.Config.BaseURL, "https://updated.chatgpt.com")
	}
}
