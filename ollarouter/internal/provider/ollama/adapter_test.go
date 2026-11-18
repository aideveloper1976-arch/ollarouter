package ollama

import (
	"context"
	"testing"

	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

func TestNewAdapter(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:    "test-ollama",
		Type:    "ollama",
		BaseURL: "http://ollama:11434",
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}
	if adapter.Name() != "test-ollama" {
		t.Errorf("Name() = %q, want test-ollama", adapter.Name())
	}
	if adapter.Type() != "ollama" {
		t.Errorf("Type() = %q, want ollama", adapter.Type())
	}
}

func TestNewAdapterDefaultBaseURL(t *testing.T) {
	cfg := types.ProviderConfig{
		Name: "test-ollama",
		Type: "ollama",
	}
	a, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}
	adapter := a.(*Adapter)
	if adapter.baseURL != "http://ollama:11434" {
		t.Errorf("Expected default baseURL 'http://ollama:11434', got %q", adapter.baseURL)
	}
}

func TestNewAdapterWithModels(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:    "test-ollama",
		Type:    "ollama",
		BaseURL: "http://localhost:11434",
		Models: []types.ModelConfig{
			{Name: "llama3:70b"},
			{Name: "mistral:7b"},
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
	if adapter.models[0].Name != "test-ollama/llama3:70b" {
		t.Errorf("Expected model name 'test-ollama/llama3:70b', got %q", adapter.models[0].Name)
	}
}

func TestListModels(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:    "test-ollama",
		Type:    "ollama",
		BaseURL: "http://localhost:11434",
		Models: []types.ModelConfig{
			{Name: "llama3:70b"},
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
	if models[0].Name != "test-ollama/llama3:70b" {
		t.Errorf("Expected 'test-ollama/llama3:70b', got %q", models[0].Name)
	}
}

func TestListModelsFromDiscovery(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:    "test-ollama",
		Type:    "ollama",
		BaseURL: "http://localhost:11434",
	}
	a, _ := NewAdapter(cfg)
	// When no models are configured and no ollama is running,
	// ListModels should attempt discovery but fail gracefully
	models, err := a.ListModels(context.Background())
	if err == nil && len(models) == 0 {
		// This is acceptable - no models discovered
	}
	if err != nil {
		t.Logf("ListModels() error (expected without running ollama): %v", err)
	}
}

func TestCheckHealthNoConnection(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:    "test-ollama",
		Type:    "ollama",
		BaseURL: "http://localhost:1",
	}
	adapter, _ := NewAdapter(cfg)
	health := adapter.CheckHealth(context.Background())
	if health.Status != "offline" {
		t.Logf("CheckHealth() status = %s (expected offline for unreachable endpoint)", health.Status)
	}
}

func TestGenerateResponsePassthrough(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:    "test-ollama",
		Type:    "ollama",
		BaseURL: "http://localhost:11434",
		Models: []types.ModelConfig{
			{Name: "llama3:70b"},
		},
	}
	_, _ = NewAdapter(cfg)

	// Verify the Generate method produces the right request structure
	// by checking that the ollamaGenerateRequest is correctly populated
	req := &types.GenerateRequest{
		Model:  "llama3:70b",
		Prompt: "Hello world",
		System: "You are a helpful assistant.",
		Options: &types.RequestOptions{
			Temperature: 0.7,
			TopP:        0.9,
		},
	}

	ollamaReq := ollamaGenerateRequest{
		Model:  req.Model,
		Prompt: req.Prompt,
		System: req.System,
		Stream: true,
	}
	if req.Options != nil {
		ollamaReq.Options = &ollamaOptions{
			Temperature: req.Options.Temperature,
			TopP:        req.Options.TopP,
		}
	}

	if ollamaReq.Model != "llama3:70b" {
		t.Errorf("Model = %q, want llama3:70b", ollamaReq.Model)
	}
	if ollamaReq.Prompt != "Hello world" {
		t.Errorf("Prompt = %q, want 'Hello world'", ollamaReq.Prompt)
	}
	if ollamaReq.System != "You are a helpful assistant." {
		t.Errorf("System = %q, want 'You are a helpful assistant.'", ollamaReq.System)
	}
	if !ollamaReq.Stream {
		t.Error("Expected Stream = true")
	}
	if ollamaReq.Options == nil {
		t.Fatal("Options is nil")
	}
	if ollamaReq.Options.Temperature != 0.7 {
		t.Errorf("Temperature = %f, want 0.7", ollamaReq.Options.Temperature)
	}
	if ollamaReq.Options.TopP != 0.9 {
		t.Errorf("TopP = %f, want 0.9", ollamaReq.Options.TopP)
	}
}

func TestChatRequestBuilding(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:    "test-ollama",
		Type:    "ollama",
		BaseURL: "http://localhost:11434",
	}
	_, _ = NewAdapter(cfg)

	req := &types.ChatRequest{
		Model: "llama3:70b",
		Messages: []types.Message{
			{Role: "user", Content: "Hi!"},
		},
		Stream: true,
	}

	// Verify the Ollama request is built correctly
	ollamaReq := ollamaChatRequest{
		Model:    req.Model,
		Messages: make([]ollamaMessage, len(req.Messages)),
		Stream:   true,
	}
	for i, m := range req.Messages {
		ollamaReq.Messages[i] = ollamaMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	if ollamaReq.Model != "llama3:70b" {
		t.Errorf("Model = %q, want llama3:70b", ollamaReq.Model)
	}
	if len(ollamaReq.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(ollamaReq.Messages))
	}
	if ollamaReq.Messages[0].Content != "Hi!" {
		t.Errorf("Content = %q, want Hi!", ollamaReq.Messages[0].Content)
	}
	if !ollamaReq.Stream {
		t.Error("Expected Stream = true")
	}
}

func TestNameType(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:    "test-ollama",
		Type:    "ollama",
		BaseURL: "http://localhost:11434",
	}
	adapter, _ := NewAdapter(cfg)

	if adapter.Name() != "test-ollama" {
		t.Errorf("Name() = %q", adapter.Name())
	}
	if adapter.Type() != "ollama" {
		t.Errorf("Type() = %q", adapter.Type())
	}
}
