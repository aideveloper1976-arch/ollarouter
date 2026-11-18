# Phase 1 — Market Research: OllaRouter

**Date:** 2026-05-20  
**Agent:** market-researcher  
**Project:** ollama-stack (C:\Projects\ollama-stack)  
**Target:** OllaRouter — Lightweight multi-provider LLM gateway with Ollama-compatible endpoints

---

## 1. Research Summary

OllaRouter is positioned as a **lightweight, Docker-native, multi-provider LLM gateway** that exposes an Ollama-compatible API surface (`POST /api/chat`, `POST /api/generate`, `GET /api/tags`) while routing to commercial providers (OpenAI, Anthropic, Google, etc.), local Ollama instances, and optionally a ChatGPT Plus web proxy mode.

The LLM gateway market is maturing rapidly. Incumbents like **LiteLLM** (47.7k GitHub stars), **Portkey** (11.8k stars), **OpenRouter** (managed SaaS), and **Bifrost/Maxim** dominate. However, **no existing solution exposes an Ollama-native API** — all use the OpenAI `/v1/chat/completions` format. This is OllaRouter's primary differentiator: native compatibility with Ollama clients (Open WebUI, LangChain Ollama integration, ollama CLI, etc.) while adding provider routing, cost tracking, and fallback.

### Market Size & Trends
- Enterprise LLM spending surpassed $8.4B in 2025 (Maxim AI)
- 90%+ of production AI teams run 5+ LLMs simultaneously
- 80% of AI costs are invisible at billing — cost tracking is a critical unmet need
- OpenRouter processes 7.9+ trillion tokens monthly across 400+ models
- Ollama has 172k+ GitHub stars and is the dominant local LLM runtime

---

## 2. Target Users

| Persona | Need | Use Case |
|---------|------|----------|
| **Local-first AI developer** | Wants to use Open WebUI or LangChain with local + cloud fallback | Configure OllaRouter as Ollama endpoint, auto-fallback to GPT-4o when local model is overloaded |
| **Home lab / homelab operator** | Runs ollama-stack Docker Compose, wants one config for local + cloud models | Drop-in replacement for the existing `ollama` service in docker-compose.yml |
| **Cost-conscious team** | Needs token/cost tracking across providers, budget visibility | Deploy OllaRouter with stats dashboard, see per-model spend |
| **ChatGPT Plus subscriber** | Wants to use GPT-5 models via API without separate OpenAI API billing | Configure ChatGPT Plus cookie proxy in OllaRouter provider config |
| **Platform engineer** | Needs failover, 429 handling, provider config UI for non-technical team | Configure YAML + UI providers, set fallback chains per model |

---

## 3. Competitors Reviewed

### 3.1 LiteLLM (BerriAI)
- **Stars:** 47.7k | **License:** MIT | **Language:** Python
- **Format:** OpenAI `/v1/chat/completions` (NOT Ollama-compatible)
- **Deployment:** Docker, K8s, Helm, Terraform
- **Features:** 100+ providers, virtual keys, spend tracking, load balancing, guardrails, caching, Prometheus metrics
- **Pricing:** Open-source (self-hosted) + managed enterprise tier
- **Docker:** Production requires PostgreSQL, Redis for high traffic. `litellm/config.yaml` for provider config
- **Gap for OllaRouter:** No Ollama-native API surface. Heavy Python dependency. Complex production setup (DB+Redis required). No ChatGPT web proxy support

### 3.2 Portkey AI Gateway
- **Stars:** 11.8k | **License:** MIT | **Language:** TypeScript/Node
- **Format:** OpenAI-compatible + custom Portkey SDK
- **Deployment:** Docker, Cloudflare Workers, Node.js
- **Features:** 1600+ models, 45+ providers, guardrails, retries (5x exp backoff), load balancing, caching
- **Pricing:** Open-source + Hosted ($49/mo Production) + Enterprise
- **Gap for OllaRouter:** No Ollama API compatibility. Complex config. No ChatGPT proxy. Requires Node.js

### 3.3 OpenRouter
- **Type:** Managed SaaS (not self-hostable)
- **Format:** OpenAI-compatible API
- **Features:** 400+ models, 60+ providers, auto-routing, fallback, credit-based billing, 5.5% platform fee on PAYG
- **Pricing:** Free tier (50 req/day), PAYG ($10+ credits gets unlimited), Enterprise
- **Gap for OllaRouter:** Fully managed — no self-hosted option, no local Ollama integration. Cannot be added to docker-compose.yml

### 3.4 Bifrost by Maxim AI
- **Language:** Go | **License:** Open-source + Enterprise
- **Latency:** 11µs overhead at 5K RPS (best in class)
- **Features:** Adaptive load balancing, semantic caching, OTel, Prometheus, 15+ providers
- **Format:** OpenAI `/v1/chat/completions`
- **Gap for OllaRouter:** No Ollama API surface. Enterprise-focused. No ChatGPT web proxy

### 3.5 any-llm-gateway (Mozilla AI)
- **Format:** OpenAI-compatible
- **Deployment:** Docker + PostgreSQL
- **Features:** Provider routing, token tracking, budget management
- **Gap for OllaRouter:** No Ollama API. Requires PostgreSQL. No ChatGPT proxy

### 3.6 ChatGPT Proxy (zhaees/chatgpt-proxy)
- **Stars:** 5 | **Language:** TypeScript + Python
- **Format:** OpenAI `/v1/chat/completions`
- **Features:** Cookie-based ChatGPT Plus/Pro access, tool call emulation, auto cookie refresh, PoW solver
- **Gap for OllaRouter:** Single-provider (ChatGPT only). No local Ollama routing. No multi-provider support or cost tracking

---

## 4. Feature Patterns

### Core features present in ALL major gateways:
| Feature | LiteLLM | Portkey | OpenRouter | Bifrost | any-llm | OllaRouter (planned) |
|---------|---------|---------|------------|---------|---------|---------------------|
| Multi-provider routing | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| OpenAI-compatible API | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ (Ollama API) |
| Ollama-compatible API | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ **Unique** |
| Streaming (SSE/NDJSON) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Fallback/failover | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ |
| 429/5xx retry handling | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ |
| Token/cost tracking | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Provider config UI | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ |
| Rate limiting | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Docker Compose | ✅ | ✅ | N/A | ✅ | ✅ | ✅ |
| ChatGPT web proxy | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ **Unique** |
| Local Ollama routing | ✅ | ✅ | ❌ | ✅ | ❌ | ✅ (native) |
| No external DB required | ❌ | ✅ | N/A | ✅ | ❌ | ✅ **Key diff** |

### Unique OllaRouter Opportunities:
1. **Ollama API compatibility** — no existing gateway exposes `/api/chat`, `/api/generate`, `/api/tags`
2. **ChatGPT Plus cookie proxy** — no gateway integrates ChatGPT Plus account access
3. **Lightweight + zero external deps** — most gateways require PostgreSQL/Redis
4. **Local-first architecture** — designed for homelab/docker-compose environments

---

## 5. Pricing Notes

| Gateway | Base | Transaction Fee | Self-Host Cost |
|---------|------|-----------------|----------------|
| **LiteLLM** | Open-source | None (proxy only) | Server + optional DB |
| **Portkey** | Open-source | None (proxy only) | Server |
| **OpenRouter** | Free tier | 5.5% PAYG / Bulk discounts (Enterprise) | N/A (managed) |
| **Bifrost** | Open-source | Enterprise licensing | Server |
| **OllaRouter (planned)** | Open-source | None | Minimal (container only) |

No gateway charges for the self-hosted proxy layer itself. OpenRouter is the only one with a platform markup (5.5%). OllaRouter will be **free, open-source, zero-markup**.

---

## 6. UX / Flow Patterns

### Common Gateway Configuration Flow:
1. **Deploy** — `docker-compose up` with a config file (YAML/JSON)
2. **Configure providers** — add API keys, endpoints, model lists in config or web UI
3. **Route requests** — client points base URL to gateway, uses model name
4. **Monitor** — view usage logs, cost, latency in dashboard

### Key UX Observations from Incumbents:
- **LiteLLM**: YAML config is powerful but complex; admin UI is functional but not beautiful
- **Portkey**: Excellent web console with real-time logs; `npx @portkey-ai/gateway` quickstart is elegant
- **OpenRouter**: Dashboard is clean, model catalog is searchable, billing is transparent
- **Bifrost**: Docker `docker run` in 30 seconds; OpenTelemetry integration is seamless

### OllaRouter UX Opportunities:
- **Open WebUI integration** — users configure OllaRouter as their Ollama host URL; everything "just works"
- **Provider config UI** — web form to add/manage providers (no YAML editing required)
- **Model listing** — `GET /api/tags` returns union of local + configured cloud models
- **Stats dashboard** — per-provider token usage, cost, latency charts
- **Health endpoints** — `/health`, `/metrics` for Docker health checks

---

## 7. Streaming Concerns (NDJSON vs SSE)

### Key Finding:
Ollama uses **NDJSON** (`application/x-ndjson`) — each response is one JSON object per line with no `data:` prefix. Commercial providers (OpenAI, Anthropic) use **SSE** (`text/event-stream`) where each event is prefixed with `data: ` and terminated by `\n\n`.

### Implications for OllaRouter:
1. **Inbound (client-facing):** Must serve Ollama NDJSON format exactly — clients expect `{"model":"...", "response":"...", "done":false}` lines
2. **Outbound (to providers):**
   - To OpenAI/Anthropic: Receive SSE, parse `data: {...}` lines, emit NDJSON lines
   - To local Ollama: Pass-through (already NDJSON)
   - To ChatGPT proxy: Receives SSE from proxy, emit NDJSON
3. **Stream=false passthrough:** Single JSON response for non-streaming requests — simpler to normalize
4. **Content-Type negotiation:** Must set `Content-Type: application/x-ndjson` for streaming, `application/json` for non-streaming
5. **Error wrapping:** Provider errors (4xx/5xx) must be wrapped in Ollama-compatible error format

### NDJSON vs SSE Tradeoff:
- **NDJSON** is simpler (no event parsing), preferred by Ollama ecosystem
- **SSE** is standard for web UIs, supports multiple event types
- **OllaRouter should:** serve NDJSON inbound (Ollama-compatible), translate from SSE/JSON outbound

---

## 8. Docker Deployment Expectations

### Existing ollama-stack Setup:
```
docker-compose.yml:
- ollama: (port 11434, GPU, volume mounts)
- open-webui: (port 3000, depends on ollama)
```

### Expected OllaRouter Integration:
```yaml
services:
  ollama:
    # unchanged — local inference engine
  ollarouter:
    image: ollarouter:latest          # new service
    container_name: ollarouter
    ports:
      - "11435:11435"                 # or same port as ollama with proxy
    environment:
      - OLLAROUTER_PORT=11435
      - OLLAROUTER_OLLAMA_HOST=http://ollama:11434
      - OLLAROUTER_PROVIDERS_CONFIG=/app/config/providers.yaml
    volumes:
      - ./ollarouter/providers.yaml:/app/config/providers.yaml
    depends_on:
      - ollama
    restart: unless-stopped
  open-webui:
    environment:
      - OLLAMA_BASE_URL=http://ollarouter:11435  # route through gateway
```

### Production Docker Patterns (from research):
| Pattern | Recommendation |
|---------|---------------|
| Health checks | `test: ["CMD", "curl", "-f", "http://localhost:11435/health"]` with interval/retries |
| Resource limits | `deploy.resources.limits: cpus: '0.5', memory: 512M` (lightweight) |
| Logging | JSON-file driver with rotation (max-size: 10m, max-file: 3) |
| Config persistence | Read-only bind mount for provider YAML |
| Secrets | API keys via env vars or .env file (not in config file) |
| Network isolation | Internal network for ollama→ollarouter, exposed port for clients |
| GPU pass-through | Not needed (OllaRouter is CPU-only proxy; Ollama handles GPU) |

### Runtime Profile (Estimated):
- **Base image:** Minimal (Alpine Linux or distroless, ~10-20MB)
- **Runtime:** Node.js/Go/Python (TBD) — prefer Go for minimal footprint (~5MB binary)
- **Memory:** 50-200MB RSS typical
- **CPU:** <0.1 core idle, spikes during request forwarding
- **Storage:** None (stateless; config mounted externally)

---

## 9. Opportunities / Gaps

### Confirmed Gaps in Current Market:
1. **No Ollama-compatible multi-provider gateway exists** — all use OpenAI format
2. **No ChatGPT Plus proxy integrated into multi-provider gateway** — separate project (zhaees/chatgpt-proxy) exists but is single-provider
3. **No lightweight, zero-dependency gateway** — LiteLLM requires PostgreSQL+Redis for production, Portkey requires Node.js
4. **No local-first gateway designed for homelab/docker-compose** — all incumbent gateways target cloud/enterprise

### OllaRouter Opportunities:
| # | Opportunity | Evidence |
|---|-------------|----------|
| 1 | **Ollama API-first** — serve clients that speak Ollama protocol (Open WebUI, LangChain Ollama, ollama CLI) | Ollama API docs show `/api/chat`, `/api/generate`, `/api/tags` — no gateway implements these |
| 2 | **ChatGPT Plus bridge** — integrate chatgpt-proxy pattern as a provider type | zhaees/chatgpt-proxy proves this works; 30+ ChatGPT models available |
| 3 | **Stateless + lightweight** — zero external DB, single binary/container | All competitors require external persistence |
| 4 | **Docker Compose native** — designed as drop-in replacement for `ollama` service in docker-compose.yml | ollama-stack already has docker-compose.yml pattern established |
| 5 | **Cost + token dashboard** — per-request token tracking without external services | Common pain point from research: "80% of AI costs invisible" |
| 6 | **Aggregated model listing** — `/api/tags` returns union of Ollama models + configured cloud models | No existing gateway provides aggregated model list across local + cloud |

---

## 10. Candidate USP

> **OllaRouter: The Ollama-native multi-provider LLM gateway.**
>
> Drop-in Ollama API compatibility for routing to any LLM provider — local Ollama, OpenAI,
> Anthropic, or ChatGPT Plus — with automatic failover, cost tracking, and a configuration UI.
> No external database required. One Docker Compose service added.

### Key Differentiators (sorted by uniqueness):
1. **Ollama API surface** (not OpenAI format) — first mover advantage
2. **ChatGPT Plus web proxy** — unique provider type
3. **Zero external dependencies** — no PostgreSQL, no Redis
4. **Local-first + docker-compose-native** — designed for the homelab/Ollama ecosystem
5. **Aggregated model discovery** — one `/api/tags` to see all available models

---

## 11. Source Evidence

| # | Source | URL | Evidence Type |
|---|--------|-----|---------------|
| 1 | Ollama API Reference (GitHub) | https://github.com/ollama/ollama/blob/main/docs/api.md | API specification for `/api/generate`, `/api/chat`, `/api/tags` |
| 2 | Ollama Streaming Docs | https://docs.ollama.com/api/streaming | NDJSON format (`application/x-ndjson`), stream=false option |
| 3 | Ollama Tags Endpoint | https://docs.ollama.com/api/tags | Response format for listing models |
| 4 | LiteLLM GitHub | https://github.com/BerriAI/litellm | Incumbent leader — 47.7k stars, 100+ providers, OpenAI format |
| 5 | LiteLLM Deployment Docs | https://docs.litellm.ai/docs/proxy/deploy | Docker/K8s deployment patterns, requires Postgres+Redis for prod |
| 6 | Portkey AI Gateway | https://github.com/Portkey-AI/gateway | 11.8k stars, 1600+ models, 45+ providers, TypeScript |
| 7 | OpenRouter Pricing | https://openrouter.ai/pricing | 400+ models, 60+ providers, 5.5% fee, managed SaaS (no self-host) |
| 8 | LiteLLM vs OpenRouter Comparison | https://www.truefoundry.com/blog/litellm-vs-openrouter | Feature matrix: provider support, rate limiting, load balancing, observability |
| 9 | ChatGPT Proxy (zhaees) | https://github.com/zhaees/chatgpt-proxy | Cookie-based ChatGPT Plus API proxy, 30+ models, PoW solver |
| 10 | any-llm Docker Deployment Guide | https://mozilla-ai.github.io/any-llm/gateway/docker-deployment/ | Production Docker patterns: health checks, Nginx, resource limits |
| 11 | Top 5 LLM Gateways 2025 | https://www.getmaxim.ai/articles/top-5-llm-gateways-in-2025 | Market overview: Bifrost, LiteLLM, OpenRouter, TensorZero, Cloudflare |
| 12 | ollama-stack docker-compose.yml | C:\Projects\ollama-stack\docker-compose.yml | Existing stack: ollama (11434) + open-webui (3000) |

### Local Evidence Files:
- `docs\product\evidence\ollama_api_specification_*.md` — Ollama API specification details
- `docs\product\evidence\ollama_streaming_protocol_*.md` — NDJSON streaming protocol
- `docs\product\evidence\llm_gateway_competitor_landscape_*.md` — LiteLLM, Portkey, Bifrost details
- `docs\product\evidence\chatgpt_plus_web_proxy_mode_*.md` — ChatGPT Plus cookie proxy details
- `docs\product\evidence\docker_deployment_for_ai_gateways_*.md` — Production Docker patterns

---

*End of Phase 1 Market Research. Ready for Phase 2 Competitor Analysis / Phase 3 USP Definition.*
