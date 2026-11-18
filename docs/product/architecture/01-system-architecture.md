# OllaRouter — System Architecture

**Version:** 0.1.0  
**Date:** 2026-05-20  
**Status:** Approved (Phase 6 Architecture Review)

---

## 1. Architecture Overview

OllaRouter is a **multi-provider LLM gateway** that exposes an Ollama-compatible API surface and routes requests to backend LLM providers. It is designed as a **single Go binary** with zero external dependencies, deployable as a Docker container alongside the existing ollama-stack.

### 1.1 Core Design Principles

| Principle | Rationale |
|-----------|-----------|
| **Ollama API-first** | Serve Ollama-compatible clients (Open WebUI, LangChain, ollama CLI) without protocol adapters |
| **Provider-agnostic routing** | Pluggable provider adapters with a common interface |
| **Resilient by default** | Retry with exponential backoff, fallback chains, circuit breakers |
| **Observable** | Structured JSON logging, Prometheus metrics, health endpoints |
| **Zero external dependencies** | No PostgreSQL, Redis, or other services |
| **Single binary deployment** | Embedded web UI, compile-time asset bundling |

### 1.2 Technology Stack

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| Language | **Go 1.23+** | Single static binary, excellent stdlib, fast compilation, cross-compile |
| HTTP Router | **chi** (go-chi/chi/v5) | Lightweight, idiomatic middleware chaining, route groups |
| Config | **YAML** (gopkg.in/yaml.v3) | Industry standard, Docker-native, human-readable |
| Logging | **slog** (Go stdlib) | Structured logging built into Go 1.21+, zero dependencies |
| Metrics | **Prometheus** (prometheus/client_golang) | Industry standard metrics exposition format |
| Admin SPA | **React + Vite + Tailwind CSS + shadcn/ui** | Modern UI framework, bundles to static assets |
| Container | **distroless static** | Minimal attack surface, non-root user |
| Streaming | **NDJSON** inbound, **SSE** outbound | Ollama protocol compatible |

---

## 2. Component Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            OllaRouter (Go binary)                           │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        HTTP Layer (chi)                             │   │
│  │                                                                     │   │
│  │  Public API Router                     Admin API Router             │   │
│  │  ┌────────────────────────┐           ┌────────────────────────┐   │   │
│  │  │ POST /api/chat         │           │ POST /admin/api/       │   │   │
│  │  │ POST /api/generate     │           │   providers           │   │   │
│  │  │ GET  /api/tags         │           │ GET /admin/api/        │   │   │
│  │  │ POST /api/show         │           │   providers           │   │   │
│  │  │ GET  /api/version      │           │ GET /admin/api/stats/* │   │   │
│  │  │ GET  /health           │           │ GET /admin/api/logs    │   │   │
│  │  │ GET  /health/providers │           │ GET /admin/api/config  │   │   │
│  │  │ GET  /metrics          │           │ ...                    │   │   │
│  │  └──────────┬─────────────┘           └───────────┬────────────┘   │   │
│  │             │                                     │                 │   │
│  │  ┌──────────┴─────────────────────────────────────┴─────────────┐   │   │
│  │  │                    Middleware Chain                           │   │   │
│  │  │  ┌──────────┐ ┌──────────────┐ ┌────────────────┐          │   │   │
│  │  │  │ Request  │ │ API Key      │ │ Rate Limiter   │          │   │   │
│  │  │  │ ID       │ │ Auth (Bearer)│ │ (Token Bucket) │          │   │   │
│  │  │  └──────────┘ └──────────────┘ └────────────────┘          │   │   │
│  │  │  ┌──────────┐ ┌──────────────┐ ┌────────────────┐          │   │   │
│  │  │  │ Request  │ │ Recovery     │ │ CORS           │          │   │   │
│  │  │  │ Logger   │ │ (panic)      │ │ (admin only)   │          │   │   │
│  │  │  └──────────┘ └──────────────┘ └────────────────┘          │   │   │
│  │  └────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                      │                                      │
│  ┌───────────────────────────────────┴───────────────────────────────────┐  │
│  │                          Routing Engine                               │  │
│  │                                                                       │  │
│  │  ┌────────────────────┐    ┌────────────────────┐                     │  │
│  │  │ Model Resolver     │    │ Fallback Executor   │                    │  │
│  │  │                    │    │                     │                    │  │
│  │  │ model name →       │    │ Try provider 1      │                    │  │
│  │  │  provider + model  │───▶│ Try provider 2      │──▶ response        │  │
│  │  │ virtual model →    │    │ Try provider N      │                    │  │
│  │  │  fallback chain    │    │                     │                    │  │
│  │  └────────────────────┘    └──────────┬──────────┘                    │  │
│  │                                       │                                │  │
│  │  ┌────────────────────┐               │                                │  │
│  │  │ Circuit Breaker    │◀──────────────┘                                │  │
│  │  │ (per provider)     │  tracks failures, opens circuit               │  │
│  │  └────────────────────┘                                                │  │
│  └────────────────────────────────────────────────────────────────────────┘  │
│                                      │                                      │
│  ┌───────────────────────────────────┴───────────────────────────────────┐  │
│  │                       Provider Adapter Layer                          │  │
│  │                                                                       │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────┐ │  │
│  │  │  OpenAI  │  │Anthropic │  │OpenAI-   │  │  Ollama  │  │ChatGPT │ │  │
│  │  │          │  │          │  │Compat    │  │Passthru  │  │ Plus   │ │  │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬────┘ │  │
│  │       │              │             │             │              │      │  │
│  │       │  ┌───────────┴─────────────┴─────────────┴──────────────┘      │  │
│  │       │  │               Provider Interface                             │  │
│  │       │  │  Chat(), ChatStream(), Generate(), GenerateStream(),        │  │
│  │       │  │  ListModels(), CheckHealth()                                 │  │
│  │       └──┴───────────────────────────────────────────────────────────┘  │
│  └────────────────────────────────────────────────────────────────────────┘  │
│                                      │                                      │
│  ┌───────────────────────────────────┴───────────────────────────────────┐  │
│  │                       Streaming Engine                                │  │
│  │                                                                       │  │
│  │  ┌─────────────────────────────────────────────────────────────┐     │  │
│  │  │  SSE Parser ←→ NDJSON Writer                               │      │  │
│  │  │  - OpenAI: data: {...} events → {"message":{...}} lines    │      │  │
│  │  │  - Anthropic: content_block_delta events → same format      │      │  │
│  │  │  - ChatGPT: custom SSE format → normalized NDJSON            │      │  │
│  │  │  - Ollama: passthrough (already NDJSON)                     │      │  │
│  │  └─────────────────────────────────────────────────────────────┘      │  │
│  └────────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  ┌──────────────────────┐  ┌──────────────────────┐                         │
│  │    Config Manager     │  │   Cost Tracker        │                        │
│  │                      │  │                      │                        │
│  │  - YAML file parser   │  │  - Token counting     │                        │
│  │  - Env var interp     │  │  - Cost calculation   │                        │
│  │  - Validation         │  │  - In-memory store    │                        │
│  │  - Runtime reload     │  │  - File persistence   │                        │
│  └──────────────────────┘  └──────────────────────┘                         │
│                                                                             │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │  Logger (slog, structured JSON output)                                │  │
│  │  - Request ID correlation                                             │  │
│  │  - Provider, model, status, duration per log line                    │  │
│  │  - API key masking                                                    │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Data Model

### 3.1 Configuration Schema

The complete configuration is defined in a single YAML file (`providers.yaml`) with environment variable interpolation for secrets.

**See complete schema:** `docs/product/architecture/01-system-architecture.md#appendix-a-config-schema`

### 3.2 Runtime State (in-memory)

| State | Type | Persistence | Details |
|-------|------|-------------|---------|
| Provider instances | `map[string]Provider` | None | Created at startup from config |
| Circuit breaker states | `map[string]*CircuitBreaker` | None | Reset on restart |
| Health status | `map[string]ProviderHealth` | None | Polled every 30s |
| Model cache | `map[string][]ModelInfo` | None | Auto-discovered models |
| Usage data | `UsageStore` | Optional file (60s interval) | Token counts, costs |
| API key hashes | `map[string]*APIKeyInfo` | None | Loaded at startup |
| Rate limit buckets | `map[string]*TokenBucket` | None | In-memory only |

### 3.3 API Data Types

**See complete API contracts:** `docs/product/architecture/01-system-architecture.md#section-5`

---

## 4. Provider Adapter Contract

Every provider adapter implements the `Provider` interface:

### 4.1 Provider Interface

```go
// Provider is the interface all provider adapters must implement.
type Provider interface {
    Name() string
    Type() string
    Chat(ctx context.Context, req *ChatRequest, stream bool) (*ChatResponse, error)
    ChatStream(ctx context.Context, req *ChatRequest, handler StreamHandler) error
    Generate(ctx context.Context, req *GenerateRequest, stream bool) (*GenerateResponse, error)
    GenerateStream(ctx context.Context, req *GenerateRequest, handler StreamHandler) error
    ListModels(ctx context.Context) ([]ModelInfo, error)
    CheckHealth(ctx context.Context) ProviderHealth
}

type StreamHandler func(chunk *StreamChunk) error

type ChatRequest struct {
    Model    string
    Messages []Message
    Options  *RequestOptions
    Stream   bool
}

type StreamChunk struct {
    Content    string
    Done       bool
    TokenUsage *TokenUsage
    Error      error
}
```

### 4.2 Provider Adapter Implementations

| Adapter | Outbound API | Streaming | Auth | Notes |
|---------|-------------|-----------|------|-------|
| **OpenAI** | `POST /v1/chat/completions` | SSE `data: {...}` | Bearer token `Authorization: Bearer {{key}}` | Organization ID header supported |
| **Anthropic** | `POST /v1/messages` | SSE `content_block_delta`, `message_delta` | `x-api-key` header + `anthropic-version` | Role validation (user/assistant only) |
| **OpenAI-Compatible** | `POST {{base_url}}/chat/completions` | SSE (configurable) | Bearer token | Works with Azure, Groq, Together, vLLM |
| **Ollama** | `POST /api/chat`, `POST /api/generate`, `GET /api/tags` | NDJSON (passthrough) | None (internal network) | Proxies to local Ollama instance |
| **ChatGPT Plus** | `POST /backend-api/conversation` | SSE (ChatGPT format) | Session cookie (`__Secure-next-auth.session-token`) | Cloudflare PoW solver for challenges |

---

## 5. API Contracts

### 5.1 Inbound Endpoints (Ollama-Compatible)

| Method | Path | Description | Stream Support |
|--------|------|-------------|----------------|
| POST | `/api/chat` | Multi-turn chat | ✅ NDJSON |
| POST | `/api/generate` | Single-prompt generate | ✅ NDJSON |
| GET | `/api/tags` | List available models | ❌ |
| POST | `/api/show` | Model details | ❌ |
| GET | `/api/version` | Server version | ❌ |

**Full request/response formats:** See the Ollama API specification at https://github.com/ollama/ollama/blob/main/docs/api.md

### 5.2 Admin API (Web UI Backend)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/admin/api/providers` | List providers with health |
| POST | `/admin/api/providers` | Add provider |
| PUT | `/admin/api/providers/:name` | Update provider |
| DELETE | `/admin/api/providers/:name` | Delete provider |
| POST | `/admin/api/providers/:name/test` | Test connection |
| GET | `/admin/api/api-keys` | List API keys |
| POST | `/admin/api/api-keys` | Create API key |
| DELETE | `/admin/api/api-keys/:id` | Revoke API key |
| GET | `/admin/api/stats/summary` | Dashboard summary cards |
| GET | `/admin/api/stats/cost-by-provider` | Cost breakdown |
| GET | `/admin/api/stats/cost-trend` | Daily cost trend |
| GET | `/admin/api/logs` | Request log (paginated) |
| GET | `/admin/api/settings` | Get settings |
| PUT | `/admin/api/settings` | Update settings |
| GET | `/admin/api/config/yaml` | Export runtime YAML |

### 5.3 Health / Metrics

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Service health (Docker HEALTHCHECK) |
| GET | `/health/providers` | Per-provider health status |
| GET | `/metrics` | Prometheus metrics |

### 5.4 Error Format (Ollama-Compatible)

All errors use the Ollama error format:
```json
{
  "error": "descriptive error message"
}
```

HTTP status codes:
- `400` — Bad request (invalid JSON, missing fields)
- `401` — Authentication required / invalid API key
- `404` — Model not found
- `429` — Rate limited / budget exceeded
- `502` — Provider unreachable
- `503` — All providers unavailable / circuit breaker open

---

## 6. Streaming Flow

### 6.1 Streaming Chat (stream=true)

```
Client                        OllaRouter                       Provider
  │                              │                                │
  │  POST /api/chat              │                                │
  │  {"model":"gpt-4o",          │                                │
  │   "stream":true}             │                                │
  │─────────────────────────────▶│                                │
  │                              │  Resolve model → provider      │
  │                              │  Translate → ChatRequest       │
  │                              │  POST /v1/chat/completions     │
  │                              │  {"stream":true,...}           │
  │                              │───────────────────────────────▶│
  │                              │                                │
  │  ◀── NDJSON ──────────────  │  ◀── SSE ────────────────────  │
  │  {"message":{"role":         │  data: {"choices":[{"delta":   │
  │   "assistant",               │   {"content":"Hello"}}]}       │
  │   "content":"Hello"},        │                                │
  │   "done":false}              │                                │
  │                              │                                │
  │  ◀── NDJSON ──────────────  │  ◀── SSE ────────────────────  │
  │  {"message":{...},           │  data: {"choices":[{"delta":   │
  │   "done":false}              │   {"content":" world"}}]}      │
  │                              │                                │
  │  ◀── NDJSON ──────────────  │  ◀── SSE ────────────────────  │
  │  {"done":true,               │  data: [DONE]                  │
  │   "total_duration":...,      │                                │
  │   "eval_count":...}          │                                │
```

### 6.2 Translation Map: OpenAI SSE → Ollama NDJSON

| OpenAI SSE Event | Ollama NDJSON Line |
|-----------------|-------------------|
| `data: {"choices":[{"delta":{"role":"assistant","content":"Hello"}}]}` | `{"model":"...","created_at":"...","message":{"role":"assistant","content":"Hello"},"done":false}` |
| `data: [DONE]` | `{"model":"...","created_at":"...","message":{"role":"assistant","content":""},"done":true,"total_duration":...,"eval_count":...}` |

### 6.3 Translation Map: Anthropic SSE → Ollama NDJSON

| Anthropic SSE Event | Ollama NDJSON Line |
|--------------------|-------------------|
| `event: content_block_start` + `data: {"content_block":{"type":"text","text":"Hel"}}` | `{"message":{"role":"assistant","content":"Hel"},"done":false}` |
| `event: content_block_delta` + `data: {"delta":{"type":"text_delta","text":"lo"}}` | `{"message":{"role":"assistant","content":"lo"},"done":false}` |
| `event: message_delta` + `data: {"delta":{"stop_reason":"end_turn"},"usage":{...}}` | `{"done":true,"total_duration":...,"eval_count":...}` |

---

## 7. State Flow Diagrams

### 7.1 Request Lifecycle

```
┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐
│  Client  │   │  Router  │   │  Auth    │   │  Engine  │   │ Provider │
└────┬─────┘   └────┬─────┘   └────┬─────┘   └────┬─────┘   └────┬─────┘
     │              │              │              │              │
     │ POST /api/   │              │              │              │
     │ chat         │              │              │              │
     │─────────────▶│              │              │              │
     │              │ Assign       │              │              │
     │              │ request_id   │              │              │
     │              │──────────────│─────────────▶│              │
     │              │              │              │              │
     │              │              │ Resolve      │              │
     │              │              │ provider +   │              │
     │              │              │ model        │              │
     │              │              │              │              │
     │              │              │ Check        │              │
     │              │              │ circuit      │              │
     │              │              │ breaker      │              │
     │              │              │              │              │
     │              │              │ Send request │─────────────▶│
     │              │              │─────────────▶│              │
     │              │              │              │  Translate   │
     │              │              │              │  + forward   │
     │              │              │              │              │
     │              │◀─── chunks ──│◀─── chunks ──│◀── response ─│
     │              │              │              │              │
     │ ◀── NDJSON ──│              │              │              │
     │   stream     │              │              │              │
     │              │              │              │              │
     │              │              │ Record       │              │
     │              │              │ usage + cost │              │
     │              │              │              │              │
```

### 7.2 Circuit Breaker State Machine

```
                ┌─────────────────┐
                │     CLOSED      │
                │ (normal ops)    │
                └────────┬────────┘
                         │
               N consecutive failures
               (default: 5)
                         │
                         ▼
                ┌─────────────────┐
          ┌────▶│      OPEN       │
          │     │ (fast-fail)     │
          │     └────────┬────────┘
          │              │
          │     Cooldown expires
          │     (30s default)
          │              │
          │              ▼
          │     ┌─────────────────┐
          │     │   HALF-OPEN     │
          │     │ (test request)  │
          │     └────────┬────────┘
          │              │
          │    ┌─────────┴──────────┐
          │    │                    │
          │  success              failure
          │    │                    │
          │    ▼                    │
          │ ┌────────┐             │
          │ │ CLOSED │             │
          │ └────────┘             │
          └────────────────────────┘
```

---

## 8. Configuration Loading

```
Startup
  │
  ▼
Load providers.yaml
  │
  ▼
Env var interpolation
  │  Walk all string fields
  │  Replace ${VAR} with os.Getenv("VAR")
  │  Missing → log warning, set empty
  │
  ▼
Validation
  │  - Required fields present
  │  - Valid provider types
  │  - No duplicates
  │  - Virtual model chains valid
  │
  ▼
Initialize providers
  │  - Create adapter instances
  │  - Initialize circuit breakers
  │  - Start health polling goroutines
  │  - Start model discovery (if enabled)
  │
  ▼
Start HTTP server
  │  - Bind to configured port
  │  - Register routes
  │  - Register middleware
  │
  ▼
Ready
```

---

## 9. Integration with ollama-stack

### 9.1 Network Topology

```
┌──────────┐     ┌──────────────┐     ┌──────────┐
│          │     │              │     │          │
│  Client  │────▶│  OllaRouter  │────▶│  Ollama  │
│ (Open    │     │  :11435      │     │  :11434  │
│  WebUI)  │     │              │     │          │
│          │     │              │     └──────────┘
│          │     │              │
│          │     │              │────▶│  OpenAI   │
│          │     │              │     │  (cloud)  │
│          │     │              │
│          │     │              │────▶│ Anthropic │
│          │     │              │     │  (cloud)  │
│          │     │              │
│          │     │              │────▶│ ChatGPT+  │
│          │     │              │     │  (web)    │
└──────────┘     └──────────────┘     └──────────┘
```

### 9.2 Docker Compose Integration

```yaml
services:
  ollarouter:
    build: .
    image: ollarouter:latest
    ports:
      - "11435:11435"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
    volumes:
      - ./providers.yaml:/app/config/providers.yaml:ro
    depends_on:
      - ollama
    healthcheck:
      test: ["CMD", "/app/ollarouter", "healthcheck"]
```

---

## Appendix A: Complete Config Schema

```yaml
# ============================================================
# OllaRouter Configuration — providers.yaml
# ============================================================

settings:
  port: 11435
  admin_port: 11436
  log_level: info
  log_format: json
  prometheus_enabled: true
  rate_limiting_enabled: true
  default_max_retries: 3
  default_timeout_seconds: 30
  usage_persistence: memory
  usage_file_path: /data/usage.json

api_keys:
  - name: admin-key
    key: "${OLLAROUTER_API_KEY}"
    rate_limit: 100
    monthly_budget_usd: 100.00

providers:
  - name: openai-main
    type: openai
    base_url: https://api.openai.com/v1
    api_key: "${OPENAI_API_KEY}"
    organization_id: ""
    models:
      - name: gpt-4o
        max_tokens: 16384
    pricing:
      input_per_1k: 0.0025
      output_per_1k: 0.01

  - name: local-ollama
    type: ollama
    base_url: http://ollama:11434
    auto_discover: true
    polling_interval_seconds: 60

virtual_models:
  - name: fast-chat
    fallback_chain:
      - openai-main/gpt-4o-mini
      - groq/llama3-70b-8192
    timeout: 30
```

---

## Appendix B: File Layout

```
ollarouter/
├── cmd/ollarouter/main.go
├── internal/
│   ├── config/        # Config parsing, validation
│   ├── router/        # HTTP handlers, middleware
│   ├── provider/      # Provider interface + adapters
│   │   ├── openai/
│   │   ├── anthropic/
│   │   ├── openaicompat/
│   │   ├── ollama/
│   │   └── chatgpt/
│   ├── streaming/     # SSE→NDJSON translation
│   ├── engine/        # Routing, fallback, circuit breaker
│   ├── auth/          # API key auth, rate limiter
│   ├── tracking/      # Token counting, cost, persistence
│   ├── admin/         # Admin API handlers
│   ├── webui/         # Go embed for admin SPA
│   └── logger/        # Structured JSON logger
├── webui/             # Admin SPA (React + Vite)
├── Dockerfile
├── go.mod / go.sum
└── Makefile
```

---

*End of System Architecture Document*
