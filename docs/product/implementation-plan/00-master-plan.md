# OllaRouter — Master Implementation Plan

**Version:** 0.1.0
**Date:** 2026-05-20
**Status:** Approved (Phase 7)
**Architecture Reference:** `docs/product/architecture/`
**Backlog Reference:** `docs/product/backlog/ollarouter-backlog.md`

---

## 1. Project Overview

OllaRouter is a multi-provider LLM gateway exposing an Ollama-compatible API. Single Go binary, zero external DB, embedded web UI, YAML config, port 11435.

**Repository root:** `C:\Projects\ollama-stack\`
**Module root:** `C:\Projects\ollama-stack\ollarouter\`

---

## 2. Sprint Structure (11 Sprints)

| Sprint | EPICs | Focus | Files Created | Depends On |
|--------|-------|-------|---------------|------------|
| **1** | EPIC-1 | Project scaffold, config parser, logger, Dockerfile | ~8 | None |
| **2** | EPIC-1 (FEAT-1.4) | Provider interface + 4 adapters | ~10 | Sprint 1 |
| **3** | EPIC-1 (FEAT-1.1) | Streaming engine (SSE→NDJSON) | ~6 | Sprint 2 |
| **4** | EPIC-1 (FEAT-1.5) | Retry, fallback, circuit breaker | ~4 | Sprint 2, 3 |
| **5** | EPIC-1 (all) | HTTP router, all API handlers, main.go | ~10 | Sprint 4, 3, 2, 1 |
| **6** | EPIC-3 | Auth middleware, rate limiter | ~3 | Sprint 5 |
| **7** | EPIC-4, EPIC-2 | Tracking, admin API handlers | ~7 | Sprint 5, 6 |
| **8** | EPIC-2 | Admin web UI (React SPA) | ~12 | Sprint 7 |
| **9** | EPIC-5 | ChatGPT Plus adapter | ~3 | Sprint 2, 3 |
| **10** | EPIC-4, EPIC-7 | Stats dashboard, model catalog UI | ~4 | Sprint 8, 7 |
| **11** | Operations | Docker compose integration, example config | ~3 | Sprint 5 |

---

## 3. Dependency Chain

```
Sprint 1 (scaffold)
   │
   ▼
Sprint 2 (providers) ──► Sprint 3 (streaming)
   │                          │
   └──────────┬───────────────┘
              ▼
         Sprint 4 (engine)
              │
              ▼
         Sprint 5 (router + main)
           │     │     │
           ▼     ▼     ▼
      Sprint 6  Sprint 7  Sprint 8
      (auth)    (track)   (web UI)
           │     │     │
           └──┬──┘     │
              ▼        │
         Sprint 9      │
         (ChatGPT)     │
              │        │
              ▼        ▼
         Sprint 10 (stats UI + model catalog)
              │
              ▼
         Sprint 11 (docker-compose)
```

---

## 4. File Manifest

All Go files under `ollarouter/cmd/` and `ollarouter/internal/`. All web UI files under `webui/`.

| Directory | Files | Package | Purpose |
|-----------|-------|---------|---------|
| `cmd/ollarouter/` | 1 | `main` | Entry point, healthcheck subcommand |
| `internal/config/` | 3 | `config` | YAML parsing, env interpolation, struct definitions |
| `internal/logger/` | 1 | `logger` | Structured JSON logger (slog) |
| `internal/provider/` | 3 | `provider` | Interface, factory, registry |
| `internal/provider/openai/` | 2 | `openai` | OpenAI adapter |
| `internal/provider/anthropic/` | 2 | `anthropic` | Anthropic adapter |
| `internal/provider/openaicompat/` | 2 | `openaicompat` | OpenAI-compatible adapter |
| `internal/provider/ollama/` | 2 | `ollama` | Ollama passthrough adapter |
| `internal/provider/chatgpt/` | 2 | `chatgpt` | ChatGPT Plus adapter |
| `internal/streaming/` | 6 | `streaming` | SSE→NDJSON translators |
| `internal/engine/` | 4 | `engine` | Model router, retry, fallback, circuit breaker |
| `internal/auth/` | 2 | `auth` | API key auth, rate limiter |
| `internal/tracking/` | 3 | `tracking` | Token/cost counting, store, persistence |
| `internal/admin/` | 5 | `admin` | Admin API handlers |
| `internal/router/` | 10 | `router` | Chi router, middleware, all HTTP handlers |
| `internal/webui/` | 1 | `webui` | go:embed of admin SPA |
| `webui/` | ~15 | — | React + Vite + Tailwind admin SPA |

---

## 5. Go Module Dependencies

| Package | Import Path | Version | Used In |
|---------|-------------|---------|---------|
| chi | `github.com/go-chi/chi/v5` | v5.1.0 | Router |
| cors | `github.com/go-chi/cors` | v1.2.1 | Router (CORS middleware) |
| yaml.v3 | `gopkg.in/yaml.v3` | v3.0.1 | Config parser |
| prometheus/client_golang | `github.com/prometheus/client_golang` | v1.20.0 | Metrics |
| slog | `log/slog` | stdlib | Logger |

---

## 6. Key Design Contracts

### Provider Interface (`internal/provider/provider.go`)
```go
type Provider interface {
    Name() string
    Type() string
    Chat(ctx, *ChatRequest) (*ChatResponse, error)
    ChatStream(ctx, *ChatRequest, StreamHandler) error
    Generate(ctx, *GenerateRequest) (*GenerateResponse, error)
    GenerateStream(ctx, *GenerateRequest, StreamHandler) error
    ListModels(ctx) ([]ModelInfo, error)
    CheckHealth(ctx) ProviderHealth
}
```

### Main Application Struct (`cmd/ollarouter/main.go`)
```go
type App struct {
    Config    *config.Config
    Logger    *slog.Logger
    Registry  *provider.Registry
    Engine    *engine.Engine
    Auth      *auth.Authenticator
    RateLimit *auth.RateLimiter
    Tracker   *tracking.Tracker
    Server    *http.Server
}
```

### Router Setup (`internal/router/router.go`)
```go
func NewRouter(app *App) http.Handler {
    r := chi.NewRouter()
    // Middleware chain
    r.Use(middleware.RequestID)
    r.Use(middleware.RequestLogger(app.Logger))
    r.Use(middleware.Recoverer)
    r.Use(middleware.CORS)
    // Auth (optional)
    if app.Auth != nil {
        r.Use(app.Auth.Middleware)
        r.Use(app.RateLimit.Middleware)
    }
    // Routes
    r.Post("/api/chat", app.handleChat)
    r.Post("/api/generate", app.handleGenerate)
    r.Get("/api/tags", app.handleTags)
    r.Get("/api/version", app.handleVersion)
    r.Post("/api/show", app.handleShow)
    r.Get("/health", app.handleHealth)
    r.Get("/health/providers", app.handleHealthProviders)
    r.Get("/metrics", app.handleMetrics)
    // Admin routes
    if app.AdminUI != nil {
        r.Mount("/admin", adminRouter(app))
    }
    return r
}
```

---

## 7. Docker Build

**Context:** `C:\Projects\ollama-stack\` (build sends `./ollarouter/` to Docker)
**Dockerfile:** `./ollarouter/Dockerfile`
**Image:** `ollarouter:latest`
**Port:** `11435` (configurable via `OLLAROUTER_PORT`)
**Health:** `CMD ["/app/ollarouter", "healthcheck"]`

---

## 8. Verification Gates

| Gate | When | Command |
|------|------|---------|
| Unit tests | After each sprint | `go test ./internal/... -v -count=1` |
| Build | After any Go change | `go build ./cmd/ollarouter` |
| Lint | After each sprint | `golangci-lint run ./...` |
| Integration | After Sprint 5 | Manual curl tests (see checkpoint) |
| Docker build | After Sprint 1, 11 | `docker build -t ollarouter:test .` |
| Web UI build | After Sprint 8 | `cd webui && npm run build` |
| Full stack | After Sprint 11 | `docker compose up -d && curl localhost:11435/health` |

---

*Refer to `docs/product/traceability/checkpoints/07-implementation-plan.md` for the complete per-sprint file-by-file breakdown.*
