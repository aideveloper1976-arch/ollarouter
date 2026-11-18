# Phase 13 — Code Review Gate: OllaRouter

**Date:** 2026-05-20
**Agent:** code-reviewer
**Project:** ollama-stack (`C:\Projects\ollama-stack`)
**Scope:** OllaRouter multi-provider LLM gateway (`C:\Projects\ollama-stack\ollarouter`)
**Previous Gate:** Automated Code Regression ✅ (Phase 10), Browser UI Regression ✅ (Phase 11), UX Review ✅ (Phase 12)

---

## Review Summary

**Gate Decision: ⚠️ BLOCKING ISSUES FOUND — Must fix before merge**

The codebase is well-structured overall with clear separation of concerns, proper Go idioms, and comprehensive architecture documentation. However, **7 blocking issues were identified** — the most critical being that the **virtual model fallback chain feature is non-functional**, which is a core architectural contract. Additionally, the Anthropic streaming adapter ignores client disconnect signals, the rate limiter ignores per-key configuration, and the settings update admin endpoint is a no-op.

### Source Files Reviewed (40 Go files)
All 37 `.go` files in `ollarouter/` plus `Dockerfile`, `Dockerfile.check`, `docker-compose.yml` reviewed.

### Architecture Documents Referenced
- `docs/product/architecture/01-system-architecture.md`
- `docs/product/architecture/02-provider-adapter-contract.md`
- `docs/product/architecture/03-docker-architecture.md`

### Prior Evidence Reviewed
- Phase 8 Build Checkpoint
- Phase 9 Dev Self-Verification Checkpoint
- Phase 10 Code Regression Checkpoint
- Phase 11.1 Browser UI Regression Rerun Checkpoint

---

## Findings by Severity

### 🔴 BLOCKING — Must Fix Before Merge

#### BLOCKING-1: Virtual model fallback chain feature is non-functional (Core Architecture Contract Drift)

| Attribute | Detail |
|-----------|--------|
| **Files** | `internal/engine/router.go:81-90`, `internal/engine/fallback.go:108-117` |
| **Severity** | **HIGH** — Core architectural feature specified in `01-system-architecture.md` §2, §3.1, §8 |
| **Evidence** | `ResolveVirtualModel()` in router.go only checks if a model name exists in model cache — it never looks up `config.VirtualModelConfig.FallbackChain`. `BuildFallbackChain()` in fallback.go returns the virtual model name as a single-element slice without expanding into the chain. The `VirtualModelConfig` from config parsing is loaded (parser_test.go:108-135) but **never wired** to the engine. |
| **Impact** | Virtual models like `fast-chat` with `fallback_chain: [openai-main/gpt-4o-mini, local-ollama/*]` will fall through as "model not found" instead of executing the chain. | 
| **Fix** | Pass `config.VirtualModels` to `FallbackExecutor` / `ModelResolver`. Implement proper chain expansion from `VirtualModelConfig.FallbackChain`. Wire config to engine in `main.go`. |

#### BLOCKING-2: Health check endpoint blocks sequentially on all providers (Performance)

| Attribute | Detail |
|-----------|--------|
| **Files** | `internal/router/handler_health.go:22-33` |
| **Severity** | **HIGH** — The `/health` endpoint calls `CheckHealth()` synchronously for each provider. With 5+ providers with 5-30s timeouts, the endpoint can block for 25-150s. |
| **Impact** | Docker HEALTHCHECK (30s interval, 3s timeout) could thrash. Prometheus scraping could fail. Health endpoint unavailable under normal cloud provider latency. |
| **Fix** | Use `errgroup` or `sync.WaitGroup` with goroutines to check providers concurrently with a context timeout. |

#### BLOCKING-3: Anthropic streaming adapter ignores handler errors (Resource Leak)

| Attribute | Detail |
|-----------|--------|
| **Files** | `internal/provider/anthropic/adapter.go:143-166` |
| **Severity** | **HIGH** — The `StreamHandler` callback returns `error` for client disconnect signaling. The Anthropic adapter calls `handler()` on lines 144, 152, 163 but **never checks the return value**. |
| **Impact** | When a client disconnects mid-stream, the `NDJSONWriter.WriteChunk` returns an error. The OpenAI adapter correctly checks this (`openai/adapter.go:144`), but the Anthropic adapter continues processing and sending data to a closed connection, wasting resources and potentially causing panics. |
| **Fix** | Check `if err := handler(...); err != nil { return err }` for all handler calls in Anthropic `ChatStream`. |

#### BLOCKING-4: Rate limiter ignores per-key rate limits from config (Contract Drift)

| Attribute | Detail |
|-----------|--------|
| **Files** | `internal/auth/rate_limit.go:108`, `internal/auth/rate_limit.go:114-118` |
| **Severity** | **MEDIUM** — All API keys get the default 60 req/min rate limit regardless of `APIKeyConfig.RateLimit` from config. |
| **Evidence** | The `SetRateLimit()` method exists but is **never called** from config loading in `main.go`. The `getBucket` method always creates buckets with `NewTokenBucket(60, 1.0)`. |
| **Impact** | Config-level rate limiting doesn't work as documented. Budget controls via rate limiting are ineffective. |
| **Fix** | After `NewAuthenticator`, iterate `cfg.APIKeys` and call `rateLimiter.SetRateLimit(key.Name, key.RateLimit)` for each configured key. |

#### BLOCKING-5: Settings update admin endpoint is a no-op (Broken Feature)

| Attribute | Detail |
|-----------|--------|
| **Files** | `internal/router/admin_handlers.go:245-253` |
| **Severity** | **MEDIUM** — `handleAdminUpdateSettings` parses the request body into a `map[string]interface{}` then **discards it** and always returns `{"status":"settings updated"}`. |
| **Impact** | Admin UI "Save Settings" will appear to succeed but no changes take effect. Users cannot change log level, rate limiting, or other settings at runtime. |
| **Fix** | Implement actual settings application: update `Config.Settings` fields based on request body, or remove the endpoint if runtime reconfiguration is out of scope. |

#### BLOCKING-6: OpenAI-compatible adapter omits TokenUsage from Generate responses (Inconsistency)

| Attribute | Detail |
|-----------|--------|
| **Files** | `internal/provider/openaicompat/adapter.go:163-177` |
| **Severity** | **MEDIUM** — `Generate()` in the OpenAI-compatible adapter creates a `GenerateResponse` with only `Model` and `Response`, **dropping `TokenUsage`**. Compare with `openai/adapter.go:182-186` which correctly propagates it. |
| **Impact** | Cost tracking for OpenAI-compatible providers (Groq, Azure, Together, vLLM) will show zero token usage for generate requests. |
| **Fix** | Add `TokenUsage: chatResp.TokenUsage` to the `GenerateResponse` in `openaicompat/adapter.go:173-176`. |

#### BLOCKING-7: Insufficient test coverage (Quality Risk)

| Attribute | Detail |
|-----------|--------|
| **Files** | All packages: `internal/provider/openai/`, `internal/provider/anthropic/`, `internal/provider/openaicompat/`, `internal/provider/ollama/`, `internal/engine/`, `internal/auth/`, `internal/tracking/`, `internal/router/`, `internal/logger/` |
| **Severity** | **MEDIUM** — Only 3 of 13 packages have tests (13 tests total across config, streaming, chatgpt). |
| **Missing** | No tests for: OpenAI adapter, Anthropic adapter, OpenAI-compatible adapter, Ollama adapter, ModelResolver, FallbackExecutor, RetryEngine, CircuitBreaker, API key auth, rate limiter, token tracking, any HTTP handler. |
| **Impact** | High risk of regressions in core routing, failover, and adapter translation logic. |
| **Fix** | At minimum: unit tests for CircuitBreaker (state machine), RetryEngine (backoff math), FallbackExecutor (chain execution), and adapter-specific tests for OpenAI/Anthropic streaming. |

---

### 🟡 NON-BLOCKING (Recommended fixes)

#### SUGGESTION-1: Admin SPA build missing from Dockerfile
- **File:** `Dockerfile`
- **Issue:** No `cd webui && npm ci && npm run build` step. The embedded SPA must be built externally before `docker build`. This is documented but fragile.
- **Recommendation:** Add a multi-stage build with Node for the SPA, or add a build-time check that `internal/webui/dist/` is non-empty.

#### SUGGESTION-2: Docker dependency caching not optimized
- **File:** `Dockerfile:9-12`
- **Issue:** `COPY . .` then `RUN go mod tidy` — caching is suboptimal. Changes to any source file invalidate the dependency layer.
- **Recommendation:** Separate into `COPY go.mod go.sum ./`, `RUN go mod download`, then `COPY . .`.

#### SUGGESTION-3: Unnecessary build dependencies
- **File:** `Dockerfile:4`
- **Issue:** `RUN apk add --no-cache gcc musl-dev` — unnecessary since `CGO_ENABLED=0`.
- **Recommendation:** Remove to reduce build time.

#### SUGGESTION-4: Architecture doc contract drift
- **File:** `docs/product/architecture/01-system-architecture.md §4.1`
- **Issue:** Shows `Chat(ctx, req *ChatRequest, stream bool)` in architecture doc, but code uses `Chat(ctx, req *ChatRequest)` with stream inside `ChatRequest`.
- **Recommendation:** Update architecture doc to match code (doc `02-provider-adapter-contract.md` already has the correct signature).

#### SUGGESTION-5: writeError uses string concatenation instead of json.Marshal
- **File:** `internal/router/router.go:136-140`
- **Issue:** Hand-rolled `escapeJSON()` for error responses. While it handles basic cases, using `json.Marshal` on a struct is more robust.
- **Recommendation:** Use `json.Marshal(struct{Error string `json:"error"`}{message})` instead.

#### SUGGESTION-6: NDJSON WriteError silently swallows marshal errors
- **File:** `internal/streaming/translator.go:106`
- **Issue:** `data, _ := json.Marshal(olChunk)` — error discarded silently.
- **Recommendation:** Return the error from `json.Marshal` if it occurs.

#### SUGGESTION-7: OpenAI Generate sends empty system message
- **File:** `internal/provider/openai/adapter.go:171`
- **Issue:** When `req.System` is empty, still sends `{Role: "system", Content: ""}` message.
- **Recommendation:** Only include system message when `req.System != ""`.

#### SUGGESTION-8: API key masking inconsistency between packages
- **Files:** `internal/logger/logger.go:60-65`, `internal/auth/api_key.go:174-179`
- **Issue:** Logger masks `key[:3] + "..." + key[len(key)-4:]`, Auth masks `key[:4] + "..." + key[len(key)-4:]`.
- **Recommendation:** Use consistent masking pattern across the codebase.

#### SUGGESTION-9: Go 1.23 `crypto/rand.Read` error ignored
- **File:** `internal/auth/api_key.go:184, 190`
- **Issue:** `rand.Read(bytes)` return error ignored. From Go 1.23+ this always returns nil error, but conventions suggest checking.
- **Recommendation:** Add `if _, err := rand.Read(bytes); err != nil { ... }` for future-proofing.

#### SUGGESTION-10: Missing `Stream` field in Anthropic request body
- **File:** `internal/provider/anthropic/adapter.go:258-261`
- **Issue:** `anthropicRequest` has `Stream` field but `buildRequest` never sets it. While `Accept: text/event-stream` header works, the JSON body should also include `"stream": true`.
- **Recommendation:** Set `anthReq.Stream = req.Stream` in `buildRequest`.

---

## Required Fixes (Summary)

| # | Priority | Area | File(s) | Summary |
|---|----------|------|---------|---------|
| 1 | 🔴 HIGH | Engine | `engine/router.go`, `engine/fallback.go`, `cmd/ollarouter/main.go` | Wire virtual model config to engine; implement proper fallback chain expansion |
| 2 | 🔴 HIGH | Router | `router/handler_health.go` | Make health checks concurrent with context timeout |
| 3 | 🔴 HIGH | Anthropic | `provider/anthropic/adapter.go` | Check handler() error returns in streaming loop |
| 4 | 🔴 MEDIUM | Auth | `auth/rate_limit.go`, `cmd/ollarouter/main.go` | Apply per-key rate limits from config |
| 5 | 🔴 MEDIUM | Admin | `router/admin_handlers.go` | Implement actual settings update or remove endpoint |
| 6 | 🔴 MEDIUM | OpenAI-Compat | `provider/openaicompat/adapter.go` | Propagate TokenUsage in Generate responses |
| 7 | 🔴 MEDIUM | Testing | Multiple packages | Add tests for engine, auth, tracking, and remaining adapters |

---

## Test / Build Risk Notes

| Risk | Assessment | Mitigation |
|------|-----------|------------|
| **Fallback chain not working** | HIGH — virtual models will silently fail | Fix #1 required before any release |
| **Health check timeout** | MEDIUM — Docker HEALTHCHECK could kill container unnecessarily | Fix #2 should be prioritized |
| **Anthropic streaming leaks** | MEDIUM — goroutines/connections leak on client disconnect | Fix #3 required |
| **Rate limit bypase** | LOW — default 60/min still applies | Fix #4 low effort |
| **Settings endpoint confusion** | LOW — UI shows "saved" but nothing changes | Fix #5 or documentation |
| **Build size** | LOW — 16.8MB image is excellent | No action needed |
| **Module integrity** | NONE — `go mod verify` passes | Clean |
| **Static analysis** | NONE — `go vet` and `staticcheck` pass at v0.5.1 | Clean |

---

## Final Decision

**⚠️ DO NOT MERGE — 7 blocking issues must be resolved first.**

The codebase is well-architected with strong separation of concerns, proper Go idioms, secure API key hashing, and excellent architecture documentation. However, the virtual model fallback chain (a core architectural feature) is non-functional, and several other contract-drift issues prevent this code from being release-ready.

### Gate Status by Review Criteria

| # | Criterion | Rating | Notes |
|---|-----------|--------|-------|
| 1 | Correctness | ⚠️ Partial | Fallback chain not functional; Anthropic handler errors ignored |
| 2 | Contract drift | ⚠️ Drift found | Virtual models not wired; rate limit per-key ignored; system arch doc has stale interface |
| 3 | Security patterns | ✅ Good | API key hashing, masking, non-root user, read-only config, distroless base |
| 4 | Error handling | ⚠️ Partial | Anthropic ignores handler errors; WriteError silently fails |
| 5 | Testing | ❌ Insufficient | 13 tests for 13 packages; engine, auth, adapters untested |
| 6 | Code style | ✅ Good | Consistent Go idioms, proper mutex usage, clean interfaces |
| 7 | Documentation | ✅ Good | Architecture docs are comprehensive; some missing Go doc comments |
| 8 | Dockerfile quality | ✅ Good | Multi-stage, distroless, non-root, HEALTHCHECK (minor optimization suggestions) |
| 9 | docker-compose integration | ✅ Good | Properly integrated with ollama + open-webui |
| 10 | Config parsing | ✅ Good | YAML with env var interpolation, validation, defaults |

---

*End of Phase 13 — Code Review Gate*
