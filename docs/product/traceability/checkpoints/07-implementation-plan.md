# Phase 7 — Implementation Plan: OllaRouter

**Date:** 2026-05-20
**Agent:** architect
**Project:** ollama-stack (C:\Projects\ollama-stack)
**Phase 6 Reference:** `docs/product/traceability/checkpoints/06-architecture-review.md`
**Architecture Docs:** `docs/product/architecture/01-system-architecture.md`, `02-provider-adapter-contract.md`, `03-docker-architecture.md`
**Backlog Reference:** `docs/product/backlog/ollarouter-backlog.md`

---

## 1. Current State

The project is a **fresh scaffold** — no Go code, no `go.mod`, no `ollarouter/` directory exists yet. The existing `docker-compose.yml` has `ollama` and `open-webui` services only.

### What Exists Today

| Artifact | Location | Status |
|----------|----------|--------|
| docker-compose.yml | `C:\Projects\ollama-stack\` | Has ollama + open-webui (no ollarouter) |
| docker-compose.yml.old | `C:\Projects\ollama-stack\` | Backup of older compose |
| Architecture docs | `docs/product/architecture/` | 3 documents (approved) |
| Backlog | `docs/product/backlog/ollarouter-backlog.md` | 7 EPICs, 24 Features, 56 Stories |
| Market research | `docs/product/traceability/checkpoints/` | Phases 1-6 completed |

### What Must Be Created

All Go source files, configuration, Dockerfile, Makefile, web UI source, and updated docker-compose.yml.

---

## 2. Project Directory Structure (Target)

```
C:\Projects\ollama-stack\
├── ollarouter/                          # Go module root
│   ├── cmd/
│   │   └── ollarouter/
│   │       └── main.go                  # Entry point (package main)
│   ├── internal/
│   │   ├── config/
│   │   │   ├── config.go                # Config struct definitions
│   │   │   ├── parser.go                # YAML parsing + env interpolation
│   │   │   └── parser_test.go
│   │   ├── logger/
│   │   │   └── logger.go                # Structured JSON logger (slog wrapper)
│   │   ├── router/
│   │   │   ├── router.go                # Chi router setup, middleware chain
│   │   │   ├── middleware.go            # Logging, auth, rate limit middleware
│   │   │   ├── handler_chat.go          # POST /api/chat
│   │   │   ├── handler_generate.go      # POST /api/generate
│   │   │   ├── handler_tags.go          # GET /api/tags
│   │   │   ├── handler_version.go       # GET /api/version
│   │   │   ├── handler_show.go          # POST /api/show
│   │   │   ├── handler_health.go        # GET /health, /health/providers
│   │   │   └── handler_metrics.go       # GET /metrics
│   │   ├── provider/
│   │   │   ├── provider.go              # Provider interface + types
│   │   │   ├── factory.go               # Adapter factory (NewProvider)
│   │   │   ├── registry.go              # Provider registry
│   │   │   ├── openai/
│   │   │   │   ├── adapter.go           # OpenAI adapter
│   │   │   │   └── adapter_test.go
│   │   │   ├── anthropic/
│   │   │   │   ├── adapter.go           # Anthropic adapter
│   │   │   │   └── adapter_test.go
│   │   │   ├── openaicompat/
│   │   │   │   ├── adapter.go           # OpenAI-compatible adapter
│   │   │   │   └── adapter_test.go
│   │   │   ├── ollama/
│   │   │   │   ├── adapter.go           # Ollama passthrough adapter
│   │   │   │   └── adapter_test.go
│   │   │   └── chatgpt/
│   │   │       ├── adapter.go           # ChatGPT Plus adapter
│   │   │       └── adapter_test.go
│   │   ├── streaming/
│   │   │   ├── translator.go            # SSE → NDJSON translation interfaces
│   │   │   ├── openai_translator.go     # OpenAI SSE parser
│   │   │   ├── anthropic_translator.go  # Anthropic SSE parser
│   │   │   ├── chatgpt_translator.go    # ChatGPT SSE parser
│   │   │   ├── ollama_writer.go          # NDJSON flush writer
│   │   │   └── translator_test.go
│   │   ├── engine/
│   │   │   ├── router.go                # Model → provider resolution
│   │   │   ├── fallback.go              # Fallback chain executor
│   │   │   ├── circuit_breaker.go        # Circuit breaker per provider
│   │   │   └── retry.go                 # Retry with exponential backoff
│   │   ├── auth/
│   │   │   ├── api_key.go               # API key validation + hashing
│   │   │   └── rate_limit.go            # Token bucket rate limiter
│   │   ├── tracking/
│   │   │   ├── counter.go               # Token + cost counting
│   │   │   ├── store.go                 # In-memory usage store
│   │   │   └── persistence.go           # Optional file persistence
│   │   ├── admin/
│   │   │   ├── handler_providers.go     # CRUD for providers
│   │   │   ├── handler_keys.go          # CRUD for API keys
│   │   │   ├── handler_stats.go         # Stats/cost dashboard API
│   │   │   ├── handler_logs.go          # Request log API
│   │   │   └── handler_settings.go      # Settings API
│   │   └── webui/
│   │       └── embed.go                 //go:embed dist/* — admin SPA
│   ├── go.mod
│   ├── go.sum
│   ├── Dockerfile
│   ├── Makefile
│   ├── providers.yaml.example           # Example config
│   └── .dockerignore
├── webui/                               # Admin SPA source (separate directory)
│   ├── src/
│   │   ├── App.tsx
│   │   ├── main.tsx
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx
│   │   │   ├── Providers.tsx
│   │   │   ├── ProviderWizard.tsx
│   │   │   ├── ApiKeys.tsx
│   │   │   ├── Settings.tsx
│   │   │   └── ModelCatalog.tsx
│   │   ├── components/
│   │   │   ├── Layout.tsx
│   │   │   ├── StatusDot.tsx
│   │   │   └── ...
│   │   └── api/
│   │       └── client.ts
│   ├── package.json
│   ├── vite.config.ts
│   ├── tsconfig.json
│   ├── tailwind.config.js
│   └── dist/                            # Pre-built (committed for go:embed)
├── .env.example
├── docker-compose.yml                    # Updated with ollarouter service
└── README.md
```

---

## 3. Build Order — Sprint by Sprint

### 3.1 Dependencies Between Major Modules

```
config ──▶ logger ──▶ provider ──▶ streaming ──▶ engine ──▶ router ──▶ main
                          │            │            │
                          │            ▼            │
                          └────▶ auth              │
                                    │              │
                                    ▼              │
                              tracking ◀───────────┘
                                    │
                                    ▼
                                admin ──▶ webui embed

Dockerfile + docker-compose: anytime after main.go works
```

### 3.2 SPRINT 1: Project Scaffold + Core Config (EPIC-1 Foundation)

**Dependency root — no upstream dependencies.**

| Step | File(s) | Description | Verifies |
|------|---------|-------------|----------|
| 1.1 | `ollarouter/go.mod` | Initialize Go module: `go mod init github.com/yourorg/ollarouter`; add dependencies: `chi`, `yaml.v3`, `slog` (stdlib), `prometheus/client_golang` | `go build ./...` succeeds |
| 1.2 | `ollarouter/internal/config/config.go` | All config struct types: `Config`, `Settings`, `ProviderConfig`, `APIKeyConfig`, `ModelConfig`, `PricingConfig`, `VirtualModelConfig` | `go vet ./internal/config/...` |
| 1.3 | `ollarouter/internal/config/parser.go` | YAML file reader, env var interpolation (`${VAR}` → `os.Getenv`), validation | `go test ./internal/config/...` |
| 1.4 | `ollarouter/internal/config/parser_test.go` | Tests: valid YAML, env interpolation, missing vars, invalid types, duplicates | `go test -v ./internal/config/...` |
| 1.5 | `ollarouter/internal/logger/logger.go` | Structured JSON logger wrapping `slog` with: level, format (json/text), request_id field helper, key masking | Manual: inspect log output format |
| 1.6 | `Dockerfile` | Multi-stage build: `golang:1.23-alpine` → `distroless/static-debian12:nonroot` | `docker build -t ollarouter:test .` |
| 1.7 | `ollarouter/.dockerignore` | Exclude `.env`, `providers.yaml`, `.git/`, `node_modules/` | Verify with `docker build` |
| 1.8 | `ollarouter/Makefile` | Targets: `build`, `test`, `lint`, `docker-build`, `clean`, `webui-build` | `make build` |

**Verification for Sprint 1:**
```bash
cd ollarouter && go mod tidy && go build ./... && go test ./...
docker build -t ollarouter:test .
```

---

### 3.3 SPRINT 2: Provider Interface + Adapters (EPIC-1, FEAT-1.4)

**Depends on:** Sprint 1 (config types, logger)

| Step | File(s) | Description | Dependencies |
|------|---------|-------------|--------------|
| 2.1 | `internal/provider/provider.go` | `Provider` interface: `Chat()`, `ChatStream()`, `Generate()`, `GenerateStream()`, `ListModels()`, `CheckHealth()`. All request/response types: `ChatRequest`, `ChatResponse`, `GenerateRequest`, `GenerateResponse`, `StreamChunk`, `Message`, `RequestOptions`, `ModelInfo`, `ProviderHealth`, `TokenUsage`, `Metrics`, sentinel errors | config types |
| 2.2 | `internal/provider/factory.go` | `NewProvider(config ProviderConfig) (Provider, error)` — type switch factory | provider.go |
| 2.3 | `internal/provider/registry.go` | `Registry` struct: `map[string]*ProviderState`, thread-safe with `sync.RWMutex`. Methods: `Get`, `Add`, `Remove`, `Update`, `List`, `AllProviders` | provider.go, config |
| 2.4 | `internal/provider/openai/adapter.go` | OpenAI adapter: translate Ollama → OpenAI `/v1/chat/completions`, SSE → NDJSON streaming, health check via `GET /models` | provider.go, streaming (see Sprint 3) |
| 2.5 | `internal/provider/anthropic/adapter.go` | Anthropic adapter: Messages API, `content_block_delta` SSE → NDJSON | provider.go |
| 2.6 | `internal/provider/openaicompat/adapter.go` | OpenAI-compatible adapter: generic, custom base URL, Azure deployment support | provider.go |
| 2.7 | `internal/provider/ollama/adapter.go` | Ollama passthrough adapter: forward as-is (already NDJSON), auto-discovery polling, `GET /api/tags` | provider.go |
| 2.8 | `internal/provider/openai/adapter_test.go` | Unit tests: request translation, SSE parsing, error handling | — |
| 2.9 | `internal/provider/anthropic/adapter_test.go` | Unit tests: request translation, content block handling | — |
| 2.10 | `internal/provider/ollama/adapter_test.go` | Unit tests: passthrough, auto-discovery, health check | — |

**Verification for Sprint 2:**
```bash
cd ollarouter && go test ./internal/provider/... -v -count=1
```

---

### 3.4 SPRINT 3: Streaming Engine (EPIC-1, FEAT-1.1, FEAT-1.2)

**Depends on:** Sprint 2 (provider types)

| Step | File(s) | Description |
|------|---------|-------------|
| 3.1 | `internal/streaming/translator.go` | `StreamTranslator` interface, `SSEParser` type, NDJSON formatting utilities |
| 3.2 | `internal/streaming/openai_translator.go` | Parse OpenAI `data: {...}` SSE events → `StreamChunk`; handle `[DONE]` |
| 3.3 | `internal/streaming/anthropic_translator.go` | Parse Anthropic `content_block_start`, `content_block_delta`, `message_delta` → `StreamChunk` |
| 3.4 | `internal/streaming/chatgpt_translator.go` | Parse ChatGPT `response.text.delta`, `message.complete` → `StreamChunk` |
| 3.5 | `internal/streaming/ollama_writer.go` | `OllamaNDJSONWriter`: flushes NDJSON lines with `\n`, sets `Content-Type: application/x-ndjson`, handles `http.Flusher` |
| 3.6 | `internal/streaming/translator_test.go` | Unit tests: OpenAI SSE → NDJSON, Anthropic SSE → NDJSON, edge cases (empty chunks, partial events) |

**Verification:**
```bash
cd ollarouter && go test ./internal/streaming/... -v -count=1
```

---

### 3.5 SPRINT 4: Routing Engine — Retry, Fallback, Circuit Breaker (EPIC-1, FEAT-1.5)

**Depends on:** Sprint 2 (provider registry), Sprint 3 (streaming)

| Step | File(s) | Description |
|------|---------|-------------|
| 4.1 | `internal/engine/router.go` | Model → provider resolution: `ResolveModel(modelName string) (*ProviderState, string, error)`. Handles `provider/model` prefix. Resolves virtual models to first provider in chain. |
| 4.2 | `internal/engine/retry.go` | `RetryEngine`: exponential backoff (1s, 2s, 4s...), max retries config, `Retry-After` header support, jitter |
| 4.3 | `internal/engine/circuit_breaker.go` | `CircuitBreaker`: `closed → open → half-open` state machine, configurable threshold (5), cooldown (30s) |
| 4.4 | `internal/engine/fallback.go` | `FallbackExecutor`: iterate fallback chain, reset retry per provider, set `x-ollarouter-provider` header, detect circular chains |

**Verification:**
```bash
cd ollarouter && go test ./internal/engine/... -v -count=1
```

---

### 3.6 SPRINT 5: HTTP Router — Public API Handlers (EPIC-1, FEAT-1.1, 1.2, 1.3, 1.7)

**Depends on:** Sprint 4 (engine), Sprint 3 (streaming), Sprint 2 (adapters), Sprint 1 (config, logger)

| Step | File(s) | Description |
|------|---------|-------------|
| 5.1 | `internal/router/router.go` | Chi router setup: route groups, middleware chain, CORS for admin. `NewRouter(app *App) http.Handler` |
| 5.2 | `internal/router/middleware.go` | Request ID (ULID), request logging, recovery/panic handler, API key auth placeholder, rate limiter placeholder |
| 5.3 | `internal/router/handler_chat.go` | `POST /api/chat`: parse body, validate model exists, call provider `Chat()` or `ChatStream()`, write NDJSON via `OllamaNDJSONWriter`, handle errors |
| 5.4 | `internal/router/handler_generate.go` | `POST /api/generate`: parse body (prompt/model/options/images), translate to chat format if needed, route to provider, write response |
| 5.5 | `internal/router/handler_tags.go` | `GET /api/tags`: aggregate models from all providers (config + auto-discovered), return `{"models":[...]}` |
| 5.6 | `internal/router/handler_version.go` | `GET /api/version`: return `{"version":"0.1.0","api_version":"1.0.0"}` |
| 5.7 | `internal/router/handler_show.go` | `POST /api/show`: return model detail (size, details, provider info) for known models |
| 5.8 | `internal/router/handler_health.go` | `GET /health`, `GET /health/providers`: service status, per-provider health |
| 5.9 | `internal/router/handler_metrics.go` | `GET /metrics`: Prometheus text format (if enabled) |
| 5.10 | `cmd/ollarouter/main.go` | Main entry: load config → init logger → init registry → init engine → init router → start HTTP server. Handle `SIGTERM`/`SIGINT` graceful shutdown. `healthcheck` subcommand. |

**Verification:**
```bash
cd ollarouter && go build -o build/ollarouter ./cmd/ollarouter
# Start with sample config:
OLLAROUTER_CONFIG_PATH=./providers.yaml.example ./build/ollarouter
# Test endpoints:
curl http://localhost:11435/api/version
curl http://localhost:11435/health
curl http://localhost:11435/api/tags
```

---

### 3.7 SPRINT 6: Auth + Rate Limiting (EPIC-3)

**Depends on:** Sprint 5 (router middleware points exist)

| Step | File(s) | Description |
|------|---------|-------------|
| 6.1 | `internal/auth/api_key.go` | `Authenticator`: parse `Authorization: Bearer` header, validate against configured keys, SHA-256 hash comparison, key masking in logs. `KeyStore` for in-memory key management. |
| 6.2 | `internal/auth/rate_limit.go` | `TokenBucket`: per-key token bucket algorithm, configurable refill rate and capacity. `RateLimiter` middleware: extract key from auth context, check bucket, set `Retry-After` header on 429. Budget cap enforcement (monthly USD). |
| 6.3 | `internal/router/middleware.go` (update) | Wire auth + rate limiter middleware into chi chain. Auth skipped if no keys configured. Rate limiter reads key context from auth. |

**Verification:**
```bash
# With auth enabled:
curl -H "Authorization: Bearer valid-key" http://localhost:11435/api/tags  # 200
curl http://localhost:11435/api/tags  # 401
# Rate limit:
for i in $(seq 1 101); do curl -s -o /dev/null -w "%{http_code}\n" -H "Authorization: Bearer test-key" http://localhost:11435/api/tags; done
```

---

### 3.8 SPRINT 7: Tracking + Admin API (EPIC-4, EPIC-2 foundation)

**Depends on:** Sprint 5 (router has request context), Sprint 6 (auth context)

| Step | File(s) | Description |
|------|---------|-------------|
| 7.1 | `internal/tracking/counter.go` | `Tracker`: increment token counts (input/output) and cost per provider/model/key. Thread-safe with `sync.Mutex`. |
| 7.2 | `internal/tracking/store.go` | `UsageStore`: in-memory aggregation with `map[string]*UsageRecord`. Methods: `Record(req)`, `GetSummary()`, `GetByProvider()`, `GetByModel()`, `GetTimeSeries()`. |
| 7.3 | `internal/tracking/persistence.go` | `FilePersistence`: write `UsageStore` to JSON file every 60s, load on startup. Configurable via `usage_persistence` and `usage_file_path`. |
| 7.4 | `internal/admin/handler_providers.go` | `GET/POST /admin/api/providers`, `PUT/DELETE /admin/api/providers/:name`, `POST .../test`, `GET .../health` |
| 7.5 | `internal/admin/handler_keys.go` | `GET/POST /admin/api/api-keys`, `DELETE /admin/api/api-keys/:id` |
| 7.6 | `internal/admin/handler_stats.go` | `GET /admin/api/stats/summary`, `GET .../cost-by-provider`, `GET .../cost-by-model`, `GET .../cost-trend` |
| 7.7 | `internal/admin/handler_logs.go` | `GET /admin/api/logs`: paginated request log with filters (provider, model, status, date range) |
| 7.8 | `internal/admin/handler_settings.go` | `GET/PUT /admin/api/settings`, `GET /admin/api/config/yaml` |

**Verification:**
```bash
curl http://localhost:11435/admin/api/providers  # requires auth
curl http://localhost:11435/admin/api/stats/summary
```

---

### 3.9 SPRINT 8: Embedded Admin Web UI (EPIC-2)

**Depends on:** Sprint 7 (admin API exists)

| Step | File(s) | Description |
|------|---------|-------------|
| 8.1 | `webui/package.json`, `webui/vite.config.ts`, `webui/tsconfig.json`, `webui/tailwind.config.js` | React + Vite + Tailwind + shadcn/ui scaffold |
| 8.2 | `webui/src/main.tsx`, `webui/src/App.tsx` | React entry point, router setup (react-router) |
| 8.3 | `webui/src/api/client.ts` | Admin API client: fetch wrapper, auth header injection |
| 8.4 | `webui/src/components/Layout.tsx` | Shell layout: sidebar nav, header, content area |
| 8.5 | `webui/src/pages/Dashboard.tsx` | Dashboard summary cards, recent requests |
| 8.6 | `webui/src/pages/Providers.tsx` | Provider list table with health dots, edit/delete actions, test button |
| 8.7 | `webui/src/pages/ProviderWizard.tsx` | 4-step wizard: type selection (Step 1), connection details (Step 2), model config (Step 3), defaults/save (Step 4) |
| 8.8 | `webui/src/pages/ApiKeys.tsx` | API key management: table, create dialog, revoke, copy on creation |
| 8.9 | `webui/src/pages/Settings.tsx` | Global settings form, YAML config preview |
| 8.10 | `internal/webui/embed.go` | `//go:embed dist/*` — embed built admin SPA into Go binary |
| 8.11 | `internal/router/handler_admin.go` (new) | Serve embedded SPA for `/admin/*`, handle SPA routing (fallback to index.html) |

**Verification:**

Build web UI:
```bash
cd webui && npm ci && npm run build
```

Build Go binary (embeds dist/):
```bash
cd ollarouter && go build -o build/ollarouter ./cmd/ollarouter
```

Open browser: http://localhost:11435/admin/

---

### 3.10 SPRINT 9: ChatGPT Plus Provider (EPIC-5)

**Depends on:** Sprint 2 (new adapter type)

| Step | File(s) | Description |
|------|---------|-------------|
| 9.1 | `internal/provider/chatgpt/adapter.go` | ChatGPT Plus adapter: session cookie auth, backend API proxy, model auto-discovery via `GET /backend-api/models` |
| 9.2 | `internal/streaming/chatgpt_translator.go` (may already exist from Sprint 3.4) | ChatGPT SSE → NDJSON: `response.text.delta`, `message.complete` events |
| 9.3 | Cookie health monitoring + auto-refresh (built into adapter) | Detect 403, attempt cookie refresh, degrade health on near-expiry, request-level retry with new cookie |
| 9.4 | `internal/provider/chatgpt/adapter_test.go` | Unit tests with mock ChatGPT server |

**Verification:**
```bash
CHATGPT_SESSION_COOKIE="__Secure-next-auth.session-token=xxx" \
  ./build/ollarouter
curl http://localhost:11435/api/tags  # Should show ChatGPT models
```

---

### 3.11 SPRINT 10: Stats Dashboard UI (EPIC-4) + Model Catalog (EPIC-7)

**Depends on:** Sprint 8 (admin UI infrastructure), Sprint 7 (stats API)

| Step | File(s) | Description |
|------|---------|-------------|
| 10.1 | `webui/src/pages/StatsDashboard.tsx` | Summary cards, cost-by-provider chart (Chart.js or Recharts), cost trend line chart, request log |
| 10.2 | `webui/src/pages/ModelCatalog.tsx` | Searchable model table, provider badges (colored), filter chips, empty state |
| 10.3 | `webui/src/components/ModelDetailPanel.tsx` | Slide-over panel: model details, pricing, fallback chain, quick test, copy name |
| 10.4 | `webui/src/components/RequestLogTable.tsx` | Infinite-scroll table, column sorting, filter controls, expandable rows, CSV export |

**Verification:**
```bash
cd webui && npm run build && cd ../ollarouter && go build -o build/ollarouter ./cmd/ollarouter
# Open browser to /admin/models and /admin/stats
```

---

### 3.12 SPRINT 11: Docker Compose Integration + Example Config

**Depends on:** Sprint 5 (binary works)

| Step | File(s) | Description |
|------|---------|-------------|
| 11.1 | `C:\Projects\ollama-stack\docker-compose.yml` (UPDATE) | Add `ollarouter` service (build, ports, env, volumes, depends_on, healthcheck). Update `open-webui` to use `OLLAMA_BASE_URL=http://ollarouter:11435`. Add `ollarouter_data` volume. |
| 11.2 | `ollarouter/providers.yaml.example` | Complete example config with comments: OpenAI, Anthropic, Ollama, OpenAI-compatible providers, virtual models |
| 11.3 | `.env.example` | Template for all env vars (keys masked) |

---

## 4. Package Import Structure

### 4.1 Go Module (`go.mod`)

```
module github.com/yourorg/ollarouter

go 1.23

require (
    github.com/go-chi/chi/v5 v5.1.0
    github.com/go-chi/cors v1.2.1
    github.com/prometheus/client_golang v1.20.0
    github.com/prometheus/client_model v0.6.1
    github.com/prometheus/common v0.55.0
    gopkg.in/yaml.v3 v3.0.1
)
```

### 4.2 Import Graph

```
cmd/ollarouter/main.go
  ├── internal/config      (Config, ParseFile)
  ├── internal/logger      (New, Info, Error, etc.)
  ├── internal/provider    (NewProvider, Registry)
  ├── internal/engine      (ResolveModel, FallbackExecutor)
  ├── internal/router      (NewRouter)
  ├── internal/auth        (Authenticator, RateLimiter)
  ├── internal/tracking    (Tracker, UsageStore)
  └── internal/webui       (AdminUI)

internal/router/router.go
  ├── internal/middleware  (chi Middlewares)
  ├── internal/auth        (AuthMiddleware)
  ├── internal/engine      (ModelResolver)
  ├── internal/streaming   (OllamaNDJSONWriter)
  └── internal/tracking    (UsageMiddleware)

internal/provider/openai/adapter.go
  ├── internal/provider    (Provider interface, types)
  └── internal/streaming   (OpenAITranslator)
```

---

## 5. Dockerfile

**Location:** `C:\Projects\ollama-stack\ollarouter\Dockerfile`

```dockerfile
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
    -o /build/ollarouter ./cmd/ollarouter

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /build/ollarouter /app/ollarouter
EXPOSE 11435
USER nonroot:nonroot
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/app/ollarouter", "healthcheck"]
ENTRYPOINT ["/app/ollarouter"]
```

---

## 6. docker-compose.yml Changes

**Update** `C:\Projects\ollama-stack\docker-compose.yml`:

### Add ollarouter service:
```yaml
  ollarouter:
    build:
      context: ./ollarouter
      dockerfile: Dockerfile
    image: ollarouter:latest
    container_name: ollarouter
    ports:
      - "11435:11435"
    environment:
      - OLLAROUTER_PORT=11435
      - OLLAROUTER_CONFIG_PATH=/app/config/providers.yaml
      - OLLAROUTER_LOG_LEVEL=info
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - GROQ_API_KEY=${GROQ_API_KEY:-}
      - CHATGPT_SESSION_COOKIE=${CHATGPT_SESSION_COOKIE:-}
      - OLLAROUTER_API_KEY=${OLLAROUTER_API_KEY:-}
    volumes:
      - ./ollarouter/providers.yaml:/app/config/providers.yaml:ro
      - ollarouter_data:/data
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
```

### Update open-webui:
```yaml
    environment:
      - OLLAMA_BASE_URL=http://ollarouter:11435   # WAS: http://ollama:11434
```

### Add volumes:
```yaml
  ollarouter_data:
    external: false
```

---

## 7. Verification Approach per Build Step

### 7.1 Per-Sprint Verification

| Sprint | Verification Command | Expected |
|--------|---------------------|----------|
| 1 (Scaffold) | `go build ./... && go test ./internal/config/...` | Build + unit tests pass |
| 2 (Adapters) | `go test ./internal/provider/... -v -count=1` | All adapter tests pass |
| 3 (Streaming) | `go test ./internal/streaming/... -v -count=1` | SSE → NDJSON translation tests pass |
| 4 (Engine) | `go test ./internal/engine/... -v -count=1` | Retry/fallback/circuit breaker tests pass |
| 5 (Router) | `go build -o build/ollarouter ./cmd/ollarouter && make test` | Binary builds; manual curl tests pass |
| 6 (Auth) | `go test ./internal/auth/... -v -count=1` | Auth + rate limit tests pass |
| 7 (Admin API) | `curl http://localhost:11435/admin/api/providers -H "Authorization: Bearer test-key"` | 200 with provider list |
| 8 (Web UI) | `npm run build && go build` | Static assets embedded; /admin loads |
| 9 (ChatGPT) | `go test ./internal/provider/chatgpt/... -v` | Mock ChatGPT API tests pass |
| 10 (Stats UI) | `npm run build && go build && open http://localhost:11435/admin` | Dashboard renders with data |

### 7.2 Full Integration Test (after Sprint 5)

```bash
# 1. Start with example config
OLLAROUTER_CONFIG_PATH=./providers.yaml.example OLLAROUTER_PORT=11435 \
  ./build/ollarouter &

# 2. Test health
curl -s http://localhost:11435/health | jq .

# 3. Test version
curl -s http://localhost:11435/api/version | jq .

# 4. Test tags
curl -s http://localhost:11435/api/tags | jq '.models | length'

# 5. Test chat (non-streaming)
curl -s -X POST http://localhost:11435/api/chat \
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"Hello"}],"stream":false}' | jq .

# 6. Test chat (streaming) — expect NDJSON
curl -s -N -X POST http://localhost:11435/api/chat \
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"Count to 5"}],"stream":true}'

# 7. Test unknown model — expect 404
curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:11435/api/chat \
  -d '{"model":"unknown-model","messages":[{"role":"user","content":"Hi"}],"stream":false}'

# 8. Test invalid JSON — expect 400
curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:11435/api/chat \
  -d 'invalid json'

# 9. Stop
kill %1
```

### 7.3 Docker Integration Test (after Sprint 11)

```bash
# Build
cd ollarouter && docker build -t ollarouter:latest .

# Start stack
cd .. && docker compose up -d

# Test via open-webui
curl http://localhost:3000  # Should load Open WebUI

# Health endpoint
curl http://localhost:11435/health

# View logs
docker compose logs ollarouter

# Cleanup
docker compose down
```

---

## 8. Dependency Graph (Visual)

```
Sprint 1:  go.mod + config + logger + Dockerfile
              │
              ▼
Sprint 2:  Provider interface + adapters (openai, anthropic, ollama, openaicompat)
              │
              ▼
Sprint 3:  Streaming engine (SSE → NDJSON translators)
              │
              ▼
Sprint 4:  Engine (retry, fallback, circuit breaker)
              │
              ├──────────────────────────────────┐
              ▼                                  │
Sprint 5:  HTTP Router + Handlers + main.go      │
              │                                  │
              ├──────────┬───────────┐            │
              ▼          ▼           ▼            │
Sprint 6:  Auth    Sprint 7:    Sprint 8:         │
           +        Tracking +   Web UI           │
           Rate      Admin API   (React)          │
           Limit       │           │              │
                       │           │              │
                       ▼           ▼              │
              Sprint 10: Stats Dashboard UI       │
                       + Model Catalog            │
                                                 │
              Sprint 9: ChatGPT Plus Provider ────┘
                                                 │
              Sprint 11: Docker Compose Integration
```

---

## 9. Risk Mitigation in Build Order

| Risk | How Build Order Mitigates |
|------|--------------------------|
| **Streaming is most complex** | Sprint 3 dedicated to streaming translation before routing code. Testable in isolation. |
| **Config parser errors cascade** | Sprint 1 builds + tests config parser first. All later code depends on known-good config types. |
| **Provider adapter differences** | Sprint 2 defines interface and tests 4 adapter types before routing code. Interface changes caught early. |
| **Docker build fails** | Sprint 1 includes Dockerfile verification before any Go business logic. |
| **Auth is optional for MVP** | Sprint 6 is deferred until after core routing works (Sprint 5). MVP can run without auth. |
| **Web UI delays API delivery** | UI is Sprint 8, after all API endpoints exist. API can be tested with curl independently. |

---

## 10. File Creation/Modification Summary

### Files to CREATE (total: ~55 files)

```
ollarouter/go.mod                                   # NEW
ollarouter/go.sum                                   # NEW (generated)
ollarouter/Dockerfile                               # NEW
ollarouter/Makefile                                 # NEW
ollarouter/.dockerignore                            # NEW
ollarouter/providers.yaml.example                   # NEW
ollarouter/cmd/ollarouter/main.go                   # NEW
ollarouter/internal/config/config.go                # NEW
ollarouter/internal/config/parser.go                # NEW
ollarouter/internal/config/parser_test.go           # NEW
ollarouter/internal/logger/logger.go                # NEW
ollarouter/internal/provider/provider.go            # NEW
ollarouter/internal/provider/factory.go             # NEW
ollarouter/internal/provider/registry.go            # NEW
ollarouter/internal/provider/openai/adapter.go      # NEW
ollarouter/internal/provider/openai/adapter_test.go # NEW
ollarouter/internal/provider/anthropic/adapter.go   # NEW
ollarouter/internal/provider/anthropic/adapter_test.go # NEW
ollarouter/internal/provider/openaicompat/adapter.go # NEW
ollarouter/internal/provider/openaicompat/adapter_test.go # NEW
ollarouter/internal/provider/ollama/adapter.go      # NEW
ollarouter/internal/provider/ollama/adapter_test.go # NEW
ollarouter/internal/provider/chatgpt/adapter.go     # NEW
ollarouter/internal/provider/chatgpt/adapter_test.go # NEW
ollarouter/internal/streaming/translator.go         # NEW
ollarouter/internal/streaming/openai_translator.go  # NEW
ollarouter/internal/streaming/anthropic_translator.go # NEW
ollarouter/internal/streaming/chatgpt_translator.go # NEW
ollarouter/internal/streaming/ollama_writer.go      # NEW
ollarouter/internal/streaming/translator_test.go    # NEW
ollarouter/internal/engine/router.go                # NEW
ollarouter/internal/engine/retry.go                 # NEW
ollarouter/internal/engine/circuit_breaker.go       # NEW
ollarouter/internal/engine/fallback.go              # NEW
ollarouter/internal/auth/api_key.go                 # NEW
ollarouter/internal/auth/rate_limit.go              # NEW
ollarouter/internal/tracking/counter.go             # NEW
ollarouter/internal/tracking/store.go               # NEW
ollarouter/internal/tracking/persistence.go         # NEW
ollarouter/internal/admin/handler_providers.go      # NEW
ollarouter/internal/admin/handler_keys.go           # NEW
ollarouter/internal/admin/handler_stats.go          # NEW
ollarouter/internal/admin/handler_logs.go           # NEW
ollarouter/internal/admin/handler_settings.go       # NEW
ollarouter/internal/webui/embed.go                  # NEW
ollarouter/internal/router/router.go                # NEW
ollarouter/internal/router/middleware.go            # NEW
ollarouter/internal/router/handler_chat.go          # NEW
ollarouter/internal/router/handler_generate.go      # NEW
ollarouter/internal/router/handler_tags.go          # NEW
ollarouter/internal/router/handler_version.go       # NEW
ollarouter/internal/router/handler_show.go          # NEW
ollarouter/internal/router/handler_health.go        # NEW
ollarouter/internal/router/handler_metrics.go       # NEW
webui/package.json                                  # NEW
webui/vite.config.ts                                # NEW
webui/tsconfig.json                                 # NEW
webui/tailwind.config.js                            # NEW
webui/src/main.tsx                                  # NEW
webui/src/App.tsx                                   # NEW
webui/src/api/client.ts                             # NEW
webui/src/components/Layout.tsx                     # NEW
webui/src/pages/Dashboard.tsx                       # NEW
webui/src/pages/Providers.tsx                       # NEW
webui/src/pages/ProviderWizard.tsx                  # NEW
webui/src/pages/ApiKeys.tsx                         # NEW
webui/src/pages/Settings.tsx                        # NEW
webui/src/pages/ModelCatalog.tsx                    # NEW
.env.example                                        # NEW
```

### Files to MODIFY (existing)

```
C:\Projects\ollama-stack\docker-compose.yml         # MODIFY: add ollarouter service, update open-webui URL
```

---

## 11. ADR References

The following architecture decisions from Phase 6 are materialized in this plan:

| ADR | Enforced In | Build Step |
|-----|------------|------------|
| ADR-001 (Go 1.23) | `go.mod` → `go 1.23`, Dockerfile → `golang:1.23-alpine` | Sprint 1.1 |
| ADR-002 (chi router) | `go.mod` dep, `router/router.go` | Sprint 5.1 |
| ADR-003 (Ollama API surface) | `handler_chat.go`, `handler_generate.go` | Sprint 5.3-5.4 |
| ADR-004 (Provider adapter interface) | `provider/provider.go` | Sprint 2.1 |
| ADR-005 (NDJSON in → SSE out) | `streaming/` package | Sprint 3.1-3.5 |
| ADR-006 (Zero external DB) | In-memory in `tracking/store.go` | Sprint 7.2 |
| ADR-007 (YAML + env vars) | `config/parser.go` | Sprint 1.3 |
| ADR-008 (Embedded web UI) | `webui/embed.go` | Sprint 8.10 |
| ADR-009 (Token bucket) | `auth/rate_limit.go` | Sprint 6.2 |
| ADR-010 (Port 11435) | `config.go` defaults, Dockerfile | Sprint 1.2, 1.6 |

---

## 12. Open Items for Implementation

| # | Issue | Impact | When to Resolve |
|---|-------|--------|-----------------|
| OI-1 | `providers.yaml` path resolution — relative to binary vs. absolute path | Config loading on startup | Sprint 1.3 (parser) |
| OI-2 | Version injection at build time: `main.version` ldflag | /api/version reporting | Sprint 5.10 (main.go) |
| OI-3 | Admin UI authentication — same API keys or separate? | /admin/* access control | Sprint 8.11 |
| OI-4 | Health check subcommand port must match runtime port | Docker HEALTHCHECK | Sprint 5.10 |
| OI-5 | File persistence write frequency — configurable or hardcoded 60s? | tracking/persistence.go | Sprint 7.3 |
| OI-6 | Web UI asset cache-busting on embedded rebuild | go:embed with versioned dir | Sprint 8.10 |

---

*End of Phase 7 — Implementation Plan. Ready for Phase 8 Build (Sprint 1).*
