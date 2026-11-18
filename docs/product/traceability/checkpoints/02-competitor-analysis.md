# Phase 2 — Competitor Analysis: OllaRouter

**Date:** 2026-05-20  
**Agent:** market-researcher  
**Project:** ollama-stack (C:\Projects\ollama-stack)  
**Phase 1 Reference:** `docs/product/traceability/checkpoints/01-market-research.md`  
**Target:** OllaRouter — Lightweight multi-provider LLM gateway with Ollama-compatible endpoints

---

## 1. Competitors Reviewed (Deep Analysis)

Ten competitors/projects were analysed via live page extraction across two tiers:
- **Tier 1 — General LLM Gateways:** LiteLLM, Portkey, OpenRouter, Bifrost (Maxim), any-llm (Mozilla)
- **Tier 2 — Ollama-Niche & Adjacent Tools:** Ollama Proxy (AstroAir), Olla (thushan/olla), 1LCB/ollama-gateway, LiteLLMOllamaAdapter (pure-san), zhaees/chatgpt-proxy

---

### 1.1 LiteLLM (BerriAI) — Tier 1

| Attribute | Detail |
|-----------|--------|
| **Stars** | 47.7k |
| **License** | MIT |
| **Language** | Python |
| **API Format** | OpenAI `/v1/chat/completions` (NOT Ollama-compatible) |
| **Deployment** | Docker, K8s (Helm), Terraform |
| **Source** | [GitHub](https://github.com/BerriAI/litellm), [Docs](https://docs.litellm.ai/docs/) |

**Key Capabilities (from extracted docs):**
- 100+ LLM providers via OpenAI-compatible SDK + Proxy Server
- Router with retry/fallback/load balancing across deployments
- Virtual keys with per-key/team/user budgets
- Centralized logging, guardrails, caching
- Admin UI for monitoring and management
- A2A agent gateway and MCP gateway
- Observability callbacks (Langfuse, MLflow, Helicone)
- Ollama as a **backend provider** (calls via `ollama_chat/` prefix against local Ollama server)
- Latest release: May 20, 2026 — 39,318+ commits, very active

**Missing for OllaRouter:**
- ❌ **No Ollama-compatible API surface** — clients must speak OpenAI format (no GET /api/tags, POST /api/chat, POST /api/generate)
- ❌ **Requires PostgreSQL + Redis for production** (docker-compose.yml ships with Prisma)
- ❌ **No ChatGPT Plus web proxy mode**
- ❌ **Heavy Python dependency** (~500MB+ container)
- ❌ **Complex production setup** — separate gateway/backend/UI components

**Ollama Integration:**
- Supports Ollama as a **provider** via `ollama/` model prefix
- Sends requests to local `http://localhost:11434` — passthrough only
- Does not expose Ollama endpoints to clients; clients must use `/chat/completions`

---

### 1.2 Portkey AI Gateway — Tier 1

| Attribute | Detail |
|-----------|--------|
| **Stars** | 11.8k |
| **License** | MIT |
| **Language** | TypeScript/Node |
| **API Format** | OpenAI-compatible + Portkey SDK |
| **Deployment** | Docker, Cloudflare Workers, Node.js |
| **Source** | [GitHub](https://github.com/Portkey-AI/gateway), [Docs](https://docs.portkey.ai/docs/product/ai-gateway) |

**Key Capabilities:**
- 1600+ models, 45+ providers
- Universal API (REST + SDKs)
- Cache (simple & semantic), MCP support
- Fallbacks, conditional routing, multimodality
- Automatic retries, circuit breaker, load balancing
- Canary testing, gRPC (Beta), request timeout
- Budget limits, rate limits
- Custom host routing to private/local models
- Quick start: `npx @portkey-ai/gateway`
- Pricing: Open-source + Hosted ($49/mo Production) + Enterprise

**Missing for OllaRouter:**
- ❌ **No Ollama API compatibility** — OpenAI format only
- ❌ **Complex configuration** — gateway configs required
- ❌ **Requires Node.js runtime**
- ❌ **No ChatGPT Plus web proxy mode**
- ❌ **No local-first design** — targets cloud/enterprise

---

### 1.3 OpenRouter — Tier 1

| Attribute | Detail |
|-----------|--------|
| **Type** | Managed SaaS (NOT self-hostable) |
| **API Format** | OpenAI-compatible API |
| **Pricing** | Free tier (50 req/day), PAYG ($10+ credits), 5.5% platform fee, Enterprise |
| **Source** | [Docs: Model Fallbacks](https://openrouter.ai/docs/guides/routing/model-fallbacks) |

**Key Capabilities:**
- 400+ models, 60+ providers
- Model fallbacks with automatic failover (`models` parameter array)
- Context length validation, moderation flag handling, rate-limit recovery
- Provider selection/routing with auto-exacto matching
- Response caching, structured outputs, service tiers
- Sovereign AI (private deployment option)
- Cleans dashboard, searchable model catalog, transparent billing

**Missing for OllaRouter:**
- ❌ **Fully managed SaaS** — no self-hosted option
- ❌ **No local Ollama integration** — cannot be added to docker-compose.yml
- ❌ **No Ollama API** — OpenAI-compatible only
- ❌ **5.5% platform markup** — not free
- ❌ **No ChatGPT Plus web proxy mode**

---

### 1.4 Bifrost by Maxim AI — Tier 1

| Attribute | Detail |
|-----------|--------|
| **Stars** | 5.1k |
| **License** | Open-source + Enterprise |
| **Language** | Go |
| **API Format** | OpenAI `/v1/chat/completions` |
| **Deployment** | Docker, Helm, NPX (`npx @maximhq/bifrost`) |
| **Source** | [GitHub](https://github.com/maximhq/bifrost), [Docs](https://www.getmaxim.ai/bifrost/) |

**Key Capabilities:**
- 23+ providers (OpenAI, Anthropic, Bedrock, Vertex, Azure, Cerebras, Cohere, Mistral, **Ollama**, Groq)
- 11µs overhead at 5,000 RPS (fastest in class — 50x faster than LiteLLM)
- Adaptive load balancing, auto-failover, semantic caching
- MCP gateway, custom plugins, governance
- Web UI for visual configuration + real-time monitoring
- Budget management with virtual keys, teams, customer budgets
- SSO (Google/GitHub), Prometheus metrics, distributed tracing
- Zero-config startup, drop-in replacement for OpenAI/Anthropic SDKs
- 100% success rate at 5k RPS benchmarks

**Missing for OllaRouter:**
- ❌ **No Ollama API surface** — OpenAI format only
- ❌ **Enterprise-focused** — advanced features are paid/licensed
- ❌ **No ChatGPT Plus web proxy mode**
- ❌ **Uses file-based + optional DB configuration**
- ❌ **Not local-first** — designed for cloud/enterprise deployments

**Ollama Provider Support:**
- Lists Ollama as a supported provider (alongside OpenAI, Anthropic, etc.)
- Routes OpenAI-format requests to Ollama backends
- Does NOT expose the Ollama API surface

---

### 1.5 any-llm (Mozilla AI) — Tier 1

| Attribute | Detail |
|-----------|--------|
| **License** | Open-source (Mozilla) |
| **Language** | Python (FastAPI) |
| **API Format** | OpenAI-compatible |
| **Deployment** | Docker + PostgreSQL |
| **Source** | [Docs](https://mozilla-ai.github.io/any-llm/gateway/overview/), [Blog](https://blog.mozilla.ai/control-llm-spend-and-access-with-any-llm-gateway/) |

**Key Capabilities:**
- Multi-provider routing with format `provider:model` (e.g., `openai:gpt-4o-mini`)
- Smart budget management — shared tiers with daily/weekly/monthly resets
- Virtual API keys with expiration, activation, spend tracking
- Complete usage analytics (token counts, per-request costs, user attribution)
- Master key + virtual API key authentication
- Docker + K8s deployment
- <50ms gateway overhead

**Missing for OllaRouter:**
- ❌ **No Ollama API compatibility** — OpenAI format only
- ❌ **Requires PostgreSQL** — not stateless
- ❌ **No ChatGPT Plus web proxy mode**
- ❌ **No local-first design**
- ❌ **FastAPI/Python dependency**

---

### 1.6 Ollama Proxy (AstroAir/ollama-proxy) — Tier 2 ★ Closest Competitor

| Attribute | Detail |
|-----------|--------|
| **Stars** | Unknown (repo not publicly accessible) |
| **Language** | Python (asyncio) |
| **API Format** | **Ollama-native API** (Ollama endpoints) |
| **Deployment** | Docker, Docker Compose |
| **Source** | [Docs](https://ollama-proxy.readthedocs.io/en/latest/), [Multi-Provider Docs](https://ollama-proxy.readthedocs.io/en/latest/multi-provider/) |

**Key Capabilities:**
- ✅ **Ollama-native API** — `/api/chat`, `/api/generate`, `/api/tags` exposed
- ✅ Multi-provider routing: OpenAI, Anthropic, Google Gemini, OpenRouter, Azure, AWS Bedrock, local Ollama
- ✅ Various routing strategies: model-based, capability-based, cost-optimized, round-robin, least-loaded, fastest-response
- ✅ Fallback strategies: next_available, retry_same, best_alternative
- ✅ Health monitoring with circuit breakers per provider
- ✅ Request/response transformation between Ollama and provider APIs
- ✅ Connection pooling, request queuing, rate limiting
- ✅ Async optimization (non-blocking I/O)
- ✅ Provider-specific health check endpoints (`/health/providers/openai`)
- ✅ Full Ollama API compatibility — can use with existing Ollama clients

**Missing for OllaRouter:**
- ❌ **No ChatGPT Plus web proxy mode** — only standard API providers
- ❌ **No stats/billing UI** — CLI/API only, no dashboard
- ❌ **Python-based** — heavier runtime (~100-200MB container)
- ❌ **Tied to OpenRouter** — first-class OpenRouter support, other providers are extensions
- ❌ **No aggregated model listing UI** — `/api/tags` works but no visual catalog
- ❌ **Limited community** — repo may be private/abandoned
- ❌ **No configuration UI** — env vars and .env files only

**Why it's the closest competitor:**
- Only other tool besides OllaRouter that exposes native Ollama API surface
- Same core concept: use Ollama clients, route to cloud providers
- But heavier (Python), no ChatGPT proxy, no stats dashboard, no web UI

---

### 1.7 Olla (thushan/olla) — Tier 2

| Attribute | Detail |
|-----------|--------|
| **Stars** | 224 |
| **License** | MIT |
| **Language** | Go (v1.24) |
| **API Format** | OpenAI + Anthropic (NOT Ollama-native) |
| **Deployment** | Single binary, Docker, multi-arch (ARM64, RPi4+) |
| **Source** | [GitHub](https://github.com/thushan/olla), [Blog Post](https://www.thushanfernando.com/2026/05/olla-a-smart-load-balancer-for-your-llm-infrastructure/) |

**Key Capabilities:**
- Smart load balancing: priority-based, least-connections, round-robin
- Model discovery + unified model catalog across backends
- Health monitoring with circuit breakers and auto-recovery
- Sticky sessions for KV-cache-aware conversation pinning
- Anthropic ↔ OpenAI format translation
- Dual proxy engines: Sherpa (simple) and Olla (high-performance)
- Sub-millisecond endpoint selection, lock-free atomic stats
- Streaming-first design, <50MB RAM
- Supports: Ollama, vLLM, LM Studio, llama.cpp, SGLang, LMDeploy, Docker Model Runner, LiteLLM
- CLI + Docker + single YAML config

**Missing for OllaRouter:**
- ❌ **No cloud provider API routing** — only routes to local inference backends
- ❌ **No Ollama-native API** — exposes OpenAI/Anthropic format, not `/api/chat`
- ❌ **No ChatGPT Plus web proxy mode**
- ❌ **No usage cost tracking** — no billing analytics
- ❌ **No provider configuration UI** — YAML only

**What Olla does well that OllaRouter should note:**
- Ultra-lightweight (<50MB, single Go binary)
- Model unification across backends
- Health monitoring with circuit breakers
- Sticky session routing
- Streaming-first architecture

---

### 1.8 1LCB/ollama-gateway — Tier 2

| Attribute | Detail |
|-----------|--------|
| **Stars** | 16 |
| **License** | MIT |
| **Language** | Go |
| **API Format** | **Ollama-native API** |
| **Deployment** | Docker |
| **Source** | [GitHub](https://github.com/1LCB/ollama-gateway) |

**Key Capabilities:**
- ✅ **Ollama-native API passthrough** — proxies to Ollama servers
- Round-robin load balancing across Ollama instances
- Rate limiting with configurable windows
- API key authentication
- IP allowlist/denylist security filters
- Prometheus metrics endpoint (requests, latency, per-IP, success/failure)
- Hot configuration reload without restart
- API key generation via HTTP endpoint
- Lightweight Go binary

**Missing for OllaRouter:**
- ❌ **Ollama servers only** — no cloud provider routing
- ❌ **No ChatGPT proxy**
- ❌ **No stats UI** — Prometheus metrics only (no dashboard)
- ❌ **No cost tracking**
- ❌ **No model listing aggregation** — just passes through requests
- ❌ **No configuration UI**
- ❌ **Very small community** (16 stars, 1 contributor, last update Jan 2025)

---

### 1.9 LiteLLMOllamaAdapter (pure-san) — Tier 2

| Attribute | Detail |
|-----------|--------|
| **Stars** | 1 |
| **License** | MIT |
| **Language** | Python (Flask) |
| **API Format** | **Ollama-native API** |
| **Deployment** | Local Python process |
| **Source** | [GitHub](https://github.com/pure-san/LiteLLMOllamaAdapter) |

**Key Capabilities:**
- ✅ **Ollama-native API** — `/api/chat`, `/api/generate`, `/api/show`, `/api/tags`, `/api/version`
- Forwards to OpenRouter via LiteLLM
- Ollama-style model names exposed through `/api/tags`
- Ollama options support for generation parameters
- Base64 image input conversion for multimodal requests
- Local-only API key handling (OPENROUTER_API_KEY stays on user machine)

**Missing for OllaRouter:**
- ❌ **Minimal implementation** — only 5 endpoints, no embeddings, no model management
- ❌ **OpenRouter-only** — no direct OpenAI/Anthropic/etc. provider routing
- ❌ **No multi-provider fallback**
- ❌ **No stats UI or cost tracking**
- ❌ **No ChatGPT Plus proxy**
- ❌ **No Docker deployment** (local Python process only)
- ❌ **No configuration UI** — model list hardcoded in Python file
- ❌ **Very early stage** (1 star, last update May 2026, 3 commits)

---

### 1.10 zhaees/chatgpt-proxy — Tier 2

| Attribute | Detail |
|-----------|--------|
| **Stars** | 5 |
| **License** | MIT |
| **Language** | TypeScript (Bun) + Python |
| **API Format** | OpenAI `/v1/chat/completions` |
| **Deployment** | Local process (Bun + Python) |
| **Source** | [GitHub](https://github.com/zhaees/chatgpt-proxy) |

**Key Capabilities:**
- ✅ **ChatGPT Plus/Pro cookie-based proxy** — access GPT-4o, o3, o4-mini, GPT-5 family via API
- ✅ 30+ ChatGPT models available (GPT-5.5, deep-research, agent mode)
- ✅ Tool call emulation — adds XML-based tool calling to models that don't support it natively
- ✅ Cookie auto-refresh via curl_cffi Session (handles Cloudflare challenges, PoW)
- ✅ Cookie persistence across restarts (`.cookies.json`)
- ✅ Auto retry on 403 — refreshes Cloudflare cookies and retries
- ✅ Health check, cookie status, manual refresh endpoints

**Missing for OllaRouter:**
- ❌ **Single-provider** — ChatGPT only, no multi-provider routing
- ❌ **No Ollama API** — OpenAI format only
- ❌ **No local Ollama routing**
- ❌ **No cost tracking or stats dashboard**
- ❌ **No Docker deployment** (manual start with `./start.sh`)
- ❌ **No configuration UI**
- ❌ **Small community** (5 stars, 1 contributor, very new - April 2026)

**Why it's important:**
- Proves the ChatGPT Plus web proxy concept works
- Shows required architecture: TypeScript proxy + Python helper (curl_cffi)
- Demonstrates cookie management, PoW solving, and Cloudflare challenge handling
- OllaRouter should integrate similar provider type

---

## 2. Feature Comparison Matrix

### 2.1 Requirements × Competitor Matrix

| Requirement | LiteLLM | Portkey | OpenRouter | Bifrost | any-llm | OllamaProxy | Olla | ollama-gateway | LLMOllamaAdapter | chatgpt-proxy | **OllaRouter (planned)** |
|---|---|---|---|---|---|---|---|---|---|---|---|
| **Ollama API mirroring** | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | ✅ | ✅ | ❌ | **✅** |
| **Provider routing** | ✅ (100+) | ✅ (45+) | ✅ (60+) | ✅ (23+) | ✅ | ✅ (7) | ❌ (local only) | ❌ (Ollama only) | ❌ (OpenRouter only) | ❌ (1 provider) | **✅** |
| **Fallback/failover** | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | **✅** |
| **Provider/model listing** | ✅ (admin UI) | ✅ (dashboard) | ✅ (web) | ✅ (web UI) | ✅ | ✅ (/api/tags) | ✅ (/v1/models) | ❌ | ✅ (/api/tags) | ✅ (/v1/models) | **✅ (/api/tags)** |
| **Local Ollama passthrough** | ✅ (as provider) | ✅ (custom hosts) | ❌ | ✅ (as provider) | ❌ | ✅ (native) | ✅ (native) | ✅ (native) | ❌ (OpenRouter only) | ❌ | **✅ (native)** |
| **ChatGPT Plus web proxy** | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | **✅** |
| **Stats UI / dashboard** | ✅ (admin UI) | ✅ (dashboard) | ✅ (dashboard) | ✅ (web UI) | ✅ (analytics) | ❌ (API only) | ❌ (CLI only) | ❌ (metrics only) | ❌ | ❌ | **✅** |
| **Docker deployment** | ✅ (needs DB) | ✅ (standalone) | N/A (SaaS) | ✅ (standalone) | ✅ (needs PG) | ✅ (standalone) | ✅ (standalone) | ✅ (standalone) | ❌ | ❌ | **✅ (standalone)** |

### 2.2 Feature Depth Comparison

| Feature | LiteLLM | Portkey | Bifrost | OllamaProxy | Olla | **OllaRouter** |
|---|---|---|---|---|---|---|
| **Streaming support** | SSE (OpenAI) | SSE | SSE | NDJSON (Ollama) | SSE | **NDJSON (Ollama)** |
| **NDJSON→SSE translation** | N/A | N/A | N/A | ✅ | N/A | **✅** |
| **Aggregated model catalog** | ❌ (per-provider) | ❌ | ❌ | ✅ (unified) | ✅ (unified) | **✅ (unified)** |
| **Zero external deps** | ❌ (DB+Redis) | ✅ | ✅ | ✅ | ✅ | **✅** |
| **Auth (API keys)** | ✅ (virtual keys) | ✅ | ✅ | ✅ | ✅ | **✅** |
| **Rate limiting** | ✅ | ✅ | ✅ | ✅ | ✅ | **✅** |
| **Cost tracking** | ✅ (per-key) | ✅ | ✅ (budgets) | ❌ | ❌ | **✅** |
| **Configuration UI** | ✅ (admin panel) | ✅ (web console) | ✅ (web UI) | ❌ (env vars) | ❌ (YAML) | **✅** |
| **Health checks** | ✅ | ✅ | ✅ | ✅ | ✅ | **✅** |
| **Prometheus metrics** | ✅ | ❌ | ✅ | ❌ | ❌ | **✅ (planned)** |

---

## 3. Pricing Notes

| Tool | Base License | Transaction Fee | Self-Host Cost | Enterprise Tier |
|-----|-------------|-----------------|----------------|-----------------|
| **LiteLLM** | MIT | None (proxy only) | Server + optional DB (PG+Redis) | Yes (SSO, audit logs) |
| **Portkey** | MIT | None (proxy only) | Server ($49/mo Production hosted) | Yes |
| **OpenRouter** | Proprietary | 5.5% PAYG markup | N/A (managed) | Yes (bulk discounts) |
| **Bifrost** | OS + Enterprise | None (proxy only) | Server | Yes (features locked) |
| **any-llm** | Apache 2.0 | None (proxy only) | Server + PostgreSQL | N/A |
| **Ollama Proxy** | MIT | None (proxy only) | Server | N/A |
| **Olla** | MIT | None | Server (minimal) | N/A |
| **1LCB/ollama-gateway** | MIT | None | Server (minimal) | N/A |
| **LiteLLMOllamaAdapter** | MIT | None | Server | N/A |
| **zhaees/chatgpt-proxy** | MIT | None (ChatGPT sub cost) | Local process | N/A |
| **OllaRouter (planned)** | MIT (planned) | **None** | **Minimal (container only)** | N/A |

**Key observation:** All self-hosted LLM gateways charge **zero transaction fees** for the proxy layer. The cost is entirely server infrastructure + API provider costs. OpenRouter is the exception at 5.5%. OllaRouter should remain **free, open-source, zero-markup** — consistent with the market.

---

## 4. Architectural Comparison

| Aspect | LiteLLM | Portkey | Bifrost | OllamaProxy | Olla | **OllaRouter (planned)** |
|---|---|---|---|---|---|---|
| **Language** | Python | TypeScript | Go | Python | Go | **Go (recommended)** |
| **Container size** | ~500MB+ | ~200MB+ | ~20MB (Go binary) | ~200MB+ | ~10MB (Go binary) | **~10-20MB** |
| **External DB** | PostgreSQL+Redis | None | Optional (configstore) | None | None | **None** |
| **API surface** | OpenAI | OpenAI | OpenAI | Ollama | OpenAI+Anthropic | **Ollama** |
| **Config format** | YAML + env | Config object | JSON/Env | .env + env vars | YAML | **YAML + env** |
| **Startup time** | Seconds (Python) | Seconds (Node) | Milliseconds (Go) | Seconds (Python) | Milliseconds (Go) | **Milliseconds (Go)** |
| **Runtime profile** | 200-500MB RSS | 100-300MB RSS | 10-50MB RSS | 50-200MB RSS | <50MB RSS | **<50MB RSS** |

---

## 5. UX / Configuration Patterns

| Pattern | LiteLLM | Portkey | Bifrost | OllamaProxy | Olla | **OllaRouter (planned)** |
|---|---|---|---|---|---|---|
| **Quickstart** | `litellm --model gpt-4o` | `npx @portkey-ai/gateway` | `npx @maximhq/bifrost` | `python -m src.main` | `bash <(curl ...)` | **`docker compose up`** |
| **Web UI** | Admin panel (built-in) | Portkey Cloud console | Built-in web UI | None | CLI only | **Built-in web UI** |
| **Config UI** | YAML + admin panel | Web console | Web UI | .env only | YAML only | **Web UI + YAML** |
| **Provider setup** | YAML config | Web console | Web UI | .env variables | YAML endpoints | **Web UI form** |
| **Model discovery** | Manual config | Auto-detect | Auto-detect | /api/tags aggregation | Auto-discovery | **Auto /api/tags union** |
| **Health monitoring** | Prometheus + alerts | Dashboard | Dashboard + alerts | Health endpoints | Health + circuit breakers | **Health + Prometheus** |

---

## 6. Gaps & Opportunities Analysis

### 6.1 Confirmed Gaps (No Competitor Addresses)

| # | Gap | Evidence |
|---|-----|----------|
| 1 | **Ollama-native multi-provider gateway** — expose `/api/chat`, `/api/generate`, `/api/tags` while routing to commercial providers | OllamaProxy (Python, no ChatGPT proxy, no UI) and LiteLLMOllamaAdapter (minimal, OpenRouter-only) are the only other tools with Ollama API. Neither has ChatGPT proxy + stats UI + lightweight Go stack. |
| 2 | **ChatGPT Plus web proxy integrated into multi-provider gateway** | zhaees/chatgpt-proxy proves the concept works (30+ models, cookie auto-refresh, PoW solving) but is single-provider only. No gateway integrates this as a provider type. |
| 3 | **Lightweight, zero-dependency, single-binary gateway with Ollama API** | All Ollama-API tools are Python-based (heavy). Go-based tools (Olla, Bifrost, 1LCB) use OpenAI format. No lightweight Go tool exposes Ollama API. |
| 4 | **Aggregated model listing across local + cloud providers via `/api/tags`** | OllamaProxy has this but without a UI. All OpenAI-format gateways require clients to know model names. |
| 5 | **Local-first + Docker Compose-native + zero external DB** | All full-featured gateways require PostgreSQL or Redis. Only niche tools (Olla, 1LCB) are truly stateless. |

### 6.2 Partially Addressed Gaps

| # | Gap | Partial Solution | What's Missing |
|---|-----|-----------------|----------------|
| 1 | Ollama API translation to commercial providers | OllamaProxy (Python) does this | No ChatGPT proxy, no stats UI, Python-only |
| 2 | Cost/usage tracking | LiteLLM, Portkey, Bifrost have this | All use OpenAI format; none work with Ollama clients |
| 3 | Provider configuration via web UI | LiteLLM, Portkey, Bifrost have admin UIs | None work with Ollama API surface |
| 4 | Lightweight local inference load balancer | Olla (Go, <50MB) does this well | Only for local backends (no cloud provider routing, no ChatGPT proxy) |

### 6.3 OllaRouter Opportunities

| # | Opportunity | Strategic Value | Feasibility |
|---|-------------|----------------|-------------|
| 1 | **First Ollama-native multi-provider gateway** with web UI + stats dashboard | High — uncontested blue ocean | High — Ollama API is simple (3-5 endpoints) |
| 2 | **First integrated ChatGPT Plus provider** in a multi-provider gateway | High — unique differentiator | Medium — requires cookie management, PoW solving |
| 3 | **Lightweight Go binary** with zero external dependencies | High — appeals to homelab/Docker Compose users | High — Go HTTP proxy is well-understood |
| 4 | **Docker Compose drop-in replacement** for ollama service | High — immediate value to ollama-stack users | High — single container, same port convention |
| 5 | **Aggregated model discovery** (`/api/tags` union) across local + cloud | Medium — nice-to-have, differentiator | High — merge local tags + configured provider models |
| 6 | **Cost dashboard** with per-model/provider spend tracking | Medium-High — common pain point | Medium — requires provider-specific cost calculation |

### 6.4 Threats

| Threat | Severity | Mitigation |
|--------|----------|------------|
| Ollama adds native multi-provider support to their runtime | High — cannibalizes core value prop | Monitor Ollama roadmap; differentiate on ChatGPT proxy + cost UI |
| LiteLLM adds Ollama API adapter layer | Medium — reduces uniqueness | LiteLLM is Python+DB-heavy; Go can still win on simplicity |
| Olla adds cloud provider routing | Medium — overlaps on local routing | Olla targets local backends; OllaRouter targets cloud-to-local |
| OllamaProxy matures with Go rewrite + ChatGPT proxy | Low — repo seems inactive | Monitor; first-mover advantage matters |

---

## 7. Candidate USP (Refined)

> **OllaRouter: The Ollama-native LLM gateway for home labs and power users.**
>
> Drop-in replacement for your Ollama endpoint that routes to **any LLM provider** — OpenAI,
> Anthropic, local Ollama, or ChatGPT Plus — with automatic failover, unified model discovery,
> cost tracking, and a configuration dashboard. **Zero external dependencies. One Docker
> Compose service. Single Go binary.**

### Key Differentiators (ranked by defensibility):

| Rank | Differentiator | Competitors Who Do This | OllaRouter Advantage |
|------|---------------|------------------------|---------------------|
| 1 | **Ollama API surface** (not OpenAI format) | OllamaProxy (Python), LLMOllamaAdapter (Python) | Go-based (smaller, faster), plus ChatGPT proxy + UI |
| 2 | **ChatGPT Plus web proxy as provider type** | zhaees/chatgpt-proxy (standalone only) | Integrated into multi-provider routing + cost tracking |
| 3 | **Zero external dependencies** | Olla, 1LCB (no cloud routing) | Full cloud routing without DB |
| 4 | **Docker Compose native + local-first** | Olla (local only), ollama-gateway (Ollama only) | Cloud + local in one config |
| 5 | **Unified `/api/tags` with cost dashboard** | None (OllamaProxy has `/api/tags` without cost tracking) | Cost visibility per model |

---

## 8. Source Evidence

| # | Source | URL | Evidence Type | Key Finding |
|---|--------|-----|---------------|-------------|
| 1 | LiteLLM GitHub | https://github.com/BerriAI/litellm | Repository | 47.7k stars, 100+ providers, OpenAI format, Python, needs DB+Redis |
| 2 | LiteLLM Docs (Getting Started) | https://docs.litellm.ai/docs/ | Documentation | Proxy Server with virtual keys, cost tracking, admin UI, Ollama as provider |
| 3 | LiteLLM Ollama Provider Docs | https://docs.litellm.ai/docs/providers/ollama | Documentation | Ollama integration via `ollama_chat/` prefix, OpenAI format only |
| 4 | Portkey AI Gateway Docs | https://docs.portkey.ai/docs/product/ai-gateway | Documentation | 1600+ models, 45+ providers, fallbacks, load balancing, caching |
| 5 | OpenRouter Model Fallbacks | https://openrouter.ai/docs/guides/routing/model-fallbacks | Documentation | `models` parameter array for automatic failover, SaaS only |
| 6 | Bifrost (Maxim AI) GitHub | https://github.com/maximhq/bifrost | Repository | 5.1k stars, Go, 11µs overhead, 23+ providers including Ollama, OpenAI format |
| 7 | any-llm Gateway Overview | https://mozilla-ai.github.io/any-llm/gateway/overview/ | Documentation | FastAPI, budget management, virtual keys, usage analytics, needs PostgreSQL |
| 8 | Ollama Proxy Docs | https://ollama-proxy.readthedocs.io/en/latest/ | Documentation | Python, **Ollama-native API**, multi-provider, routing strategies, fallback |
| 9 | Ollama Proxy Multi-Provider | https://ollama-proxy.readthedocs.io/en/latest/multi-provider/ | Documentation | Extensive multi-provider config: OpenAI, Anthropic, Google, Azure, Bedrock, Ollama |
| 10 | Olla GitHub | https://github.com/thushan/olla | Repository | 224 stars, Go, <50MB, local inference load balancer, no cloud routing |
| 11 | Olla Blog Post | https://www.thushanfernando.com/2026/05/olla-a-smart-load-balancer-for-your-llm-infrastructure/ | Article | Detailed architecture: dual engine, streaming-first, sticky sessions |
| 12 | 1LCB/ollama-gateway GitHub | https://github.com/1LCB/ollama-gateway | Repository | 16 stars, Go, Ollama-only, reverse proxy with rate limiting, Prometheus metrics |
| 13 | LiteLLMOllamaAdapter GitHub | https://github.com/pure-san/LiteLLMOllamaAdapter | Repository | 1 star, Python/Flask, **Ollama-native API**, forwards to OpenRouter via LiteLLM |
| 14 | zhaees/chatgpt-proxy GitHub | https://github.com/zhaees/chatgpt-proxy | Repository | 5 stars, TypeScript+Python, cookie-based ChatGPT Plus proxy, 30+ models |
| 15 | ollama-stack docker-compose.yml | C:\Projects\ollama-stack\docker-compose.yml | Local file | Existing stack: ollama (11434) + open-webui (3000), GPU passthrough |

---

## 9. Detailed Requirements Gap Analysis

### Requirement: Ollama API Mirroring

**What it means:** Expose `POST /api/chat`, `POST /api/generate`, `GET /api/tags` (and optionally `/api/show`, `/api/version`, `/api/embed`) so existing Ollama clients (Open WebUI, LangChain Ollama, `ollama` CLI, continue.dev, etc.) can use the gateway without code changes.

| Tool | Status | Details |
|------|--------|---------|
| LiteLLM | ❌ | OpenAI `/v1/chat/completions` only. No `/api/tags` or Ollama endpoints. |
| Portkey | ❌ | OpenAI format only. Custom host routing but no API translation. |
| OpenRouter | ❌ | OpenAI-compatible API only. |
| Bifrost | ❌ | OpenAI format with provider prefix routing. |
| any-llm | ❌ | OpenAI format via `provider:model` naming. |
| **OllamaProxy** | **✅** | **Full Ollama API: `/api/chat`, `/api/generate`, `/api/tags`, plus `/api/providers`, `/health`** |
| Olla | ❌ | OpenAI `/olla/proxy/v1` and Anthropic `/olla/anthropic/v1` formats. |
| 1LCB/ollama-gateway | ✅ | Pure Ollama passthrough — proxies to Ollama servers only. |
| LLMOllamaAdapter | ✅ | Minimal Ollama API: `/api/chat`, `/api/generate`, `/api/tags`, `/api/show`, `/api/version`. |
| chatgpt-proxy | ❌ | OpenAI format only. |
| **OllaRouter (planned)** | **✅** | **Must implement: POST /api/chat, POST /api/generate, GET /api/tags with NDJSON streaming.** |

### Requirement: Provider Routing

| Tool | Providers | Selection Strategy |
|------|-----------|-------------------|
| LiteLLM | 100+ | Model prefix + YAML routing rules |
| Portkey | 45+ (1600+ models) | Config-based conditional routing |
| OpenRouter | 60+ (400+ models) | Model-based auto-routing + provider selection |
| Bifrost | 23+ | Provider prefix in model name |
| OllamaProxy | 7 (OpenAI, Anthropic, Gemini, OpenRouter, Azure, Bedrock, Ollama) | Model-based, capability-based, cost-optimized, round-robin, least-loaded, fastest-response |
| Olla | Local only (Ollama, vLLM, LM Studio, etc.) | Priority + least-connections |
| **OllaRouter (planned)** | **5-10 initially** | **Model-based routing with YAML provider config** |

### Requirement: Fallback

| Tool | Strategy |
|------|----------|
| LiteLLM | Router with retry/fallback across deployments |
| Portkey | Fallback configs with automatic retries |
| OpenRouter | `models` parameter array — tries in priority order |
| Bifrost | Auto-failover with zero-config |
| OllamaProxy | next_available, retry_same, best_alternative, circuit breakers |
| Olla | Circuit breakers + automatic failover between endpoints |
| **OllaRouter (planned)** | **Per-model fallback chain (YAML config), circuit breaker pattern** |

### Requirement: Provider/Model Listing

| Tool | Method | UI? |
|------|--------|-----|
| LiteLLM | Admin panel + API | ✅ |
| Portkey | Dashboard | ✅ |
| OpenRouter | Web catalog | ✅ |
| Bifrost | Web UI | ✅ |
| OllamaProxy | `GET /api/tags` (all providers) + `GET /api/providers` | ❌ (CLI only) |
| Olla | `GET /olla/proxy/v1/models` (unified) | ❌ (CLI only) |
| **OllaRouter (planned)** | **`GET /api/tags` (unified) + provider config UI** | **✅** |

### Requirement: Local Ollama Passthrough

| Tool | Implementation |
|------|---------------|
| LiteLLM | `ollama_chat/` prefix → sends to `api_base: http://localhost:11434` |
| Portkey | Custom host URL routing |
| Bifrost | Ollama listed as supported provider |
| OllamaProxy | `OLLAMA_ENABLED=true`, `OLLAMA_BASE_URL=http://localhost:11434` — native passthrough |
| Olla | Native endpoint type `type: ollama`, health check + model discovery |
| 1LCB/ollama-gateway | Native — routes entirely to Ollama servers |
| **OllaRouter (planned)** | **Native passthrough — local Ollama as first-class provider with auto-detection** |

### Requirement: ChatGPT Plus Web Proxy Mode

| Tool | Implementation |
|------|---------------|
| zhaees/chatgpt-proxy | Cookie-based, PoW solver, 30+ models, OpenAI-compatible API |
| **OllaRouter (planned)** | **Integrate as a provider type — `chatgpt-plus/` model prefix, cookie config in provider YAML** |

### Requirement: Stats UI

| Tool | Dashboard Features |
|------|-------------------|
| LiteLLM | Admin UI: spend tracking per key/team/user, guardrails, monitoring |
| Portkey | Cloud dashboard: real-time logs, observability, cost tracking |
| OpenRouter | Dashboard: model catalog, billing, usage history |
| Bifrost | Web UI: real-time monitoring, analytics, configuration |
| any-llm | Usage analytics: token counts, per-request costs, user attribution |
| **OllaRouter (planned)** | **Built-in web dashboard: per-provider token usage, cost, latency, request history** |

### Requirement: Docker Deployment

| Tool | Docker | docker-compose | External deps |
|------|--------|----------------|---------------|
| LiteLLM | ✅ | ✅ | PostgreSQL + Redis |
| Portkey | ✅ | ✅ | None |
| Bifrost | ✅ | ✅ | None (optional DB configstore) |
| any-llm | ✅ | ✅ | PostgreSQL |
| OllamaProxy | ✅ | ✅ | None |
| Olla | ✅ | ✅ | None |
| 1LCB/ollama-gateway | ✅ | ❌ | None |
| **OllaRouter (planned)** | **✅** | **✅ (drop-in for ollama)** | **None** |

---

## 10. Competitive Landscape Summary

```
                    Cloud Provider Routing
                    +                       -
                    |                        
   Full-featured    |  LiteLLM, Portkey     |  
   Gateways         |  Bifrost, any-llm     |  
                    |                        |
                    +------------------------+--------
                    |                        |
   Ollama-native    |  OllamaProxy ✓         |  1LCB/ollama-gateway
   Gateways         |  LLMOllamaAdapter ✓    |  (pure Ollama routing)
                    |  **OllaRouter (goal)**  |  
                    |                        |
                    +------------------------+--------
                    |                        |
   Local-only       |  (no cloud routing)    |  Olla (local backends)
   Tools            |                        |  
                    |                        |
                    +                        +
                    Local (Ollama)
```

**Where OllaRouter fits:** The top-right quadrant — **Ollama-native API with cloud provider routing** — is served only by OllamaProxy (Python, inactive, no UI, no ChatGPT proxy) and the minimal LLMOllamaAdapter. This is a **virtually uncontested space**.

---

## 11. Key Strategic Recommendations

1. **Go language** — matches Olla and Bifrost for performance; differentiates from Python-based competitors (LiteLLM, OllamaProxy, any-llm)
2. **Ollama API first** — the core differentiator; implement `/api/chat`, `/api/generate`, `/api/tags` with NDJSON streaming
3. **ChatGPT Plus provider type** — second differentiator; integrate cookie-based proxy as a first-class provider
4. **Built-in web UI** — configuration + cost dashboard; no competitor with Ollama API has this
5. **Zero external DB** — stateless design; config file + optional Prometheus for persistence
6. **Docker Compose drop-in** — match existing ollama-stack patterns (port 11434 or adjacent)
7. **Free + MIT license** — consistent with community expectations

---

*End of Phase 2 Competitor Analysis. Ready for Phase 3 USP Definition.*
