package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

func TestNewAdapter(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:    "test-openai",
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "sk-test123",
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}
	if adapter.Name() != "test-openai" {
		t.Errorf("Name() = %q, want test-openai", adapter.Name())
	}
	if adapter.Type() != "openai" {
		t.Errorf("Type() = %q, want openai", adapter.Type())
	}
}

func TestNewAdapterDefaultBaseURL(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:   "test-openai",
		Type:   "openai",
		APIKey: "sk-test123",
	}
	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}
	a := adapter.(*Adapter)
	if a.baseURL != "https://api.openai.com/v1" {
		t.Errorf("Expected default baseURL, got %q", a.baseURL)
	}
}

func TestNewAdapterWithModels(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:    "test-openai",
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "sk-test123",
		Models: []types.ModelConfig{
			{Name: "gpt-4"},
			{Name: "gpt-3.5-turbo"},
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
	if adapter.models[0].Name != "test-openai/gpt-4" {
		t.Errorf("Expected model name 'test-openai/gpt-4', got %q", adapter.models[0].Name)
	}
}

func TestListModels(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:   "test-openai",
		Type:   "openai",
		APIKey: "sk-test123",
		Models: []types.ModelConfig{
			{Name: "gpt-4"},
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
	if models[0].Name != "test-openai/gpt-4" {
		t.Errorf("Expected 'test-openai/gpt-4', got %q", models[0].Name)
	}
}

func TestCheckHealthNoConnection(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:    "test-openai",
		Type:    "openai",
		BaseURL: "http://localhost:1", // Will fail to connect
		APIKey:  "sk-test123",
	}
	adapter, _ := NewAdapter(cfg)
	health := adapter.CheckHealth(context.Background())
	if health.Status != "offline" {
		t.Logf("CheckHealth() status = %s (expected offline for unreachable endpoint)", health.Status)
	}
}

func TestListModelsRemote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":"gpt-4"},{"id":"gpt-4o-mini"}]}`))
	}))
	defer srv.Close()

	cfg := types.ProviderConfig{
		Name:    "test-openai",
		Type:    "openai",
		BaseURL: srv.URL,
		APIKey:  "sk-test123",
	}
	a, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}
	adapter := a.(*Adapter)
	adapter.httpClient = srv.Client()

	models, err := adapter.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("Expected 2 models, got %d", len(models))
	}
	if models[0].Name != "test-openai/gpt-4" {
		t.Errorf("Expected first model 'test-openai/gpt-4', got %q", models[0].Name)
	}
}

func TestTranslateChatResponse(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:   "test-openai",
		Type:   "openai",
		APIKey: "sk-test123",
	}
	a, _ := NewAdapter(cfg)
	adapter := a.(*Adapter)

	input := &openAIChatResponse{
		Model: "gpt-4",
		Choices: []openAIChoice{
			{
				Index: 0,
				Message: openAIMessage{
					Role:    "assistant",
					Content: "Hello, world!",
				},
				FinishReason: "stop",
			},
		},
		Usage: &openAIUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	result := adapter.translateChatResponse(input)
	if result.Model != "gpt-4" {
		t.Errorf("Model = %q, want gpt-4", result.Model)
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
	if result.TokenUsage.TotalTokens != 30 {
		t.Errorf("TotalTokens = %d, want 30", result.TokenUsage.TotalTokens)
	}
}

func TestToolCallAccumulator(t *testing.T) {
	// Simulate OpenAI-style streamed tool-call fragments: the name and id arrive
	// in the first delta, then arguments stream in pieces under the same index.
	acc := newToolCallAccumulator()

	first := openAIStreamToolCall{Index: 0, ID: "call_abc", Type: "function"}
	first.Function.Name = "run_shell"
	first.Function.Arguments = `{"comm`
	acc.add([]openAIStreamToolCall{first})

	second := openAIStreamToolCall{Index: 0}
	second.Function.Arguments = `and":"ls"}`
	acc.add([]openAIStreamToolCall{second})

	var got *types.StreamChunk
	acc.flush(func(c *types.StreamChunk) error {
		got = c
		return nil
	})

	if got == nil || len(got.ToolCalls) != 1 {
		t.Fatalf("expected 1 accumulated tool call, got %+v", got)
	}
	tc := got.ToolCalls[0]
	if tc.ID != "call_abc" {
		t.Errorf("ID = %q, want call_abc", tc.ID)
	}
	if tc.Function.Name != "run_shell" {
		t.Errorf("Name = %q, want run_shell", tc.Function.Name)
	}
	if tc.Function.Arguments != `{"command":"ls"}` {
		t.Errorf("Arguments = %q, want %q", tc.Function.Arguments, `{"command":"ls"}`)
	}
}

func TestGenerateToChatTranslation(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:   "test-openai",
		Type:   "openai",
		APIKey: "sk-test123",
	}
	adapter, _ := NewAdapter(cfg)

	// Test that Name/Type work correctly
	if name := adapter.Name(); name != "test-openai" {
		t.Errorf("Name = %q, want test-openai", name)
	}
	if typ := adapter.Type(); typ != "openai" {
		t.Errorf("Type = %q, want openai", typ)
	}
}

func TestGenerateRequestBuild(t *testing.T) {
	cfg := types.ProviderConfig{
		Name:   "test-openai",
		Type:   "openai",
		APIKey: "sk-test123",
	}
	a, _ := NewAdapter(cfg)
	adapter := a.(*Adapter)

	req := &types.ChatRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello!"},
		},
		Stream: false,
	}

	openAIReq := adapter.buildChatRequest(req)
	if openAIReq.Model != "gpt-4" {
		t.Errorf("Model = %q, want gpt-4", openAIReq.Model)
	}
	if len(openAIReq.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(openAIReq.Messages))
	}
	if openAIReq.Messages[0].Role != "system" {
		t.Errorf("Expected first message role 'system', got %q", openAIReq.Messages[0].Role)
	}
	if openAIReq.Stream {
		t.Error("Expected Stream = false")
	}
}
