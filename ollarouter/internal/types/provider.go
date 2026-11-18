package types

import (
	"context"
	"errors"
	"time"
)

// Sentinel errors.
var (
	ErrProviderUnavailable   = errors.New("provider unavailable")
	ErrProviderTimeout       = errors.New("provider request timed out")
	ErrProviderAuthFailed    = errors.New("provider authentication failed")
	ErrProviderRateLimited   = errors.New("provider rate limited")
	ErrProviderModelNotFound = errors.New("provider model not found")
	ErrAllFallbacksExhausted = errors.New("all providers in fallback chain failed")
)

// Provider is the interface all provider adapters must implement.
type Provider interface {
	Name() string
	Type() string
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	ChatStream(ctx context.Context, req *ChatRequest, handler StreamHandler) error
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
	GenerateStream(ctx context.Context, req *GenerateRequest, handler StreamHandler) error
	ListModels(ctx context.Context) ([]ModelInfo, error)
	CheckHealth(ctx context.Context) ProviderHealth
}

// BalanceProvider is implemented by providers that can report account balance
// or credits through their upstream APIs.
type BalanceProvider interface {
	GetBalance(ctx context.Context) (*ProviderBalance, error)
}

// StreamHandler is called for each streaming chunk.
type StreamHandler func(chunk *StreamChunk) error

// ToolDefinition describes a tool/function available to the model.
type ToolDefinition struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction describes a callable function tool.
type ToolFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

// ChatRequest is the normalized internal chat completion request.
type ChatRequest struct {
	Model    string           `json:"model"`
	Messages []Message        `json:"messages"`
	Tools    []ToolDefinition `json:"tools,omitempty"`
	Options  *RequestOptions  `json:"options,omitempty"`
	Stream   bool             `json:"stream"`
}

// Message represents a single message in the conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	// ReasoningContent carries the model's "thinking" output (e.g. DeepSeek
	// reasoner models). It must be echoed back on subsequent tool turns or the
	// upstream rejects the request, so it is preserved round-trip.
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	Images           []string   `json:"images,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
}

// ToolCall represents a function call from the model.
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// RequestOptions maps to Ollama's "options" field.
type RequestOptions struct {
	Temperature      float32  `json:"temperature,omitempty"`
	TopP             float32  `json:"top_p,omitempty"`
	TopK             int      `json:"top_k,omitempty"`
	MaxTokens        int      `json:"max_tokens,omitempty"`
	Stop             []string `json:"stop,omitempty"`
	FrequencyPenalty float32  `json:"frequency_penalty,omitempty"`
	PresencePenalty  float32  `json:"presence_penalty,omitempty"`
	Seed             int64    `json:"seed,omitempty"`
}

// GenerateRequest maps to Ollama's /api/generate format.
type GenerateRequest struct {
	Model   string          `json:"model"`
	Prompt  string          `json:"prompt"`
	System  string          `json:"system,omitempty"`
	Images  []string        `json:"images,omitempty"`
	Options *RequestOptions `json:"options,omitempty"`
	Stream  bool            `json:"stream"`
}

// ChatResponse is the non-streaming chat completion response.
type ChatResponse struct {
	Model      string      `json:"model"`
	Message    Message     `json:"message"`
	TokenUsage *TokenUsage `json:"usage,omitempty"`
}

// GenerateResponse is the non-streaming generate completion response.
type GenerateResponse struct {
	Model      string      `json:"model"`
	Response   string      `json:"response"`
	TokenUsage *TokenUsage `json:"usage,omitempty"`
}

// StreamChunk represents a single streaming response chunk.
type StreamChunk struct {
	Content          string      `json:"content"`
	ReasoningContent string      `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall  `json:"tool_calls,omitempty"`
	Done             bool        `json:"done"`
	TokenUsage       *TokenUsage `json:"usage,omitempty"`
	Metrics          *Metrics    `json:"metrics,omitempty"`
	Error            error       `json:"-"`
}

// Metrics holds timing information for the request.
type Metrics struct {
	TotalDuration   time.Duration `json:"total_duration"`
	LoadDuration    time.Duration `json:"load_duration"`
	PromptEvalCount int           `json:"prompt_eval_count"`
	EvalCount       int           `json:"eval_count"`
	EvalDuration    time.Duration `json:"eval_duration"`
}

// TokenUsage holds token count information.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ModelInfo represents a model available from a provider.
type ModelInfo struct {
	Name         string       `json:"name"`
	ModifiedAt   string       `json:"modified_at"`
	Size         int64        `json:"size"`
	Digest       string       `json:"digest,omitempty"`
	Details      ModelDetails `json:"details,omitempty"`
	ProviderName string       `json:"provider_name"`
}

// ModelDetails holds additional model metadata.
type ModelDetails struct {
	ParentModel       string   `json:"parent_model,omitempty"`
	Format            string   `json:"format,omitempty"`
	Family            string   `json:"family,omitempty"`
	Families          []string `json:"families,omitempty"`
	ParameterSize     string   `json:"parameter_size,omitempty"`
	QuantizationLevel string   `json:"quantization_level,omitempty"`
}

// ProviderHealth represents the health status of a provider.
type ProviderHealth struct {
	Status       string    `json:"status"`
	LastCheck    time.Time `json:"last_check"`
	LatencyMS    int64     `json:"latency_ms"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// ProviderState holds the runtime state for a provider.
type ProviderState struct {
	Provider       Provider
	Config         ProviderConfig
	CircuitBreaker *CircuitBreakerStatus
	ModelCache     []ModelInfo
	Health         ProviderHealth
	LastUsed       time.Time
}

// GuardrailsConfig holds tool call guardrails settings for a provider.
type GuardrailsConfig struct {
	Enabled            bool
	ToolCallValidation bool
	RescueMalformed    bool
	MixedOutputPolicy  string
	MaxRetries         int
}

// ProviderConfig holds provider configuration.
type ProviderConfig struct {
	Name             string
	Type             string
	BaseURL          string
	APIKey           string
	OrganizationID   string
	AnthropicVersion string
	AutoDiscover     bool
	PollingInterval  int
	Models           []ModelConfig
	Pricing          *PricingConfig
	MaxRetries       int
	TimeoutSec       int
	Headers          map[string]string
	Guardrails       *GuardrailsConfig
	SessionCookie    string
	UserAgent        string
	Auth             *AuthSettings
}

// AuthSettings defines the upstream authentication configuration for a provider.
type AuthSettings struct {
	Mode            string // "oauth" or "session_cookie" or "none"
	ClientID        string
	ClientSecret    string
	RedirectURI     string
	CallbackPort    int
	KeychainService string
	TokenEndpoint   string
	AuthEndpoint    string
	Scopes          string
}

// ModelConfig is a simplified model configuration.
type ModelConfig struct {
	Name       string
	MaxTokens  int
	ContextLen int
}

// PricingConfig holds pricing info.
type PricingConfig struct {
	InputPer1K  float64
	OutputPer1K float64
}

// ProviderBalance holds provider-reported credit or usage balance data.
type ProviderBalance struct {
	Currency         string  `json:"currency,omitempty"`
	TotalCredits     float64 `json:"total_credits,omitempty"`
	TotalUsage       float64 `json:"total_usage,omitempty"`
	RemainingCredits float64 `json:"remaining_credits,omitempty"`
	Source           string  `json:"source,omitempty"`
}

// CircuitBreakerStatus holds the state of a circuit breaker.
type CircuitBreakerStatus struct {
	State    string    `json:"state"`
	Failures int       `json:"failures"`
	LastOpen time.Time `json:"last_open_at"`
}
