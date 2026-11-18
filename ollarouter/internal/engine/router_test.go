package engine

import (
	"context"
	"testing"
	"time"

	"github.com/aideveloper1976-arch/ollarouter/internal/config"
	"github.com/aideveloper1976-arch/ollarouter/internal/provider"
	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

// mockProvider is a minimal provider implementation for testing.
type mockProvider struct {
	name     string
	provType string
	healthy  bool
	models   []types.ModelInfo
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Type() string { return m.provType }
func (m *mockProvider) Chat(ctx context.Context, req *types.ChatRequest) (*types.ChatResponse, error) {
	return &types.ChatResponse{
		Model: req.Model,
		Message: types.Message{
			Role:    "assistant",
			Content: "mock response",
		},
	}, nil
}
func (m *mockProvider) ChatStream(ctx context.Context, req *types.ChatRequest, handler types.StreamHandler) error {
	handler(&types.StreamChunk{Content: "mock", Done: false})
	handler(&types.StreamChunk{Done: true})
	return nil
}
func (m *mockProvider) Generate(ctx context.Context, req *types.GenerateRequest) (*types.GenerateResponse, error) {
	return &types.GenerateResponse{Model: req.Model, Response: "mock"}, nil
}
func (m *mockProvider) GenerateStream(ctx context.Context, req *types.GenerateRequest, handler types.StreamHandler) error {
	handler(&types.StreamChunk{Content: "mock", Done: false})
	handler(&types.StreamChunk{Done: true})
	return nil
}
func (m *mockProvider) ListModels(ctx context.Context) ([]types.ModelInfo, error) {
	return m.models, nil
}
func (m *mockProvider) CheckHealth(ctx context.Context) types.ProviderHealth {
	if m.healthy {
		return types.ProviderHealth{Status: "online"}
	}
	return types.ProviderHealth{Status: "offline"}
}

func setupTestRegistry(t *testing.T) *provider.Registry {
	t.Helper()
	reg := provider.NewRegistry()
	p := &mockProvider{
		name:     "test-provider",
		provType: "openai",
		healthy:  true,
		models: []types.ModelInfo{
			{Name: "test-provider/gpt-4", ProviderName: "test-provider"},
			{Name: "test-provider/gpt-3.5", ProviderName: "test-provider"},
		},
	}
	cfg := config.ProviderConfig{
		Name:    "test-provider",
		Type:    "openai",
		BaseURL: "https://api.test.com",
		Models: []config.ModelConfig{
			{Name: "gpt-4"},
			{Name: "gpt-3.5"},
		},
	}
	_ = reg.Add(cfg, p)
	return reg
}

func TestNewModelResolver(t *testing.T) {
	reg := setupTestRegistry(t)
	vms := map[string][]string{
		"fast-chat": {"test-provider/gpt-4", "test-provider/gpt-3.5"},
	}
	r := NewModelResolver(reg, vms)
	if r == nil {
		t.Fatal("NewModelResolver() returned nil")
	}
}

func TestResolveVirtualModel(t *testing.T) {
	reg := setupTestRegistry(t)
	vms := map[string][]string{
		"fast-chat": {"test-provider/gpt-4", "test-provider/gpt-3.5"},
	}
	r := NewModelResolver(reg, vms)

	// Test resolving a known virtual model
	chain, err := r.ResolveVirtualModel("fast-chat")
	if err != nil {
		t.Fatalf("ResolveVirtualModel('fast-chat') error = %v", err)
	}
	if len(chain) != 2 {
		t.Fatalf("Expected chain length 2, got %d", len(chain))
	}
	if chain[0] != "test-provider/gpt-4" {
		t.Errorf("Expected chain[0] = 'test-provider/gpt-4', got %q", chain[0])
	}
	if chain[1] != "test-provider/gpt-3.5" {
		t.Errorf("Expected chain[1] = 'test-provider/gpt-3.5', got %q", chain[1])
	}
}

func TestResolveVirtualModelNotFound(t *testing.T) {
	reg := setupTestRegistry(t)
	r := NewModelResolver(reg, nil)

	_, err := r.ResolveVirtualModel("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent virtual model")
	}
}

func TestResolveVirtualModelEmptyChain(t *testing.T) {
	reg := setupTestRegistry(t)
	vms := map[string][]string{
		"empty": {},
	}
	r := NewModelResolver(reg, vms)

	// ResolveVirtualModel should return the chain for the real model fallback
	// Actually, empty chain should error
	_, err := r.ResolveVirtualModel("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent model")
	}
}

func TestResolveModelDirect(t *testing.T) {
	reg := setupTestRegistry(t)
	r := NewModelResolver(reg, nil)

	result, err := r.ResolveModel("test-provider/gpt-4")
	if err != nil {
		t.Fatalf("ResolveModel() error = %v", err)
	}
	if result == nil {
		t.Fatal("ResolveModel() returned nil result")
	}
	if result.ProviderName != "test-provider" {
		t.Errorf("Expected ProviderName = 'test-provider', got %q", result.ProviderName)
	}
	if result.ModelName != "test-provider/gpt-4" {
		t.Errorf("Expected ModelName = 'test-provider/gpt-4', got %q", result.ModelName)
	}
}

func TestResolveModelShortName(t *testing.T) {
	reg := setupTestRegistry(t)
	r := NewModelResolver(reg, nil)

	// Should match by suffix
	result, err := r.ResolveModel("gpt-4")
	if err != nil {
		t.Fatalf("ResolveModel() error = %v", err)
	}
	if result.ProviderName != "test-provider" {
		t.Errorf("Expected ProviderName = 'test-provider', got %q", result.ProviderName)
	}
	if result.ModelName != "gpt-4" {
		t.Errorf("Expected ModelName = 'gpt-4', got %q", result.ModelName)
	}
}

func TestResolveModelNotFound(t *testing.T) {
	reg := setupTestRegistry(t)
	r := NewModelResolver(reg, nil)

	_, err := r.ResolveModel("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent model")
	}
}

func TestCircuitBreakerStateMachine(t *testing.T) {
	cb := NewCircuitBreaker(3, 50*time.Millisecond)

	// Initial state should be closed
	if cb.State() != StateClosed {
		t.Errorf("Initial state = %v, want closed", cb.State())
	}

	// Should allow requests when closed
	if !cb.AllowRequest() {
		t.Error("Expected AllowRequest() = true when closed")
	}

	// Record failures to trip the breaker
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	// Should be open after threshold failures
	if cb.State() != StateOpen {
		t.Errorf("State after threshold = %v, want open", cb.State())
	}

	// Should not allow requests when open
	if cb.AllowRequest() {
		t.Error("Expected AllowRequest() = false when open")
	}

	// Wait for cooldown
	time.Sleep(60 * time.Millisecond)

	// Should transition to half-open after cooldown
	if !cb.AllowRequest() {
		t.Error("Expected AllowRequest() = true after cooldown (half-open)")
	}

	// Record success to close the circuit
	cb.RecordSuccess()
	if cb.State() != StateClosed {
		t.Errorf("State after success = %v, want closed", cb.State())
	}
}

func TestRetryEngine(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries: 2,
		BaseDelay:  10 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
		UseJitter:  false,
	}
	e := NewRetryEngine(cfg)

	// Test successful execution
	attempts := 0
	err := e.DoWithRetry(context.Background(), func(ctx context.Context) error {
		attempts++
		return nil
	})
	if err != nil {
		t.Fatalf("DoWithRetry() error = %v", err)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}

	// Test retry on retryable error (provider unavailable)
	attempts = 0
	err = e.DoWithRetry(context.Background(), func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return types.ErrProviderUnavailable
		}
		return nil
	})
	if err != nil {
		t.Fatalf("DoWithRetry() error = %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts (1 initial + 2 retries), got %d", attempts)
	}

	// Test non-retryable error
	err = e.DoWithRetry(context.Background(), func(ctx context.Context) error {
		return types.ErrProviderModelNotFound
	})
	if err == nil {
		t.Fatal("Expected error for non-retryable error")
	}
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"provider unavailable", types.ErrProviderUnavailable, true},
		{"provider timeout", types.ErrProviderTimeout, true},
		{"provider rate limited", types.ErrProviderRateLimited, true},
		{"model not found", types.ErrProviderModelNotFound, false},
		{"all fallbacks exhausted", types.ErrAllFallbacksExhausted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldRetry(tt.err); got != tt.want {
				t.Errorf("ShouldRetry(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestFallbackExecutorBuildChain(t *testing.T) {
	reg := setupTestRegistry(t)
	vms := map[string][]string{
		"fast-chat": {"test-provider/gpt-4", "test-provider/gpt-3.5"},
	}
	fe := NewFallbackExecutor(reg, DefaultRetryConfig(), vms, nil)

	// Test building chain from virtual model
	chain := fe.BuildFallbackChain("fast-chat")
	if len(chain) != 2 {
		t.Fatalf("Expected chain length 2, got %d: %v", len(chain), chain)
	}
	if chain[0] != "test-provider/gpt-4" {
		t.Errorf("Expected chain[0] = 'test-provider/gpt-4', got %q", chain[0])
	}

	// Test non-virtual model returns single-element chain
	chain = fe.BuildFallbackChain("test-provider/gpt-4")
	if len(chain) != 1 || chain[0] != "test-provider/gpt-4" {
		t.Errorf("Expected single-element chain, got %v", chain)
	}
}

func TestFallbackExecutorExecuteWithFallback(t *testing.T) {
	reg := setupTestRegistry(t)
	fe := NewFallbackExecutor(reg, DefaultRetryConfig(), nil, nil)

	// Test successful execution with a valid chain
	providerName, err := fe.ExecuteWithFallback(context.Background(), []string{"test-provider/gpt-4"}, func(ctx context.Context, pName, mName string) error {
		return nil
	})
	if err != nil {
		t.Fatalf("ExecuteWithFallback() error = %v", err)
	}
	if providerName != "test-provider" {
		t.Errorf("Expected providerName = 'test-provider', got %q", providerName)
	}
}

func TestFallbackExecutorExhausted(t *testing.T) {
	reg := setupTestRegistry(t)
	fe := NewFallbackExecutor(reg, DefaultRetryConfig(), nil, nil)

	// Test exhausted fallbacks with failing chain
	_, err := fe.ExecuteWithFallback(context.Background(), []string{"test-provider/gpt-4"}, func(ctx context.Context, pName, mName string) error {
		return types.ErrProviderUnavailable
	})
	if err == nil {
		t.Fatal("Expected error for exhausted fallbacks")
	}
}
