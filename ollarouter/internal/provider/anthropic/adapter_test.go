package anthropic

import (
	"context"
	"testing"

	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

func TestNewAdapter(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:             "test-anthropic",
		Type:             "anthropic",
		BaseURL:          "https://api.anthropic.com/v1",
		APIKey:           "sk-ant-test123",
		AnthropicVersion: "2023-06-01",
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}
	if adapter.Name() != "test-anthropic" {
		t.Errorf("Name() = %q, want test-anthropic", adapter.Name())
	}
	if adapter.Type() != "anthropic" {
		t.Errorf("Type() = %q, want anthropic", adapter.Type())
	}
}

func TestNewAdapterDefaultBaseURL(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:   "test-anthropic",
		Type:   "anthropic",
		APIKey: "sk-ant-test123",
	}
	a, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}
	adapter := a.(*Adapter)
	if adapter.baseURL != "https://api.anthropic.com/v1" {
		t.Errorf("Expected default baseURL, got %q", adapter.baseURL)
	}
	if adapter.version != "2023-06-01" {
		t.Errorf("Expected default version, got %q", adapter.version)
	}
}

func TestNewAdapterWithModels(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:    "test-anthropic",
		Type:    "anthropic",
		BaseURL: "https://api.anthropic.com/v1",
		APIKey:  "sk-ant-test123",
		Models: []types.ModelConfig{
			{Name: "claude-3-opus-20240229"},
			{Name: "claude-3-sonnet-20240229"},
		},
	}
	a, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}
	adapter := a.(*Adapter)
	if len(adapter.models) != 2 {
		t.Fatalf("Expected 2 models, got %d", len(adapter.models))
	}
	if adapter.models[0].Name != "test-anthropic/claude-3-opus-20240229" {
		t.Errorf("Expected model name 'test-anthropic/claude-3-opus-20240229', got %q", adapter.models[0].Name)
	}
}

func TestListModels(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:   "test-anthropic",
		Type:   "anthropic",
		APIKey: "sk-ant-test123",
		Models: []types.ModelConfig{
			{Name: "claude-3-sonnet-20240229"},
		},
	}
	adapter, _ := NewAdapter(cfg)
	models, err := adapter.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("Expected 1 model, got %d", len(models))
	}
	if models[0].Name != "test-anthropic/claude-3-sonnet-20240229" {
		t.Errorf("Expected 'test-anthropic/claude-3-sonnet-20240229', got %q", models[0].Name)
	}
}

func TestBuildRequest(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:   "test-anthropic",
		Type:   "anthropic",
		APIKey: "sk-ant-test123",
	}
	a, _ := NewAdapter(cfg)
	adapter := a.(*Adapter)

	req := &types.ChatRequest{
		Model: "claude-3-sonnet-20240229",
		Messages: []types.Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello!"},
		},
		Options: &types.RequestOptions{
			MaxTokens:   1000,
			Temperature: 0.7,
		},
		Stream: true,
	}

	anthReq := adapter.buildRequest(req)
	if anthReq.Model != "claude-3-sonnet-20240229" {
		t.Errorf("Model = %q, want claude-3-sonnet-20240229", anthReq.Model)
	}
	if anthReq.MaxTokens != 1000 {
		t.Errorf("MaxTokens = %d, want 1000", anthReq.MaxTokens)
	}
	if anthReq.Temperature != 0.7 {
		t.Errorf("Temperature = %f, want 0.7", anthReq.Temperature)
	}
	if !anthReq.Stream {
		t.Error("Expected Stream = true")
	}
	if anthReq.System != "You are a helpful assistant." {
		t.Errorf("System = %q, want 'You are a helpful assistant.'", anthReq.System)
	}
	if len(anthReq.Messages) != 1 {
		t.Fatalf("Expected 1 message (system filtered), got %d", len(anthReq.Messages))
	}
	if anthReq.Messages[0].Role != "user" {
		t.Errorf("Expected message role 'user', got %q", anthReq.Messages[0].Role)
	}
}

func TestTranslateResponse(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:   "test-anthropic",
		Type:   "anthropic",
		APIKey: "sk-ant-test123",
	}
	p, _ := NewAdapter(cfg)
	adapter := p.(*Adapter)

	input := &anthropicResponse{
		ID:    "msg_123",
		Type:  "message",
		Role:  "assistant",
		Model: "claude-3-sonnet-20240229",
		Content: []anthropicContent{
			{Type: "text", Text: "Hello, world!"},
		},
		Usage: &anthropicUsage{
			InputTokens:  10,
			OutputTokens: 20,
		},
	}

	result := adapter.translateResponse(input)
	if result.Model != "claude-3-sonnet-20240229" {
		t.Errorf("Model = %q, want claude-3-sonnet-20240229", result.Model)
	}
	if result.Message.Content != "Hello, world!" {
		t.Errorf("Content = %q, want 'Hello, world!'", result.Message.Content)
	}
	if result.Message.Role != "assistant" {
		t.Errorf("Role = %q, want assistant", result.Message.Role)
	}
	if result.TokenUsage == nil {
		t.Fatal("TokenUsage is nil")
	}
	if result.TokenUsage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want 10", result.TokenUsage.PromptTokens)
	}
	if result.TokenUsage.CompletionTokens != 20 {
		t.Errorf("CompletionTokens = %d, want 20", result.TokenUsage.CompletionTokens)
	}
	if result.TokenUsage.TotalTokens != 30 {
		t.Errorf("TotalTokens = %d, want 30", result.TokenUsage.TotalTokens)
	}
}

func TestTranslateResponseNoUsage(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:   "test-anthropic",
		Type:   "anthropic",
		APIKey: "sk-ant-test123",
	}
	p, _ := NewAdapter(cfg)
	adapter := p.(*Adapter)

	input := &anthropicResponse{
		Model: "claude-3-sonnet-20240229",
		Content: []anthropicContent{
			{Type: "text", Text: "Hello"},
		},
	}

	result := adapter.translateResponse(input)
	if result.TokenUsage != nil {
		t.Error("Expected TokenUsage to be nil when not present in response")
	}
}

func TestCheckHealthNoConnection(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:    "test-anthropic",
		Type:    "anthropic",
		BaseURL: "http://localhost:1",
		APIKey:  "sk-ant-test123",
	}
	a, _ := NewAdapter(cfg)
	adapter := a.(*Adapter)
	health := adapter.CheckHealth(context.Background())
	if health.Status != "offline" {
		t.Logf("CheckHealth() status = %s (expected offline for unreachable endpoint)", health.Status)
	}
}

func TestNameType(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:   "test-anthropic",
		Type:   "anthropic",
		APIKey: "sk-ant-test123",
	}
	adapter, _ := NewAdapter(cfg)

	if adapter.Name() != "test-anthropic" {
		t.Errorf("Name() = %q", adapter.Name())
	}
	if adapter.Type() != "anthropic" {
		t.Errorf("Type() = %q", adapter.Type())
	}
}

func TestContentBlockMultiple(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:   "test-anthropic",
		Type:   "anthropic",
		APIKey: "sk-ant-test123",
	}
	a, _ := NewAdapter(cfg)
	adapter := a.(*Adapter)

	input := &anthropicResponse{
		Model: "claude-3",
		Content: []anthropicContent{
			{Type: "text", Text: "Hello"},
			{Type: "text", Text: " World"},
		},
	}

	result := adapter.translateResponse(input)
	if result.Message.Content != "Hello World" {
		t.Errorf("Expected combined content 'Hello World', got %q", result.Message.Content)
	}
}
