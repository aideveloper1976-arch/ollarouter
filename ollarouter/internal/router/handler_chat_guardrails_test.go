package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aideveloper1976-arch/ollarouter/internal/auth"
	"github.com/aideveloper1976-arch/ollarouter/internal/config"
	"github.com/aideveloper1976-arch/ollarouter/internal/engine"
	"github.com/aideveloper1976-arch/ollarouter/internal/logger"
	"github.com/aideveloper1976-arch/ollarouter/internal/provider"
	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

// toolCallProvider returns tool calls in the assistant response.
type toolCallProvider struct {
	ToolCalls []types.ToolCall
	Content   string
}

func (t *toolCallProvider) Name() string { return "test-provider" }
func (t *toolCallProvider) Type() string { return "ollama" }
func (t *toolCallProvider) Chat(ctx context.Context, req *types.ChatRequest) (*types.ChatResponse, error) {
	return &types.ChatResponse{
		Model: req.Model,
		Message: types.Message{
			Role:      "assistant",
			Content:   t.Content,
			ToolCalls: t.ToolCalls,
		},
	}, nil
}
func (t *toolCallProvider) ChatStream(ctx context.Context, req *types.ChatRequest, handler types.StreamHandler) error {
	return handler(&types.StreamChunk{Content: "ok", Done: true})
}
func (t *toolCallProvider) Generate(ctx context.Context, req *types.GenerateRequest) (*types.GenerateResponse, error) {
	return &types.GenerateResponse{Model: req.Model, Response: "ok"}, nil
}
func (t *toolCallProvider) GenerateStream(ctx context.Context, req *types.GenerateRequest, handler types.StreamHandler) error {
	return handler(&types.StreamChunk{Content: "ok", Done: true})
}
func (t *toolCallProvider) ListModels(ctx context.Context) ([]types.ModelInfo, error) {
	return []types.ModelInfo{{Name: "test-provider/gpt-4", ProviderName: "test-provider"}}, nil
}
func (t *toolCallProvider) CheckHealth(ctx context.Context) types.ProviderHealth {
	return types.ProviderHealth{Status: "online", LatencyMS: 1}
}

func TestV1ChatCompletions_NonStreaming_ToolCallContentPreserved(t *testing.T) {
	reg := provider.NewRegistry()
	p := &toolCallProvider{
		Content: "I'll check that for you",
		ToolCalls: []types.ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      "get_weather",
					Arguments: `{"location":"London"}`,
				},
			},
		},
	}
	if err := reg.Add(config.ProviderConfig{
		Name:    "test-provider",
		Type:    "ollama",
		BaseURL: "http://localhost:11434",
		Models: []config.ModelConfig{
			{Name: "gpt-4"},
		},
	}, p); err != nil {
		t.Fatalf("reg.Add() error = %v", err)
	}

	eng := engine.NewModelResolver(reg, map[string][]string{})
	app := &App{
		Registry: reg,
		Logger:   logger.New("error", "text"),
		Engine:   eng,
	}
	r := NewRouter(app)

	body := `{"model":"gpt-4","messages":[{"role":"user","content":"hello"}],"tools":[{"type":"function","function":{"name":"get_weather","description":"Get weather","parameters":{"type":"object"}}}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	choices, ok := resp["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		t.Fatal("expected choices array in response")
	}
	choice := choices[0].(map[string]interface{})
	msg := choice["message"].(map[string]interface{})

	// Content should be preserved (sanitized but not removed)
	if msg["content"] != "I'll check that for you" {
		t.Errorf("content = %q, want 'I'll check that for you'", msg["content"])
	}

	// Tool calls should be present
	tcs, ok := msg["tool_calls"].([]interface{})
	if !ok || len(tcs) == 0 {
		t.Fatal("expected tool_calls in response")
	}

	finishReason := choice["finish_reason"].(string)
	if finishReason != "tool_calls" {
		t.Errorf("finish_reason = %q, want tool_calls", finishReason)
	}
}

func TestGuardrailsEnabledInConfigParsing(t *testing.T) {
	yaml := `
providers:
  - name: local-models
    type: ollama
    base_url: http://ollama:11434
    auto_discover: true
    guardrails:
      enabled: true
      tool_call_validation: true
      rescue_malformed_tool_calls: true
      mixed_output_policy: text_first
      max_retries: 5
`
	cfg, err := config.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	p := cfg.Providers[0]
	if p.Guardrails == nil {
		t.Fatal("Guardrails config should not be nil")
	}
	if !p.Guardrails.Enabled {
		t.Errorf("Guardrails.Enabled = false, want true")
	}
	if !p.Guardrails.ToolCallValidation {
		t.Errorf("Guardrails.ToolCallValidation = false, want true")
	}
	if !p.Guardrails.RescueMalformed {
		t.Errorf("Guardrails.RescueMalformed = false, want true")
	}
	if p.Guardrails.MixedOutputPolicy != "text_first" {
		t.Errorf("Guardrails.MixedOutputPolicy = %q, want text_first", p.Guardrails.MixedOutputPolicy)
	}
	if p.Guardrails.MaxRetries != 5 {
		t.Errorf("Guardrails.MaxRetries = %d, want 5", p.Guardrails.MaxRetries)
	}
}

func TestGuardrailsDefaultWhenNotSpecified(t *testing.T) {
	yaml := `
providers:
  - name: test-openai
    type: openai
    api_key: sk-test
    models:
      - name: gpt-4o
`
	cfg, err := config.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Providers[0].Guardrails != nil {
		t.Errorf("Guardrails should be nil for openai provider, got %+v", cfg.Providers[0].Guardrails)
	}
}

func TestGuardrailsEnabledInRegistry(t *testing.T) {
	reg := provider.NewRegistry()

	cfg := config.ProviderConfig{
		Name:    "test-ollama",
		Type:    "ollama",
		BaseURL: "http://localhost:11434",
		Guardrails: &config.GuardrailsConfig{
			Enabled:            true,
			ToolCallValidation: true,
			RescueMalformed:    true,
			MixedOutputPolicy:  "tool_call_first",
			MaxRetries:         3,
		},
	}

	if err := reg.Add(cfg, &dummyProvider{}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	ps, ok := reg.Get("test-ollama")
	if !ok {
		t.Fatal("Get() returned not found")
	}

	if ps.Config.Guardrails == nil {
		t.Fatal("Guardrails should be propagated to provider state")
	}
	if !ps.Config.Guardrails.Enabled {
		t.Errorf("Guardrails.Enabled = false, want true")
	}
	if ps.Config.Guardrails.MixedOutputPolicy != "tool_call_first" {
		t.Errorf("Guardrails.MixedOutputPolicy = %q, want tool_call_first", ps.Config.Guardrails.MixedOutputPolicy)
	}
}

func TestV1ChatCompletions_GuardrailsEnabled_Ollama_ValidToolCall(t *testing.T) {
	reg := provider.NewRegistry()
	p := &toolCallProvider{
		Content: "Let me check the weather",
		ToolCalls: []types.ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      "get_weather",
					Arguments: `{"location":"London"}`,
				},
			},
		},
	}

	pCfg := config.ProviderConfig{
		Name:    "test-ollama",
		Type:    "ollama",
		BaseURL: "http://localhost:11434",
		Models: []config.ModelConfig{
			{Name: "gpt-4"},
		},
		Guardrails: &config.GuardrailsConfig{
			Enabled:            true,
			ToolCallValidation: true,
			RescueMalformed:    true,
			MixedOutputPolicy:  "tool_call_first",
			MaxRetries:         3,
		},
	}

	if err := reg.Add(pCfg, p); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	eng := engine.NewModelResolver(reg, map[string][]string{})
	app := &App{
		Registry: reg,
		Logger:   logger.New("error", "text"),
		Engine:   eng,
	}

	// Need to set up auth so the /v1 route is handled properly
	authCfg := &config.Config{
		Settings: config.DefaultSettings(),
	}
	authenticator := auth.NewAuthenticator(authCfg)
	app.Auth = authenticator
	app.Config = authCfg

	r := NewRouter(app)

	body := `{"model":"gpt-4","messages":[{"role":"user","content":"what is the weather?"}],"tools":[{"type":"function","function":{"name":"get_weather","description":"Get weather","parameters":{"type":"object"}}}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	// Tool calls should be present and valid
	choices := resp["choices"].([]interface{})
	msg := choices[0].(map[string]interface{})["message"].(map[string]interface{})
	tcs, ok := msg["tool_calls"].([]interface{})
	if !ok || len(tcs) == 0 {
		t.Fatal("expected tool_calls in response when guardrails enabled with valid tool calls")
	}
}
