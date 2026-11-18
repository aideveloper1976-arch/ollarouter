# OllaRouter — Provider Adapter Contract

**Version:** 0.1.0  
**Date:** 2026-05-20  
**Status:** Approved

---

## 1. Provider Interface (Go)

```go
package provider

import (
    "context"
    "time"
)

// Provider is the interface all provider adapters must implement.
// Each adapter translates between Ollama format and its provider's API format.
type Provider interface {
    // Name returns the provider's configured name (e.g., "openai-main").
    Name() string

    // Type returns the provider type (e.g., "openai", "anthropic", "ollama").
    Type() string

    // Chat sends a chat completion request.
    // If stream is true, the provider must return streaming chunks via
    // the StreamHandler callback. The response will be nil for streaming calls.
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

    // ChatStream sends a streaming chat completion request.
    // Each content delta is delivered via the handler callback.
    // The final chunk will have Done=true and may contain TokenUsage.
    // Return handler error to abort streaming.
    ChatStream(ctx context.Context, req *ChatRequest, handler StreamHandler) error

    // Generate sends a completion request (mapped from Ollama /api/generate).
    Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)

    // GenerateStream sends a streaming completion request.
    GenerateStream(ctx context.Context, req *GenerateRequest, handler StreamHandler) error

    // ListModels returns the models available from this provider.
    // For cloud providers, returns the configured models from config.
    // For Ollama/ChatGPT Plus, returns auto-discovered models.
    ListModels(ctx context.Context) ([]ModelInfo, error)

    // CheckHealth verifies the provider is reachable and operational.
    // Returns health status with optional latency and error info.
    CheckHealth(ctx context.Context) ProviderHealth
}
```

---

## 2. Data Types

### 2.1 Request Types

```go
// ChatRequest is the normalized internal chat completion request.
type ChatRequest struct {
    // Model is the provider's model name (already resolved from routing).
    Model string `json:"model"`

    // Messages contains the conversation history.
    Messages []Message `json:"messages"`

    // Options contains inference parameters.
    Options *RequestOptions `json:"options,omitempty"`

    // Stream indicates whether to stream the response (non-streaming fallback).
    Stream bool `json:"stream"`
}

// Message represents a single message in the conversation.
type Message struct {
    Role     string    `json:"role"`              // "user", "assistant", "system", "tool"
    Content  string    `json:"content"`            // Text content
    Images   []string  `json:"images,omitempty"`   // Base64-encoded images (vision)
    ToolCalls []ToolCall `json:"tool_calls,omitempty"`
    ToolCallID string  `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
    ID       string `json:"id"`
    Type     string `json:"type"`     // "function"
    Function struct {
        Name      string `json:"name"`
        Arguments string `json:"arguments"` // JSON string
    } `json:"function"`
}

// RequestOptions maps to Ollama's "options" field.
type RequestOptions struct {
    Temperature float32  `json:"temperature,omitempty"`
    TopP        float32  `json:"top_p,omitempty"`
    TopK        int      `json:"top_k,omitempty"`
    MaxTokens   int      `json:"max_tokens,omitempty"`
    Stop        []string `json:"stop,omitempty"`
    FrequencyPenalty float32 `json:"frequency_penalty,omitempty"`
    PresencePenalty  float32 `json:"presence_penalty,omitempty"`
    Seed        int64   `json:"seed,omitempty"`
}

// GenerateRequest maps to Ollama's /api/generate format.
type GenerateRequest struct {
    Model   string           `json:"model"`
    Prompt  string           `json:"prompt"`
    System  string           `json:"system,omitempty"`
    Images  []string         `json:"images,omitempty"`
    Options *RequestOptions  `json:"options,omitempty"`
    Stream  bool             `json:"stream"`
}
```

### 2.2 Response Types

```go
// ChatResponse is the non-streaming chat completion response.
type ChatResponse struct {
    Model      string     `json:"model"`
    Message    Message    `json:"message"`
    TokenUsage *TokenUsage `json:"usage,omitempty"`
}

// GenerateResponse is the non-streaming generate completion response.
type GenerateResponse struct {
    Model      string     `json:"model"`
    Response   string     `json:"response"`
    TokenUsage *TokenUsage `json:"usage,omitempty"`
}

// StreamChunk represents a single streaming response chunk.
type StreamChunk struct {
    Content    string      `json:"content"`              // Next token(s)
    Done       bool        `json:"done"`                 // Is this the final chunk?
    TokenUsage *TokenUsage `json:"usage,omitempty"`      // Only on final chunk
    Metrics    *Metrics    `json:"metrics,omitempty"`    // Timing (final chunk only)
    Error      error       `json:"-"`                     // Provider error mid-stream
}

type Metrics struct {
    TotalDuration    time.Duration `json:"total_duration"`
    LoadDuration     time.Duration `json:"load_duration"`
    PromptEvalCount  int           `json:"prompt_eval_count"`
    EvalCount        int           `json:"eval_count"`
    EvalDuration     time.Duration `json:"eval_duration"`
}

type TokenUsage struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
    TotalTokens      int `json:"total_tokens"`
}
```

### 2.3 Provider State Types

```go
// StreamHandler is called for each streaming chunk.
// Return an error to abort the stream (e.g., client disconnected).
type StreamHandler func(chunk *StreamChunk) error

// ModelInfo represents a model available from a provider.
type ModelInfo struct {
    Name         string `json:"name"`
    ModifiedAt   string `json:"modified_at"`
    Size         int64  `json:"size"`     // 0 for cloud models
    Digest       string `json:"digest,omitempty"`
    Details      ModelDetails `json:"details,omitempty"`
    ProviderName string `json:"provider_name"` // Source provider
}

type ModelDetails struct {
    ParentModel       string `json:"parent_model,omitempty"`
    Format            string `json:"format,omitempty"`
    Family            string `json:"family,omitempty"`
    Families          []string `json:"families,omitempty"`
    ParameterSize     string `json:"parameter_size,omitempty"`
    QuantizationLevel string `json:"quantization_level,omitempty"`
}

// ProviderHealth represents the health status of a provider.
type ProviderHealth struct {
    Status       string    `json:"status"`        // "online", "degraded", "offline"
    LastCheck    time.Time `json:"last_check"`
    LatencyMS    int64     `json:"latency_ms"`
    ErrorMessage string    `json:"error_message,omitempty"`
}

// ProviderState holds the runtime state for a provider.
type ProviderState struct {
    Provider       Provider
    Config         ProviderConfig
    CircuitBreaker *CircuitBreaker
    ModelCache     []ModelInfo
    Health         ProviderHealth
    LastUsed       time.Time
}
```

---

## 3. Adapter Implementations

### 3.1 OpenAI Provider

| Property | Value |
|----------|-------|
| Type ID | `"openai"` |
| Default Base URL | `https://api.openai.com/v1` |
| Auth | `Authorization: Bearer {{api_key}}` |
| Chat Endpoint | `POST {{base_url}}/chat/completions` |
| Streaming | SSE (`data: {...}` events, `data: [DONE]` termination) |
| Model List | Static from config |
| Health Check | `GET {{base_url}}/models` (list models) |

**Request Translation (Ollama → OpenAI):**
```json
{
  "model": "gpt-4o",
  "messages": [
    {"role": "system", "content": "You are helpful"},
    {"role": "user", "content": "Hi"}
  ],
  "stream": true,
  "temperature": 0.7,
  "max_tokens": 2048
}
```

**Response Translation (OpenAI SSE → Ollama NDJSON):**

| OpenAI SSE | Ollama NDJSON |
|-----------|--------------|
| `data: {"choices":[{"delta":{"role":"assistant","content":"Hello"}}]}` | `{"message":{"role":"assistant","content":"Hello"},"done":false}` |
| `data: {"choices":[{"delta":{"content":" world"}}]}` | `{"message":{"role":"assistant","content":" world"},"done":false}` |
| `data: {"choices":[{"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":20}}` | `{"message":{"role":"assistant","content":""},"done":true,"prompt_eval_count":10,"eval_count":20}` |
| `data: [DONE]` | (ignored, final chunk already sent) |

**Error Translation:**
```json
// OpenAI error
{"error":{"message":"Incorrect API key","type":"authentication_error","code":"invalid_api_key"}}
// → Ollama error
{"error":"OpenAI: authentication failed (401)"}
```

### 3.2 Anthropic Provider

| Property | Value |
|----------|-------|
| Type ID | `"anthropic"` |
| Default Base URL | `https://api.anthropic.com/v1` |
| Auth | `x-api-key: {{api_key}}` + `anthropic-version: {{version}}` |
| Chat Endpoint | `POST {{base_url}}/messages` |
| Streaming | SSE (`content_block_start`, `content_block_delta`, `message_delta`) |
| Model List | Static from config |
| Health Check | `GET {{base_url}}/models` |

**Request Translation (Ollama → Anthropic):**
```json
{
  "model": "claude-sonnet-4",
  "max_tokens": 2048,
  "system": "You are helpful",
  "messages": [
    {"role": "user", "content": "Hi"}
  ],
  "stream": true,
  "temperature": 0.7
}
```

**Response Translation (Anthropic SSE → Ollama NDJSON):**

| Anthropic SSE | Ollama NDJSON |
|--------------|--------------|
| `event: content_block_start\ndata: {"content_block":{"type":"text","text":"Hello"}}` | `{"message":{"role":"assistant","content":"Hello"},"done":false}` |
| `event: content_block_delta\ndata: {"delta":{"type":"text_delta","text":" world"}}` | `{"message":{"role":"assistant","content":" world"},"done":false}` |
| `event: message_delta\ndata: {"delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":20}}` | `{"message":{"role":"assistant","content":""},"done":true,"eval_count":20}` |

### 3.3 OpenAI-Compatible Provider

| Property | Value |
|----------|-------|
| Type ID | `"openai-compatible"` |
| Default Base URL | Configurable (no default) |
| Auth | `Authorization: Bearer {{api_key}}` (configurable) |
| Chat Endpoint | `POST {{base_url}}/chat/completions` |
| Streaming | SSE (same as OpenAI, may vary) |
| Model List | Static from config |
| Health Check | `GET {{base_url}}/models` |

**Special behavior:**
- If streaming is not supported, falls back to non-streaming and buffers the full response
- Supports Azure OpenAI via `deployment_name` substitution in URL
- Error format is assumed to be OpenAI-like, falls back to generic parsing

### 3.4 Ollama Provider

| Property | Value |
|----------|-------|
| Type ID | `"ollama"` |
| Default Base URL | `http://ollama:11434` |
| Auth | None (internal network) |
| Chat Endpoint | `POST {{base_url}}/api/chat` |
| Generate Endpoint | `POST {{base_url}}/api/generate` |
| Tags Endpoint | `GET {{base_url}}/api/tags` |
| Streaming | NDJSON (passthrough, no translation needed) |
| Model List | Auto-discovered via polling `GET /api/tags` |

**Passthrough behavior:**
- Requests are forwarded as-is (already in Ollama format)
- Responses are returned as-is (already NDJSON)
- Only Connection refused (502) and timeout errors are wrapped
- Health check queries `GET /api/tags` with a 5s timeout

### 3.5 ChatGPT Plus Provider

| Property | Value |
|----------|-------|
| Type ID | `"chatgpt-plus"` |
| Default Base URL | `https://chatgpt.com` |
| Auth | `Cookie: __Secure-next-auth.session-token={{cookie}}` |
| Chat Endpoint | `POST {{base_url}}/backend-api/conversation` |
| Streaming | Custom SSE format |
| Model List | Auto-discovered via `GET {{base_url}}/backend-api/models` |
| Health Check | `GET {{base_url}}/api/auth/session` (verify cookie) |

**Request Translation (Ollama → ChatGPT API):**
```json
{
  "model": "gpt-5",
  "messages": [
    {"role": "user", "content": "Hi"}
  ],
  "stream": true
}
```

**ChatGPT SSE → Ollama NDJSON:**
The ChatGPT backend uses a different SSE format. The adapter must handle:
- `data: {"type":"response.text.delta","delta":"Hello"}` → content chunk
- `data: {"type":"response.text.done"}` → signal end of text
- `data: {"type":"message.complete","message":{"id":"...","usage":{...}}}` → final message with usage

**Cookie Management:**
- Cookie stored in memory only (never to disk)
- Health endpoint verifies cookie validity
- Degraded state when cookie < 48h from expiry
- Offline state when cookie expired
- On 403 response, attempt cookie refresh (or require manual reconfiguration)

---

## 4. Provider Registration

```go
// Registry manages all configured provider instances.
type Registry struct {
    providers map[string]*ProviderState
    mu        sync.RWMutex
}

func (r *Registry) Get(name string) (*ProviderState, bool)
func (r *Registry) Add(config ProviderConfig) error
func (r *Registry) Remove(name string)
func (r *Registry) Update(name string, config ProviderConfig) error
func (r *Registry) List() []*ProviderState

// AllProviders returns all providers (for model aggregation).
func (r *Registry) AllProviders() []Provider
```

---

## 5. Adapter Factory

```go
// NewProvider creates a provider adapter from the given config.
func NewProvider(config ProviderConfig) (Provider, error) {
    switch config.Type {
    case "openai":
        return openai.NewAdapter(config)
    case "anthropic":
        return anthropic.NewAdapter(config)
    case "openai-compatible":
        return openaicompat.NewAdapter(config)
    case "ollama":
        return ollama.NewAdapter(config)
    case "chatgpt-plus":
        return chatgpt.NewAdapter(config)
    default:
        return nil, fmt.Errorf("unknown provider type: %s", config.Type)
    }
}
```

---

## 6. Error Handling Contract

All providers must return errors that implement or wrap these sentinel errors:

```go
var (
    ErrProviderUnavailable   = errors.New("provider unavailable")
    ErrProviderTimeout       = errors.New("provider request timed out")
    ErrProviderAuthFailed    = errors.New("provider authentication failed")
    ErrProviderRateLimited   = errors.New("provider rate limited")
    ErrProviderModelNotFound = errors.New("provider model not found")
    ErrAllFallbacksExhausted = errors.New("all providers in fallback chain failed")
)
```

The error wrapper in `internal/router/` will translate these to Ollama-compatible JSON error responses.

---

*End of Provider Adapter Contract*
