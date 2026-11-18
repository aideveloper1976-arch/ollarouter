# Phase 6 — Architecture Review: OllaRouter

**Date:** 2026-05-20  
**Agent:** architect  
**Project:** ollama-stack (C:\Projects\ollama-stack)  
**Phase 5 Reference:** `docs/product/traceability/checkpoints/05-epic-feature-story-backlog.md`  
**Backlog Reference:** `docs/product/backlog/ollarouter-backlog.md`

---

## Summary

This architecture review covers the full OllaRouter system: a Dockerized multi-provider LLM gateway exposing an Ollama-compatible API that routes to OpenAI, Anthropic, OpenAI-compatible providers, local Ollama, and ChatGPT Plus. The review validates the existing key architecture decisions from research (Go language, NDJSON streaming, zero external DB, YAML config, port 11435, embedded web UI) and produces detailed API contracts, data model schemas, component architecture, provider adapter interfaces, Docker build strategy, security design, and observability design.

---

## Key Architecture Decisions

| # | Decision | Rationale | Risk | Status |
|---|----------|-----------|------|--------|
| ADR-001 | **Go 1.23** for implementation | Single static binary (~5-15MB), ms startup time, excellent HTTP/streaming/stdlib, cross-compile, ideal for Docker/distroless | Team must know Go | **Confirmed** |
| ADR-002 | **chi router** (not net/http mux or Gin) | Lightweight, idiomatic Go, middleware chaining, route groups, no heavy framework; Go 1.22 stdlib mux is insufficient for middleware patterns | None significant | **Confirmed** |
| ADR-003 | **Ollama API surface** (not OpenAI) | Core differentiator — no existing gateway exposes `/api/chat`, `/api/generate`, `/api/tags` | Clients must speak Ollama protocol | **Confirmed** |
| ADR-004 | **Provider adapter interface** | Pluggable providers: OpenAI, Anthropic, OpenAI-compatible, Ollama passthrough, ChatGPT Plus | Interface changes require adapter updates | **Confirmed** |
| ADR-005 | **NDJSON inbound → SSE/JSON outbound** | Ollama clients expect `application/x-ndjson`; commercial providers use SSE | Translation complexity | **Accepted** |
| ADR-006 | **Zero external DB** | In-memory state with optional file persistence; no PostgreSQL/Redis dependency | Data loss on restart (acceptable for gateway) | **Confirmed** |
| ADR-007 | **YAML config + env var interpolation** | Docker-native pattern, human-readable, supports `$(ENV_VAR)` for secrets | Config reload requires signal/restart (hot-reload deferred) | **Confirmed** |
| ADR-008 | **Embedded web UI via Go embed** | Single binary serves admin UI at `/admin/*`; no separate frontend server | Frontend assets bundled at build time | **Confirmed** |
| ADR-009 | **Token bucket rate limiter in-memory** | Per-key rate limiting without external Redis; sufficient for single-instance | Not suitable for multi-replica without shared state | **Accepted** |
| ADR-010 | **Port 11435** default | Alongside Ollama (11434), not replacing it | Users must configure OLLAMA_BASE_URL | **Confirmed** |

---

## Component Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        OllaRouter (Go binary)                       │
│                                                                     │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐   │
│  │   HTTP Router     │  │  Admin Web UI     │  │  Health/Metrics  │   │
│  │   (chi)           │  │  (Go embed SPA)   │  │  Endpoints       │   │
│  │                   │  │                   │  │                  │   │
│  │  /api/chat        │  │  /admin/*         │  │  GET /health     │   │
│  │  /api/generate    │  │  /api/admin/*     │  │  GET /health/providers │
│  │  /api/tags        │  │                   │  │  GET /metrics    │   │
│  │  /api/version     │  │                   │  │  GET /          │   │
│  │  /api/show        │  │                   │  │  (Ollama ping)   │   │
│  └────────┬─────────┘  └────────┬──────────┘  └────────┬─────────┘   │
│           │                     │                      │             │
│           ▼                     │                      │             │
│  ┌─────────────────────────────────────────────────┐   │             │
│  │            Middleware Stack                      │   │             │
│  │  ┌──────────┐ ┌──────────┐ ┌────────────────┐   │   │             │
│  │  │ Request  │ │ API Key  │ │ Rate Limiter   │   │   │             │
│  │  │ Logging  │ │ Auth     │ │ (Token Bucket) │   │   │             │
│  │  └──────────┘ └──────────┘ └────────────────┘   │   │             │
│  └─────────────────────┬───────────────────────────┘   │             │
│                        │                               │             │
│                        ▼                               │             │
│  ┌──────────────────────────────────────────────────────┐           │
│  │              Routing Engine                          │           │
│  │  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  │           │
│  │  │ Model Router  │  │ Fallback     │  │ Circuit    │  │           │
│  │  │ (model →     │  │ Chain        │  │ Breaker    │  │           │
│  │  │  provider)   │  │ Executor     │  │ (per-prov) │  │           │
│  │  └──────────────┘  └──────────────┘  └────────────┘  │           │
│  └─────────────────────┬────────────────────────────────┘           │
│                        │                                             │
│                        ▼                                             │
│  ┌──────────────────────────────────────────────────────┐           │
│  │         Provider Adapter Layer                        │           │
│  │                                                       │           │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ │           │
│  │  │  OpenAI   │ │ Anthropic│ │ Ollama   │ │ChatGPT   │ │           │
│  │  │  Adapter  │ │ Adapter  │ │ Passthru │ │ Plus     │ │           │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘ │           │
│  │  ┌──────────────────────────────────────────────────┐ │           │
│  │  │  OpenAI-Compatible Adapter (generic)              │ │           │
│  │  └──────────────────────────────────────────────────┘ │           │
│  └─────────────────────┬────────────────────────────────┘           │
│                        │                                             │
│                        ▼                                             │
│  ┌─────────────────────────────────────────────────────┐            │
│  │        Streaming Engine / Response Translator        │            │
│  │  ┌─────────────────┐  ┌────────────────────────┐    │            │
│  │  │ SSE → NDJSON    │  │ Error Normalizer       │    │            │
│  │  │ Translator      │  │ (Ollama error format)  │    │            │
│  │  └─────────────────┘  └────────────────────────┘    │            │
│  └─────────────────────────────────────────────────────┘            │
│                                                                     │
│  ┌──────────────────┐  ┌──────────────────┐                         │
│  │  Config Manager   │  │  Cost Tracker    │                        │
│  │  (YAML + env)     │  │  (in-memory)     │                        │
│  └──────────────────┘  └──────────────────┘                         │
│                                                                     │
│  ┌──────────────────┐                                               │
│  │  Logger          │                                               │
│  │  (structured     │                                               │
│  │   JSON output)   │                                               │
│  └──────────────────┘                                               │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Data Model / Config Schema

### providers.yaml

```yaml
# ============================================================
# OllaRouter — Provider Configuration
# ============================================================
# Providers define backend LLM services that OllaRouter can
# route requests to. Each provider has a type, connection
# details, and a list of models it serves.
#
# API keys should use ${ENV_VAR} references — never plaintext.
# ============================================================

# Global settings (optional)
settings:
  port: 11435                    # Gateway listen port (default 11435)
  admin_port: 11436              # Admin web UI port (default: same as port)
  log_level: info                # debug | info | warn | error
  log_format: json               # json | text
  prometheus_enabled: true       # Enable /metrics endpoint
  rate_limiting_enabled: true    # Global toggle
  default_max_retries: 3
  default_timeout_seconds: 30
  usage_persistence: memory      # memory | file
  usage_file_path: /data/usage.json

# API keys for client authentication
# Empty array = no auth required (open proxy)
api_keys:
  - name: admin-key
    key: "${OLLAROUTER_API_KEY}"    # Hashed/referenced at runtime
    rate_limit: 100                  # requests per minute
    monthly_budget_usd: 100.00
  - name: dev-key
    key: "${DEV_API_KEY}"

# Provider definitions
providers:
  # ----- Type: openai -----
  - name: openai-main
    type: openai
    base_url: https://api.openai.com/v1     # default
    api_key: "${OPENAI_API_KEY}"
    organization_id: ""                      # optional
    models:
      - name: gpt-4o
        max_tokens: 16384
      - name: gpt-4o-mini
        max_tokens: 16384
      - name: o3-mini
        max_tokens: 100000
    pricing:
      input_per_1k: 0.0025       # USD (gpt-4o input price)
      output_per_1k: 0.01        # USD (gpt-4o output price)

  # ----- Type: anthropic -----
  - name: anthropic-main
    type: anthropic
    api_key: "${ANTHROPIC_API_KEY}"
    api_version: "2023-06-01"
    models:
      - name: claude-sonnet-4
      - name: claude-3-5-sonnet-20241022
    pricing:
      input_per_1k: 0.003
      output_per_1k: 0.015

  # ----- Type: openai-compatible -----
  - name: groq
    type: openai-compatible
    base_url: https://api.groq.com/openai/v1
    api_key: "${GROQ_API_KEY}"
    models:
      - name: llama3-70b-8192
      - name: mixtral-8x7b-32768

  - name: azure-openai
    type: openai-compatible
    base_url: https://my-resource.openai.azure.com
    api_key: "${AZURE_OPENAI_KEY}"
    deployment_name: gpt-4o-deployment  # Azure deployment name
    models:
      - name: gpt-4o-azure

  # ----- Type: ollama -----
  - name: local-ollama
    type: ollama
    base_url: http://ollama:11434       # Docker service name
    auto_discover: true                  # Auto-discover models
    polling_interval_seconds: 60

  # ----- Type: chatgpt-plus -----
  - name: chatgpt-plus
    type: chatgpt-plus
    session_cookie: "${CHATGPT_SESSION_COOKIE}"
    user_agent: "Mozilla/5.0 ..."
    model_auto_discover: true
    polling_interval_seconds: 300

# ----- Virtual Models (named aliases with fallback chains) -----
# Virtual models let you define a logical model name that maps to
# a prioritized list of actual models/providers with fallback.
virtual_models:
  - name: fast-chat                # Friendly name clients use
    fallback_chain:
      - openai-main/gpt-4o-mini    # Primary: fastest/cheapest
      - groq/llama3-70b-8192       # Fallback 1: fast local-ish
      - local-ollama/llama3        # Fallback 2: local only
    timeout: 30

  - name: best-quality
    fallback_chain:
      - anthropic-main/claude-sonnet-4
      - openai-main/gpt-4o
      - local-ollama/mixtral
    timeout: 60

  - name: local-only
    fallback_chain:
      - local-ollama/*              # "*" = any model from that provider
```

### Runtime Config Data Structures (Go)

```go
// Config is the top-level parsed configuration.
type Config struct {
    Settings    Settings            `yaml:"settings"`
    APIKeys     []APIKeyConfig      `yaml:"api_keys"`
    Providers   []ProviderConfig    `yaml:"providers"`
    VirtualModels []VirtualModelConfig `yaml:"virtual_models"`
}

type Settings struct {
    Port                int    `yaml:"port" default:"11435"`
    AdminPort           int    `yaml:"admin_port"`
    LogLevel            string `yaml:"log_level" default:"info"`
    LogFormat           string `yaml:"log_format" default:"json"`
    PrometheusEnabled   bool   `yaml:"prometheus_enabled" default:"false"`
    RateLimitingEnabled bool   `yaml:"rate_limiting_enabled" default:"true"`
    DefaultMaxRetries   int    `yaml:"default_max_retries" default:"3"`
    DefaultTimeoutSec   int    `yaml:"default_timeout_seconds" default:"30"`
    UsagePersistence    string `yaml:"usage_persistence" default:"memory"`
    UsageFilePath       string `yaml:"usage_file_path" default:"/data/usage.json"`
}

type APIKeyConfig struct {
    Name             string  `yaml:"name"`
    Key              string  `yaml:"key"`       // After env var interpolation
    RateLimit        int     `yaml:"rate_limit"` // requests per minute, 0 = unlimited
    MonthlyBudgetUSD float64 `yaml:"monthly_budget_usd"`
}

type ProviderConfig struct {
    Name       string   `yaml:"name"`
    Type       string   `yaml:"type"` // openai | anthropic | openai-compatible | ollama | chatgpt-plus
    BaseURL    string   `yaml:"base_url"`
    APIKey     string   `yaml:"api_key"`     // After env var interpolation
    Models     []ModelConfig `yaml:"models"`
    Pricing    *PricingConfig `yaml:"pricing,omitempty"`
    // Type-specific fields
    OrganizationID   string `yaml:"organization_id,omitempty"`
    APIVersion       string `yaml:"api_version,omitempty"`
    DeploymentName   string `yaml:"deployment_name,omitempty"`
    AutoDiscover     bool   `yaml:"auto_discover,omitempty"`
    PollingInterval  int    `yaml:"polling_interval_seconds,omitempty"`
    SessionCookie    string `yaml:"session_cookie,omitempty"`
    UserAgent        string `yaml:"user_agent,omitempty"`
}

type ModelConfig struct {
    Name      string `yaml:"name"`
    MaxTokens int    `yaml:"max_tokens,omitempty"`
}

type PricingConfig struct {
    InputPer1K  float64 `yaml:"input_per_1k"`
    OutputPer1K float64 `yaml:"output_per_1k"`
}

type VirtualModelConfig struct {
    Name          string   `yaml:"name"`
    FallbackChain []string `yaml:"fallback_chain"` // e.g. ["openai-main/gpt-4o", "anthropic-main/claude-sonnet-4"]
    Timeout       int      `yaml:"timeout" default:"30"`
}

// Runtime provider state (not persisted in YAML)
type ProviderState struct {
    Config         ProviderConfig
    CircuitBreaker *CircuitBreaker
    ModelCache     []ModelInfo     // For auto-discovered models
    Health         ProviderHealth
}

type CircuitBreaker struct {
    State              string    // closed | open | half-open
    FailureCount       int
    LastFailureTime    time.Time
    CooldownPeriod     time.Duration
    FailureThreshold   int
}

type ProviderHealth struct {
    Status       string    // online | degraded | offline
    LastCheck    time.Time
    LatencyMS    int64
    ErrorMessage string
}
```

---

## API Contracts

### Inbound: Ollama-Compatible Endpoints

All endpoints follow the [Ollama API specification](https://github.com/ollama/ollama/blob/main/docs/api.md).

#### POST /api/chat

**Request:**
```json
{
  "model": "gpt-4o",
  "messages": [
    {"role": "user", "content": "Hello, how are you?"}
  ],
  "stream": true,
  "options": {
    "temperature": 0.7,
    "top_p": 0.9,
    "max_tokens": 2048
  }
}
```

**Response (stream=true):** `Content-Type: application/x-ndjson`
```
{"model":"gpt-4o","created_at":"2026-05-20T12:00:00Z","message":{"role":"assistant","content":"Hello"},"done":false}
{"model":"gpt-4o","created_at":"2026-05-20T12:00:01Z","message":{"role":"assistant","content":"! I'm doing great!"},"done":false}
{"model":"gpt-4o","created_at":"2026-05-20T12:00:02Z","message":{"role":"assistant","content":""},"done":true,"total_duration":2500000000,"load_duration":100000,"prompt_eval_count":12,"eval_count":18,"eval_duration":2200000000}
```

**Response (stream=false):** `Content-Type: application/json`
```json
{
  "model": "gpt-4o",
  "created_at": "2026-05-20T12:00:02Z",
  "message": { "role": "assistant", "content": "Hello! I'm doing great!" },
  "done": true,
  "total_duration": 2500000000,
  "load_duration": 100000,
  "prompt_eval_count": 12,
  "eval_count": 18,
  "eval_duration": 2200000000
}
```

#### POST /api/generate

**Request:**
```json
{
  "model": "claude-sonnet-4",
  "prompt": "Write a poem about Go programming",
  "system": "You are a poet.",
  "stream": true,
  "options": {
    "temperature": 0.8
  }
}
```

**Response (stream=true):** `Content-Type: application/x-ndjson`
```
{"model":"claude-sonnet-4","created_at":"...","response":"Go","done":false}
{"model":"claude-sonnet-4","created_at":"...","response":" is a language of grace","done":false}
{"model":"claude-sonnet-4","created_at":"...","response":"","done":true,"total_duration":...,"prompt_eval_count":5,"eval_count":42}
```

#### GET /api/tags

**Response:**
```json
{
  "models": [
    {
      "name": "openai-main/gpt-4o",
      "modified_at": "2026-01-01T00:00:00Z",
      "size": 0,
      "details": { "parameter_size": "unknown", "quantization_level": "unknown" }
    },
    {
      "name": "anthropic-main/claude-sonnet-4",
      "modified_at": "2026-01-01T00:00:00Z",
      "size": 0,
      "details": { "parameter_size": "unknown", "quantization_level": "unknown" }
    },
    {
      "name": "local-ollama/llama3:8b",
      "modified_at": "2026-05-19T10:00:00Z",
      "size": 4700000000,
      "details": { "parameter_size": "8B", "quantization_level": "Q4_K_M" }
    }
  ]
}
```

#### GET /api/version

```json
{
  "version": "0.1.0",
  "api_version": "1.0.0"
}
```

#### POST /api/show

```json
{
  "model": "openai-main/gpt-4o",
  "modified_at": "2026-01-01T00:00:00Z",
  "size": 0,
  "details": { "parameter_size": "unknown", "quantization_level": "unknown" },
  "provider": { "name": "openai-main", "type": "openai" }
}
```

### Admin API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | /admin/api/providers | List all providers with health status |
| POST | /admin/api/providers | Add a new provider |
| PUT | /admin/api/providers/:name | Update a provider |
| DELETE | /admin/api/providers/:name | Remove a provider |
| POST | /admin/api/providers/:name/test | Test provider connection |
| GET | /admin/api/providers/:name/health | Get provider health |
| GET | /admin/api/api-keys | List API keys |
| POST | /admin/api/api-keys | Create a new API key |
| DELETE | /admin/api/api-keys/:id | Revoke an API key |
| GET | /admin/api/stats/summary | Get usage summary (cards) |
| GET | /admin/api/stats/cost-by-provider | Cost breakdown |
| GET | /admin/api/stats/cost-by-model | Cost breakdown by model |
| GET | /admin/api/stats/cost-trend | Daily cost time series |
| GET | /admin/api/logs | Request log (paginated) |
| GET | /admin/api/settings | Get current settings |
| PUT | /admin/api/settings | Update settings |
| GET | /admin/api/config/yaml | Export runtime config as YAML |

### Health / Observability Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | /health | Service health (used by Docker HEALTHCHECK) |
| GET | /health/providers | Per-provider health status |
| GET | /metrics | Prometheus metrics format |

---

## Component Details

### 1. Provider Adapter Interface

```go
// Provider is the interface all provider adapters must implement.
type Provider interface {
    // Name returns the provider's configured name.
    Name() string

    // Type returns the provider type identifier (e.g., "openai", "anthropic").
    Type() string

    // Chat sends a chat completion request and returns the response.
    // If stream is true, the provider should return a streaming response
    // via the provided callback. If stream is false, a single response is returned.
    Chat(ctx context.Context, req *ChatRequest, stream bool) (*ChatResponse, error)

    // ChatStream sends a streaming chat completion request.
    // Each chunk is delivered to the handler callback.
    // The handler returns an error if the caller wants to abort streaming.
    ChatStream(ctx context.Context, req *ChatRequest, handler StreamHandler) error

    // Generate sends a completion request (mapped from Ollama /api/generate).
    Generate(ctx context.Context, req *GenerateRequest, stream bool) (*GenerateResponse, error)

    // GenerateStream sends a streaming completion request.
    GenerateStream(ctx context.Context, req *GenerateRequest, handler StreamHandler) error

    // ListModels returns the models available from this provider.
    ListModels(ctx context.Context) ([]ModelInfo, error)

    // CheckHealth checks if the provider is reachable and operational.
    CheckHealth(ctx context.Context) ProviderHealth
}

// StreamHandler is called for each streaming chunk.
// Return an error to abort streaming.
type StreamHandler func(chunk *StreamChunk) error

// ChatRequest is the internal normalized chat request format.
type ChatRequest struct {
    Model    string              // Provider's model name (already resolved)
    Messages []Message           // Normalized messages
    Options  *RequestOptions     // temperature, top_p, max_tokens, etc.
    Stream   bool
}

// GenerateRequest is the normalized generate request format.
type GenerateRequest struct {
    Model    string
    Prompt   string
    System   string
    Images   []string            // base64-encoded images
    Options  *RequestOptions
    Stream   bool
}

// StreamChunk represents a single streaming response chunk.
type StreamChunk struct {
    Content      string
    Done         bool
    TokenUsage   *TokenUsage      // Only populated on final chunk
    Error        error            // If provider error mid-stream
}

// TokenUsage tracks token counts per request.
type TokenUsage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}
```

### 2. Streaming Translation (SSE → NDJSON)

**OpenAI → Ollama NDJSON translation:**
```
OpenAI SSE chunks:
  data: {"choices":[{"delta":{"content":"Hello"},"index":0}]}
  data: {"choices":[{"delta":{"content":" world"},"index":0}]}
  data: [DONE]

  ↓ OllaRouter Streaming Engine ↓

Ollama NDJSON:
  {"model":"gpt-4o","created_at":"...","message":{"role":"assistant","content":"Hello"},"done":false}
  {"model":"gpt-4o","created_at":"...","message":{"role":"assistant","content":" world"},"done":false}
  {"model":"gpt-4o","created_at":"...","message":{"role":"assistant","content":""},"done":true,"total_duration":...,"prompt_eval_count":...,"eval_count":...}
```

**Anthropic → Ollama NDJSON translation:**
```
Anthropic SSE events:
  event: content_block_delta
  data: {"delta":{"type":"text_delta","text":"Hello"}}

  event: message_delta
  data: {"delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":42}}

  ↓ OllaRouter Streaming Engine ↓

Ollama NDJSON:
  {"model":"claude-sonnet-4","created_at":"...","message":{"role":"assistant","content":"Hello"},"done":false}
  {"model":"claude-sonnet-4","created_at":"...","message":{"role":"assistant","content":""},"done":true,"total_duration":...,"prompt_eval_count":...,"eval_count":...}
```

### 3. Retry / Fallback / Circuit Breaker Flow

```
Client Request
  │
  ▼
Model Router: resolve model → provider
  │
  ├── Virtual model? → expand to fallback chain
  │
  ▼
Provider Adapter: send request
  │
  ├── Success → return response
  │
  └── Error (429 / 5xx / network)
        │
        ▼
      Retry Engine
        │  exponential backoff: 1s → 2s → 4s
        │  respect Retry-After header if present
        │
        ├── Success → return response
        │
        └── All retries exhausted
              │
              ▼
            Circuit Breaker (per provider)
              │  Track consecutive failures
              │  Threshold (default 5) → OPEN state
              │  Cooldown (30s) → HALF-OPEN → test request
              │
              ▼
            Fallback Chain (if configured)
              │  Try next provider in chain
              │  Reset retry/circuit for new provider
              │
              ├── Success → return response
              │            Add header: x-ollarouter-provider: <fallback-name>
              │
              └── All fallbacks exhausted → return last error
```

### 4. Config Parser Flow

```
providers.yaml
  │
  ▼
Read file → YAML unmarshal
  │
  ▼
Env var interpolation:
  Walk all string fields, replace ${VAR} with os.Getenv("VAR")
  Missing vars → log warning, use empty string
  │
  ▼
Validation:
  - Provider name non-empty
  - API key present for non-Ollama providers
  - Valid provider type
  - No duplicate provider names (last wins with warning)
  - Virtual model fallback chains reference valid providers
  │
  ▼
Runtime Config
  │
  ▼
Provider initialization:
  - Create provider adapters
  - Initialize circuit breakers
  - Start health polling goroutines
  - Start model discovery polling (if auto_discover)
```

---

## Docker Architecture

### Multi-Stage Build

```dockerfile
# Stage 1: Build Go binary
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build admin UI (npm build would happen in a separate stage or be pre-built)
# For MVP: pre-built static assets are committed to repo under webui/dist/
COPY webui/dist ./webui/dist

# Build the Go application
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/ollarouter ./cmd/ollarouter

# Stage 2: Minimal runtime image
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /app/ollarouter /app/ollarouter
# Embedded static assets are compiled into the binary via go:embed

EXPOSE 11435

USER nonroot:nonroot

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/app/ollarouter", "healthcheck"]

ENTRYPOINT ["/app/ollarouter"]
```

### Docker Compose Integration

```yaml
version: '3.8'

services:
  ollama:
    image: ollama/ollama:latest
    container_name: ollama
    volumes:
      - ollama_data:/root/.ollama
    environment:
      - OLLAMA_HOST=0.0.0.0
      - OLLAMA_FLASH_ATTENTION=1
      - OLLAMA_NUM_PARALLEL=1
      - OLLAMA_MAX_LOADED_MODELS=1
      - OLLAMA_KEEP_ALIVE=30m
    ports:
      - "11434:11434"
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: all
              capabilities: [gpu]
    restart: unless-stopped

  ollarouter:
    build:
      context: .
      dockerfile: Dockerfile
    image: ollarouter:latest
    container_name: ollarouter
    ports:
      - "11435:11435"           # Ollama-compatible API
      - "11436:11436"           # Admin web UI (optional, can be same port)
    environment:
      - OLLAROUTER_PORT=11435
      - OLLAROUTER_ADMIN_PORT=11436
      - OLLAROUTER_CONFIG_PATH=/app/config/providers.yaml
      - OLLAROUTER_OLLAMA_HOST=http://ollama:11434
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - GROQ_API_KEY=${GROQ_API_KEY}
      - CHATGPT_SESSION_COOKIE=${CHATGPT_SESSION_COOKIE:-}
    volumes:
      - ./ollarouter/providers.yaml:/app/config/providers.yaml:ro
      - ollarouter_data:/data             # For usage persistence (optional)
    depends_on:
      - ollama
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "/app/ollarouter", "healthcheck"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 5s
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 256M

  open-webui:
    image: ghcr.io/open-webui/open-webui:main
    container_name: open-webui
    ports:
      - "3000:8080"
    volumes:
      - open_webui_data:/app/backend/data
    environment:
      - OLLAMA_BASE_URL=http://ollarouter:11435   # Routes through OllaRouter
      - WEBUI_AUTH=False
      - WEBUI_SECRET_KEY=${WEBUI_SECRET_KEY:-change-me}
    depends_on:
      - ollarouter
    restart: unless-stopped

volumes:
  ollama_data:
    external: false
  ollarouter_data:
    external: false
  open_webui_data:
    external: false
```

### Project Directory Structure

```
ollarouter/
├── cmd/
│   └── ollarouter/
│       └── main.go                    # Entry point
├── internal/
│   ├── config/
│   │   ├── config.go                  # Config struct definitions
│   │   ├── parser.go                  # YAML parsing + env interpolation
│   │   └── parser_test.go
│   ├── router/
│   │   ├── router.go                  # Chi router setup, middleware
│   │   ├── middleware.go              # Logging, auth, rate limit middleware
│   │   ├── handler_chat.go           # POST /api/chat handler
│   │   ├── handler_generate.go       # POST /api/generate handler
│   │   ├── handler_tags.go           # GET /api/tags handler
│   │   ├── handler_version.go        # GET /api/version handler
│   │   ├── handler_show.go           # POST /api/show handler
│   │   ├── handler_health.go         # GET /health, /health/providers
│   │   └── handler_metrics.go        # GET /metrics
│   ├── provider/
│   │   ├── provider.go               # Provider interface
│   │   ├── registry.go               # Provider registry (name → Provider)
│   │   ├── openai/
│   │   │   ├── adapter.go            # OpenAI provider adapter
│   │   │   └── adapter_test.go
│   │   ├── anthropic/
│   │   │   ├── adapter.go            # Anthropic provider adapter
│   │   │   └── adapter_test.go
│   │   ├── openaicompat/
│   │   │   ├── adapter.go            # OpenAI-compatible adapter
│   │   │   └── adapter_test.go
│   │   ├── ollama/
│   │   │   ├── adapter.go            # Local Ollama passthrough
│   │   │   └── adapter_test.go
│   │   └── chatgpt/
│   │       ├── adapter.go            # ChatGPT Plus web proxy
│   │       └── adapter_test.go
│   ├── streaming/
│   │   ├── translator.go             # SSE → NDJSON translation
│   │   ├── openai_translator.go      # OpenAI-specific SSE parsing
│   │   ├── anthropic_translator.go   # Anthropic-specific SSE parsing
│   │   ├── ollama_writer.go          # NDJSON writer (flush per line)
│   │   └── translator_test.go
│   ├── engine/
│   │   ├── router.go                 # Model → provider resolution
│   │   ├── fallback.go               # Fallback chain executor
│   │   ├── circuit_breaker.go        # Circuit breaker per provider
│   │   └── retry.go                  # Retry with exponential backoff
│   ├── auth/
│   │   ├── api_key.go                # API key validation
│   │   └── rate_limit.go             # Token bucket rate limiter
│   ├── tracking/
│   │   ├── counter.go                # Token + cost counting
│   │   ├── store.go                  # In-memory usage store
│   │   └── persistence.go            # Optional file persistence
│   ├── admin/
│   │   ├── handler_providers.go      # CRUD for providers
│   │   ├── handler_keys.go           # CRUD for API keys
│   │   ├── handler_stats.go          # Stats/cost dashboard API
│   │   ├── handler_logs.go           # Request log API
│   │   └── handler_settings.go       # Settings API
│   ├── webui/
│   │   └── embed.go                  //go:embed dist/* — admin SPA
│   └── logger/
│       └── logger.go                 # Structured JSON logger
├── webui/                            # Admin SPA source (separate package)
│   ├── src/
│   ├── package.json
│   └── dist/                         # Pre-built static assets
├── providers.yaml.example            # Example config
├── go.mod
├── go.sum
├── Dockerfile
└── Makefile
```

---

## Security Design

### 1. API Key Authentication

- **Bearer token** in `Authorization` header
- Configurable via `api_keys` in YAML (or env vars)
- Empty `api_keys` array = no auth (open proxy mode)
- Keys are hashed at rest using SHA-256; only hash stored in runtime memory
- All key values masked in logs: `sk-...xxxx`
- Key rotation: multiple active keys supported

### 2. Provider API Key Storage

- Provider API keys NEVER stored in YAML as plaintext
- All keys via `${ENV_VAR}` interpolation from environment
- Environment variables read from Docker Compose `.env` file or host environment
- In-memory only: keys not persisted to disk after loading

### 3. Rate Limiting

- Token bucket algorithm (in-memory, per key)
- Configurable requests-per-minute per key
- Burst allowance: 2x rate limit (allows short bursts)
- Global toggle: `rate_limiting_enabled`
- 429 response includes `Retry-After` header

### 4. Budget Enforcement

- Per-key monthly budget cap in USD
- Checked before each request (after auth)
- 429 with descriptive error when exceeded
- Budget resets at calendar month boundary

### 5. Security Headers

The admin web UI should set:
- `Content-Security-Policy`
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Strict-Transport-Security` (if HTTPS configured)

### 6. Docker Security

- **Non-root user** (`nonroot:nonroot` in distroless)
- **Read-only config mount** (`:ro`)
- No shell in runtime container (distroless)
- Config file excluded from image builds (`.dockerignore`)
- API keys via env vars, never in image layers

### 7. ChatGPT Plus Cookie Security

- Session cookie stored in memory only
- Cookie masked in logs
- Cookie never written to disk
- Automatic expiry detection (>48h warning)
- Cookie refresh on 403 response

---

## Observability Design

### 1. Structured JSON Logging

```json
{"level":"info","time":"2026-05-20T12:00:00Z","msg":"request completed","request_id":"req-abc123","method":"POST","path":"/api/chat","status":200,"duration_ms":1523,"provider":"openai-main","model":"gpt-4o","prompt_tokens":42,"completion_tokens":128,"api_key":"sk-...xxxx"}
```

Log levels: `debug`, `info`, `warn`, `error`
- `info`: request summary, provider health transitions
- `debug`: full request/response body summaries (masked)
- `warn`: circuit breaker state changes, degraded provider, missing env vars
- `error`: provider failures, retry exhaustion, auth failures

### 2. Health Endpoints

**GET /health** — Service health (Docker HEALTHCHECK compatible):
```json
{
  "status": "ok",
  "version": "0.1.0",
  "uptime_seconds": 86400,
  "providers_total": 5,
  "providers_online": 4,
  "providers_degraded": 1,
  "providers_offline": 0
}
```

**GET /health/providers** — Per-provider health:
```json
{
  "providers": [
    {
      "name": "openai-main",
      "type": "openai",
      "status": "online",
      "latency_ms": 450,
      "last_check": "2026-05-20T12:00:00Z",
      "circuit_breaker": "closed"
    },
    {
      "name": "local-ollama",
      "type": "ollama",
      "status": "degraded",
      "latency_ms": 5000,
      "last_check": "2026-05-20T11:59:30Z",
      "circuit_breaker": "half-open"
    }
  ]
}
```

### 3. Prometheus Metrics (GET /metrics)

```
# HELP ollarouter_requests_total Total requests processed
# TYPE ollarouter_requests_total counter
ollarouter_requests_total{provider="openai-main",model="gpt-4o",status="200"} 1452
ollarouter_requests_total{provider="anthropic-main",model="claude-sonnet-4",status="429"} 12

# HELP ollarouter_request_duration_seconds Request latency distribution
# TYPE ollarouter_request_duration_seconds histogram
ollarouter_request_duration_seconds_bucket{provider="openai-main",le="0.1"} 200
ollarouter_request_duration_seconds_bucket{provider="openai-main",le="0.5"} 800
ollarouter_request_duration_seconds_bucket{provider="openai-main",le="1"} 1200
ollarouter_request_duration_seconds_bucket{provider="openai-main",le="5"} 1440
ollarouter_request_duration_seconds_bucket{provider="openai-main",le="+Inf"} 1452
ollarouter_request_duration_seconds_sum{provider="openai-main"} 890.5
ollarouter_request_duration_seconds_count{provider="openai-main"} 1452

# HELP ollarouter_tokens_total Total tokens processed
# TYPE ollarouter_tokens_total counter
ollarouter_tokens_total{provider="openai-main",direction="input"} 58200
ollarouter_tokens_total{provider="openai-main",direction="output"} 185600

# HELP ollarouter_providers_active Current number of active providers
# TYPE ollarouter_providers_active gauge
ollarouter_providers_active 5

# HELP ollarouter_circuit_breaker_state Circuit breaker state (0=closed, 1=open, 2=half-open)
# TYPE ollarouter_circuit_breaker_state gauge
ollarouter_circuit_breaker_state{provider="openai-main"} 0
ollarouter_circuit_breaker_state{provider="local-ollama"} 2
```

### 4. Request ID Tracing

- Each request receives a unique `request_id` (ULID or UUIDv7)
- Propagated to provider requests via custom header (`X-OllaRouter-Request-ID`)
- Included in all log lines for correlation
- Exposed in admin request log for debugging

---

## Integration Impact

### With Existing ollama-stack

| Area | Impact | Migration |
|------|--------|-----------|
| docker-compose.yml | Add `ollarouter` service, update `open-webui` OLLAMA_BASE_URL | Add ~30 lines to compose file |
| Network | OllaRouter between ollama and open-webui | `depends_on` chain updated |
| Port allocation | 11434 (ollama) + 11435 (ollarouter) + 3000 (webui) | No port conflicts |

### With Open WebUI

- Change `OLLAMA_BASE_URL` from `http://ollama:11434` to `http://ollarouter:11435`
- All existing models still visible via `/api/tags` aggregation
- Cloud models appear as `provider/model-name` in model dropdown
- No Open WebUI code changes required

### With External Clients

- Any Ollama-compatible client (LangChain, LlamaIndex, Continue.dev, ollama CLI)
- Set `OLLAMA_HOST` or base URL to `http://ollarouter:11435`
- Use `provider/model-name` format for cloud models
- Virtual model names work as well (e.g., `fast-chat`, `best-quality`)

---

## Risks and Tradeoffs

| Risk | Severity | Mitigation |
|------|----------|------------|
| **ChatGPT Plus cookie expiry/detection** | High | Cookie health monitoring, auto-refresh, degraded mode warnings |
| **Streaming translation reliability** | High | Extensive unit tests on SSE→NDJSON; timeout after 30s idle stream |
| **In-memory state lost on restart** | Medium | Optional file persistence every 60s; acceptable for gateway use case |
| **Model name collision (local + cloud)** | Low | Provider prefix (`provider/model`); configurable alias in virtual models |
| **Config hot-reload complexity** | Low | Deferred to post-MVP; SIGHUP reload acceptable for v1.0 |
| **Single-instance rate limiter** | Medium | Sufficient for Docker/homelab; multi-replica requires shared state (future) |
| **OpenAI-compatible provider variations** | Medium | Generic adapter with per-provider overrides; test with top 3 (Azure, Groq, Together) |
| **Distroless image debugging** | Low | Use `gcr.io/distroless/static-debian12:debug` for troubleshooting |

---

## Implementation Plan (Build Order)

The build order follows dependency chain:

### Sprint 1: Core API (EPIC-1) — P0 MVP

1. **Project scaffold** — Go module, chi router, config parser, logger
2. **Provider interface + adapter skeleton** — Define `Provider` interface, empty implementations
3. **Config parser** — YAML parsing, env var interpolation, validation
4. **POST /api/chat** — Request parsing, model routing, provider adapter call, NDJSON streaming response (OpenAI adapter first)
5. **OpenAI provider adapter** — Full implementation with SSE→NDJSON translation
6. **POST /api/generate** — Generate format translation, Ollama→chat mapping
7. **GET /api/tags** — Aggregated model listing (static from config)
8. **Local Ollama passthrough** — Proxy to local Ollama, auto-discovery

### Sprint 2: Resilience (EPIC-1 continued)

9. **Retry engine** — Exponential backoff, Retry-After header support
10. **Fallback chain** — Virtual models, chain executor, `x-ollarouter-provider` header
11. **Circuit breaker** — Per-provider state machine, half-open probing

### Sprint 3: Admin Web UI (EPIC-2) — P1

12. **Admin API** — CRUD for providers, settings, config export
13. **Admin SPA scaffold** — React/Vite app, embedded via `go:embed`
14. **Provider list page** — Table, health status dots, actions
15. **Add/Edit provider wizard** — Type selection, connection form, model config
16. **Settings page** — Global settings, YAML preview

### Sprint 4: Auth & Rate Limiting (EPIC-3) — P1

17. **API key auth middleware** — Bearer token validation, hashing
18. **Token bucket rate limiter** — Per-key limits
19. **API key management UI** — Create/revoke keys in admin UI
20. **Budget cap enforcement** — Monthly budget tracking

### Sprint 5: Usage Tracking (EPIC-4) — P1

21. **Token counting** — Provider response parsing, estimation fallback
22. **Cost calculation** — Input/output pricing per model
23. **In-memory usage store** — Aggregation per provider/model/time
24. **File persistence** — Periodic write, load on startup
25. **Stats dashboard** — Summary cards, cost charts, request log
26. **CSV export** — Download usage data

### Sprint 6: ChatGPT Plus (EPIC-5) — P1 v1.1

27. **ChatGPT Plus adapter** — Cookie-based auth, conversation API proxy
28. **SSE→NDJSON for ChatGPT** — ChatGPT streaming format translation
29. **Model auto-detection** — Poll ChatGPT model list
30. **Cookie health monitoring** — Expiry detection, degraded/offline states
31. **Cookie auto-refresh** — 403 handling, retry logic

### Sprint 7: Observability (EPIC-6) — P1 v1.1

32. **Health endpoints** — `/health`, `/health/providers`
33. **Prometheus metrics** — Counter, histogram, gauge exports
34. **Structured JSON logging enhancements** — Debug-level body summaries

### Sprint 8: Extended Endpoints & Model Catalog (EPIC-7) — P2 v1.2

35. **GET /api/version** — Version info
36. **POST /api/show** — Model detail endpoint
37. **Model catalog web UI** — Searchable table, provider badges
38. **Model detail slide-over** — Pricing, context, fallback chain

---

## Open Design Questions

| # | Question | Suggested Resolution | Owner |
|---|----------|---------------------|-------|
| Q1 | Should admin web UI be on same port or separate port? | Same port (`/admin/*`) for simplicity; separate port optional | Architect |
| Q2 | File persistence format for usage data? | Append-only JSON lines (`/data/usage.ndjson`), rotated daily | Architect |
| Q3 | How to handle provider rate limit headers (OpenAI `x-ratelimit-*`)? | Log them, surface in health endpoint, potentially use for dynamic backoff | Architect |
| Q4 | Should model discovery for cloud providers be push or pull? | Pull (polling) for MVP; webhook push considered for future | Architect |
| Q5 | What is the maximum configurable timeout for provider requests? | Default 30s, max 300s, configurable per provider | Architect |

---

## Traceability

```
[c843c293] OllaRouter Idea
    └── derived_from ◀── [06-architecture-review] This Document
```

**Entity:** `06-architecture-review` — type: `decision`  
**Linked from:** EPIC-1 through EPIC-7 (all reference this architecture)

---

## Artifacts

| File | Description |
|------|-------------|
| `docs/product/traceability/checkpoints/06-architecture-review.md` | **This file** — Phase 6 checkpoint |
| `docs/product/architecture/01-system-architecture.md` | Full architecture document with complete API contracts, data models, component design |
| `docs/product/architecture/02-provider-adapter-contract.md` | Provider adapter interface specification |
| `docs/product/architecture/03-docker-architecture.md` | Docker multi-stage build and docker-compose design |

---

*End of Phase 6 — Architecture Review. Ready for Phase 7 Implementation Plan.*
