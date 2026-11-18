# Phase 3 — USP Definition: OllaRouter

**Date:** 2026-05-20  
**Agent:** product-analyst  
**Project:** ollama-stack (C:\Projects\ollama-stack)  
**Phase 1 Reference:** `docs/product/traceability/checkpoints/01-market-research.md`  
**Phase 2 Reference:** `docs/product/traceability/checkpoints/02-competitor-analysis.md`  
**Target:** OllaRouter — Lightweight multi-provider LLM gateway with Ollama-compatible endpoints

---

## Table of Contents

1. [Product Vision](#1-product-vision)
2. [Target Users & Personas](#2-target-users--personas)
3. [User Problems & Pain Points](#3-user-problems--pain-points)
4. [Table-Stakes Features](#4-table-stakes-features)
5. [Differentiators](#5-differentiators)
6. [Unique Selling Proposition (USP)](#6-unique-selling-proposition-usp)
7. [Functional Requirements](#7-functional-requirements)
8. [Non-Functional Requirements](#8-non-functional-requirements)
9. [Risks, Assumptions, Dependencies & Open Questions](#9-risks-assumptions-dependencies--open-questions)
10. [UX Design Phase Plan](#10-ux-design-phase-plan)
11. [UX Goals & Key User Flows](#11-ux-goals--key-user-flows)
12. [UX Acceptance Criteria](#12-ux-acceptance-criteria)
13. [Recommended EPICs](#13-recommended-epics)
14. [Traceability Links](#14-traceability-links)
15. [Evidence Sources](#15-evidence-sources)

---

## 1. Product Vision

> **OllaRouter is the Ollama-native LLM gateway for home labs, developers, and cost-conscious teams.**
>
> Drop-in replacement for your Ollama endpoint that routes to **any LLM provider** — OpenAI, Anthropic, Google, local Ollama, or ChatGPT Plus — with automatic failover, unified model discovery, per-request cost tracking, and a configuration dashboard. **Zero external dependencies. One Docker Compose service. Single Go binary.**

### Vision Statement (Elevator Pitch)

> "Configure OllaRouter once in your docker-compose.yml, point Open WebUI at it, and instantly get access to GPT-4o, Claude Sonnet, local Mistral, and ChatGPT Plus — all through the same Ollama API your tools already speak. No code changes. No separate SDKs. No database to manage."

### Strategic Goals

| Goal | Measure |
|------|---------|
| First Ollama-native multi-provider gateway shipped | MVP with 5+ providers, `/api/chat`, `/api/generate`, `/api/tags` |
| Sub-50MB container, zero external DB | Docker image size, no PostgreSQL/Redis dependency |
| Unified model discovery via `/api/tags` | Union of local + configured cloud models returned |
| Cost visibility without external services | Built-in dashboard with per-model/provider spend |
| Open-source community adoption | MIT license, GitHub stars, Docker pulls |

---

## 2. Target Users & Personas

### Persona 1: Local-First AI Developer

| Attribute | Detail |
|-----------|--------|
| **Name** | Alex (Developer) |
| **Role** | AI/ML engineer, full-stack developer, hobbyist |
| **Tools** | Open WebUI, LangChain, continue.dev, ollama CLI |
| **Current Pain** | Must juggle multiple API endpoints: one for local Ollama, separate keys/endpoints for OpenAI, Anthropic. No unified model listing. No automatic fallback when local models are loaded. |
| **Goal** | Configure one Ollama-compatible endpoint, use any model from any provider, auto-fallback when local is busy. |
| **Quote** | "I want to use Open WebUI with local models for quick chats and GPT-4 for complex reasoning — without switching endpoints." |

### Persona 2: Homelab Operator

| Attribute | Detail |
|-----------|--------|
| **Name** | Priya (SysAdmin) |
| **Role** | DevOps engineer, self-hoster, docker-compose power user |
| **Tools** | Docker, docker-compose, Portainer, Linux, home server |
| **Current Pain** | Existing ollama-stack is simple but limited to Ollama models. Adding cloud models requires separate services, proxy configs, API key management. LiteLLM is too heavy (needs PostgreSQL+Redis) for a homelab. |
| **Goal** | One-line addition to docker-compose.yml that unlocks cloud models alongside local ones, with minimal resource overhead. |
| **Quote** | "I run ollama-stack on my Proxmox server. I don't want to spin up PostgreSQL just for an API gateway." |

### Persona 3: Cost-Conscious Team Lead

| Attribute | Detail |
|-----------|--------|
| **Name** | Jordan (Tech Lead) |
| **Role** | Engineering manager, startup CTO, freelancer |
| **Tools** | Team using various LLM APIs, need usage visibility |
| **Current Pain** | 80% of AI costs are invisible at billing (research finding). No per-model, per-user, per-project cost tracking. Budget overruns are discovered after the fact. |
| **Goal** | Track token usage and costs per provider/model without adding a third-party observability service. Get a dashboard that shows where the budget is going. |
| **Quote** | "I need to know which team member is spending $500/mo on GPT-4 vs Claude — and cap it when necessary." |

### Persona 4: ChatGPT Plus Subscriber

| Attribute | Detail |
|-----------|--------|
| **Name** | Sam (Power User) |
| **Role** | Researcher, writer, AI enthusiast |
| **Tools** | ChatGPT Plus subscription ($20/mo), Open WebUI, various AI tools |
| **Current Pain** | Paying $20/mo for ChatGPT Plus with GPT-5 access, but cannot use it programmatically. OpenAI API is separate billing with per-token costs. Wants to use the Plus subscription via API without additional API fees. |
| **Goal** | Configure OllaRouter with ChatGPT Plus cookie, access all Plus models through the Ollama API — zero additional API cost. |
| **Quote** | "I already pay for ChatGPT Plus. I should be able to use those models in my own tools." |

### Persona 5: Platform Engineer

| Attribute | Detail |
|-----------|--------|
| **Name** | Morgan (Platform Engineer) |
| **Role** | Infrastructure engineer, SRE, enterprise architect |
| **Tools** | K8s, Terraform, Helm, Prometheus, Grafana |
| **Current Pain** | Need to provide LLM access to non-technical teams (marketing, support) without exposing raw API keys. Need rate limiting, usage quotas, and monitoring. Current solutions are too complex or don't integrate with existing monitoring. |
| **Goal** | Deploy a lightweight gateway with API key auth, usage limits, Prometheus metrics, and a UI for non-technical users to manage providers. |
| **Quote** | "I need to give my marketing team access to GPT-4o without giving them my OpenAI API key — and I need to see what they're spending." |

---

## 3. User Problems & Pain Points

| # | Problem | Affected Personas | Evidence Source |
|---|---------|-------------------|-----------------|
| P1 | **No unified endpoint** — users must manage separate URLs, keys, and SDKs for each LLM provider | Alex, Priya, Morgan | Market research §2, Competitor analysis §6.1 (Gap 1) |
| P2 | **No Ollama-native multi-provider gateway exists** — all existing gateways expose OpenAI format, breaking Ollama clients | Alex, Priya | Market research §4 (Feature matrix shows 0/5 competitors with Ollama API) |
| P3 | **Heavy gateways require databases** — LiteLLM needs PostgreSQL+Redis, adding operational complexity for homelabs | Priya, Morgan | Market research §8; Competitor analysis §4 (Architectural comparison) |
| P4 | **No ChatGPT Plus API access** — subscribers pay $20/mo but cannot use those models programmatically | Sam | Market research §3.6; Competitor analysis §1.10 |
| P5 | **Invisible AI costs** — no per-model, per-provider spend tracking without external services | Jordan | Market research §1 (80% of AI costs invisible); Competitor analysis §6.1 (Gap 5) |
| P6 | **No unified model discovery** — users must know exact model names per provider, no `/api/tags` equivalent across cloud providers | Alex, Priya | Market research §9 (Opportunity 6) |
| P7 | **Complex provider configuration** — YAML-only config requires technical expertise, no web UI for non-technical users | Morgan | Competitor analysis §5 (Only LiteLLM/Portkey/Bifrost have web UIs, none with Ollama API) |
| P8 | **No automatic failover** — when a provider is down or rate-limited, requests fail instead of falling back | Alex, Morgan | Market research §4 (All major gateways have fallback; OllaRouter should match) |

---

## 4. Table-Stakes Features

These features are present in ALL major LLM gateways (LiteLLM, Portkey, OpenRouter, Bifrost) and are **minimum requirements** for OllaRouter to be credible:

| # | Feature | Rationale | Competitor Baseline |
|---|---------|-----------|-------------------|
| T1 | **Multi-provider routing** — route requests to OpenAI, Anthropic, Google, local Ollama | Every competitor does this | 100% of Tier 1 gateways |
| T2 | **Streaming support** — SSE/NDJSON streaming responses | Required for real-time chat UX | 100% of Tier 1 gateways |
| T3 | **Fallback/failover** — auto-retry on 429/5xx, fallback chain | Required for production reliability | 80% of Tier 1 gateways |
| T4 | **API key authentication** — protect the gateway with API keys | Required for production use | 100% of Tier 1 gateways |
| T5 | **Rate limiting** — per-key/second/minute request limits | Required for cost control | 100% of Tier 1 gateways |
| T6 | **Docker deployment** — containerized with docker-compose support | All self-hostable competitors offer this | 100% of self-hosted gateways |
| T7 | **Health check endpoint** — `/health` for Docker health checks and monitoring | Docker best practice | 100% of self-hosted gateways |
| T8 | **Configuration via YAML** — providers, models, routing rules in config file | Common pattern across all competitors | 100% of self-hosted gateways |
| T9 | **Environment variable config** — API keys, ports, hostnames via env | Docker/12-factor app standard | 100% of self-hosted gateways |
| T10 | **Request logging** — log requests, responses, errors for debugging | Required for operational visibility | 100% of Tier 1 gateways |

---

## 5. Differentiators

Features that set OllaRouter apart from every competitor, ranked by defensibility:

| Rank | Differentiator | OllaRouter Advantage | Defensibility | Competitors |
|------|---------------|---------------------|---------------|-------------|
| **1** | **Ollama API surface** — expose `POST /api/chat`, `POST /api/generate`, `GET /api/tags` (not OpenAI `/v1/chat/completions`) | Zero-config integration with Open WebUI, LangChain Ollama, ollama CLI, continue.dev | **High** — OllamaProxy (Python, inactive) and LLMOllamaAdapter (minimal, 1 star) are only others. OllaRouter will be Go-based, lighter, with UI. | OllamaProxy (Python, no UI), LLMOllamaAdapter (minimal) |
| **2** | **ChatGPT Plus web proxy** as first-class provider type | Use $20/mo ChatGPT Plus subscription via API — 30+ models including GPT-5 family, no per-token API costs | **High** — No multi-provider gateway integrates this. Proved possible by zhaees/chatgpt-proxy (standalone, single-provider). | zhaees/chatgpt-proxy (standalone only) |
| **3** | **Zero external dependencies** — no PostgreSQL, no Redis, no external DB | Single container, stateless, instant startup. Config file + optional Prometheus only. | **Medium-High** — Olla and 1LCB have this but no cloud routing. Portkey and Bifrost are standalone but OpenAI format. | LiteLLM (PG+Redis), any-llm (PG), Portkey (standalone but OpenAI), Bifrost (standalone but OpenAI) |
| **4** | **Built-in stats dashboard** — per-provider token usage, cost, latency, request history | No competitor with Ollama API has a web UI. LiteLLM/Portkey have dashboards but serve OpenAI format. | **Medium** — LiteLLM and Portkey have dashboards. OllaRouter's dashboard targets Ollama-native metrics. | None (among Ollama-API tools), LiteLLM/Portkey (OpenAI format only) |
| **5** | **Provider configuration web UI** — add/manage providers via browser form | Non-technical users can add providers without editing YAML. | **Medium** — LiteLLM and Portkey have admin UIs, but for OpenAI format. OllaRouter's is tailored for Ollama-native config. | LiteLLM (admin panel), Portkey (console), Bifrost (web UI) — all OpenAI format |
| **6** | **Unified model listing via `/api/tags`** — returns union of local Ollama models + configured cloud provider models | One endpoint to discover ALL available models. No competitor does aggregated cloud+local model listing with Ollama API. | **Medium** — OllamaProxy has `/api/tags` but no cost tracking or UI. | OllamaProxy (no UI/cost), LLMOllamaAdapter (minimal) |
| **7** | **Local-first + Docker Compose-native** — designed as drop-in replacement for `ollama` service | Matches existing ollama-stack patterns (port, volume, env var conventions). | **Low-Medium** — Easy to copy once proven. First-mover advantage matters. | No competitor targets ollama-stack specifically |

### Differentiator Impact Matrix

| Differentiator | User Problem Solved | Persona Impact | Implementation Complexity |
|----------------|-------------------|----------------|--------------------------|
| Ollama API surface | P2 (no Ollama-native gateway) | Alex, Priya — critical | Medium (3-5 endpoints, NDJSON streaming) |
| ChatGPT Plus proxy | P4 (no ChatGPT API access) | Sam — high value | High (cookie management, PoW solving) |
| Zero external DB | P3 (heavy gateways) | Priya, Morgan — high | Low (stateless architecture) |
| Stats dashboard | P5 (invisible costs) | Jordan — high | Medium (data collection, aggregation, charting) |
| Provider config UI | P7 (complex config) | Morgan, Alex — medium | Medium (web forms, API endpoints) |
| Unified `/api/tags` | P6 (no unified discovery) | Alex, Priya — medium | Low (merge local + configured model lists) |
| Docker Compose-native | P1 (no unified endpoint) | All — high | Low (single service definition) |

---

## 6. Unique Selling Proposition (USP)

### Core USP Statement

> **OllaRouter: The only LLM gateway that speaks Ollama natively.**
>
> Route requests from any Ollama-compatible client — Open WebUI, LangChain, continue.dev, or the `ollama` CLI — to OpenAI, Anthropic, Google, local Ollama, or ChatGPT Plus. Automatic failover. Zero external databases. Cost dashboard included.
>
> Add one service to your `docker-compose.yml`, change one environment variable in Open WebUI, and unlock every major LLM provider through the API your tools already speak.

### USP Breakdown by Audience

| Audience | What OllaRouter Says to Them |
|----------|------------------------------|
| **Open WebUI users** | "Set OLLAMA_BASE_URL to OllaRouter and instantly use GPT-4, Claude, Gemini, and your local models — all from the same chat interface." |
| **Homelab operators** | "One container, no database, no config complexity. Add OllaRouter to your existing ollama-stack in 30 seconds." |
| **Cost-conscious teams** | "See exactly what you're spending per model and provider — no extra observability tools required." |
| **ChatGPT Plus subscribers** | "Your $20/mo subscription becomes a programmable API. Use GPT-5 models in your own tools." |
| **Platform engineers** | "API key auth, rate limiting, Prometheus metrics, and a provider UI — all in a single 20MB Go binary." |

---

## 7. Functional Requirements

### FR-1: Ollama API Mirroring (Core Differentiator)

| ID | Requirement | Priority | Source |
|----|------------|----------|--------|
| FR-1.1 | Expose `POST /api/chat` with Ollama-compatible request/response format (NDJSON streaming for stream=true, JSON for stream=false) | P0 | Ollama API spec, Market research §7 |
| FR-1.2 | Expose `POST /api/generate` with Ollama-compatible request/response format (NDJSON streaming for stream=true, JSON for stream=false) | P0 | Ollama API spec |
| FR-1.3 | Expose `GET /api/tags` returning a unified list of all available models from all configured providers | P0 | Ollama API spec, Competitor analysis §9 |
| FR-1.4 | Expose `GET /api/version` returning OllaRouter version and supported API version | P1 | Ollama API spec |
| FR-1.5 | Expose `POST /api/show` returning model details (parameters, template) for known models | P2 | Ollama API spec |
| FR-1.6 | Support `Content-Type: application/x-ndjson` for streaming responses (Ollama format) | P0 | Market research §7.3 |
| FR-1.7 | Support `Content-Type: application/json` for non-streaming responses | P0 | Market research §7.4 |
| FR-1.8 | Wrap all provider errors (4xx/5xx) in Ollama-compatible error format `{"error":"..."}` | P0 | Market research §7.5 |

### FR-2: Provider Routing & Configuration

| ID | Requirement | Priority | Source |
|----|------------|----------|--------|
| FR-2.1 | Support YAML-based provider configuration file mounted at runtime | P0 | Competitor analysis §5 |
| FR-2.2 | Support OpenAI-compatible API provider type (`openai`, `openai-compatible`) | P0 | Market research §9 |
| FR-2.3 | Support Anthropic provider type | P0 | Market research §9 |
| FR-2.4 | Support Google/Gemini provider type | P1 | Competitor analysis |
| FR-2.5 | Support local Ollama as first-class connected provider (auto-discover models) | P0 | Competitor analysis §9 |
| FR-2.6 | Support ChatGPT Plus web proxy as a provider type (cookie-based) | P1 | Market research §3.6 |
| FR-2.7 | Model-based routing: user requests `gpt-4o` → route to OpenAI, `claude-sonnet-4` → route to Anthropic | P0 | Competitor analysis §9 (provider routing) |
| FR-2.8 | Support provider-specific model name mapping (e.g., user model `gpt-4o` → provider model `gpt-4o-2024-11-20`) | P1 | OpenRouter pattern |
| FR-2.9 | Support environment variable interpolation in YAML config for API keys (`${OPENAI_API_KEY}`) | P0 | Docker/12-factor pattern |
| FR-2.10 | Support multiple provider instances of the same type (e.g., two OpenAI accounts) | P2 | LiteLLM pattern |

### FR-3: Fallback & Failover

| ID | Requirement | Priority | Source |
|----|------------|----------|--------|
| FR-3.1 | Support per-model fallback chains (e.g., `gpt-4o` → `claude-sonnet-4` → local `llama3`) | P0 | Competitor analysis §9 |
| FR-3.2 | Auto-retry on 429 (rate limited) with configurable exponential backoff | P0 | Competitor analysis §2.1 |
| FR-3.3 | Auto-retry on 5xx (server error) with configurable max retries | P0 | Competitor analysis §2.1 |
| FR-3.4 | Circuit breaker pattern: temporarily disable failing providers after N consecutive failures | P1 | OllamaProxy pattern |
| FR-3.5 | Support configurable timeout per provider | P1 | Standard gateway feature |
| FR-3.6 | Log fallback events for debugging and auditing | P1 | Operational requirement |

### FR-4: Cost & Usage Tracking

| ID | Requirement | Priority | Source |
|----|------------|----------|--------|
| FR-4.1 | Track token usage (input + output) per request | P0 | Market research §1 (80% invisible costs) |
| FR-4.2 | Calculate and store estimated cost per request (using provider pricing model) | P0 | Market research §1 |
| FR-4.3 | Track usage per provider, model, and API key | P1 | Jordan persona |
| FR-4.4 | Provide cumulative usage/cost statistics (today, this week, this month) | P1 | Jordan persona |
| FR-4.5 | Support configurable budget alerts/limits per API key | P2 | Morgan persona |
| FR-4.6 | Store usage data in-memory with optional persistence to volume-mounted file | P1 | Stateless design goal |

### FR-5: Provider Configuration Web UI

| ID | Requirement | Priority | Source |
|----|------------|----------|--------|
| FR-5.1 | Web-based UI for adding, editing, and removing providers | P1 | Competitor analysis §5 |
| FR-5.2 | Web-based UI for viewing current provider status (online/offline/error) | P1 | Morgan persona |
| FR-5.3 | Web-based UI for viewing /api/tags model catalog | P2 | Alex persona |
| FR-5.4 | Web UI must read/write to the provider configuration file or runtime state | P1 | Operational requirement |
| FR-5.5 | Web UI must be served on a configurable port (separate from proxy port) | P1 | Security separation |

### FR-6: Stats Dashboard UI

| ID | Requirement | Priority | Source |
|----|------------|----------|--------|
| FR-6.1 | Web dashboard displaying per-provider token usage (input/output) | P1 | Jordan persona |
| FR-6.2 | Web dashboard displaying per-provider estimated cost with time range filter | P1 | Jordan persona |
| FR-6.3 | Web dashboard displaying per-model usage breakdown | P2 | Jordan persona |
| FR-6.4 | Web dashboard displaying request latency per provider | P2 | Alex persona |
| FR-6.5 | Web dashboard displaying recent request log with status codes | P1 | Operational requirement |

### FR-7: API Key Authentication

| ID | Requirement | Priority | Source |
|----|------------|----------|--------|
| FR-7.1 | Support optional API key authentication on proxy endpoints | P0 | Competitor analysis §9 |
| FR-7.2 | Support multiple API keys with optional per-key rate limits | P1 | Morgan persona |
| FR-7.3 | API keys configurable via YAML or environment variable | P1 | Docker pattern |
| FR-7.4 | Return 401 Unauthorized for requests with missing/invalid API keys when auth is enabled | P0 | Standard auth pattern |

### FR-8: Monitoring & Observability

| ID | Requirement | Priority | Source |
|----|------------|----------|--------|
| FR-8.1 | Expose `/health` endpoint returning 200 OK with service status | P0 | Docker best practice |
| FR-8.2 | Expose `/health/providers` endpoint returning per-provider status (online/offline) | P1 | OllamaProxy pattern |
| FR-8.3 | Expose `/metrics` endpoint in Prometheus format | P1 | Morgan persona, Competitor analysis §9 |
| FR-8.4 | Log requests in structured JSON format (method, path, status, latency, model, provider) | P1 | Operational requirement |
| FR-8.5 | Support configurable log level (debug, info, warn, error) | P1 | Operational requirement |

### FR-9: Docker Deployment

| ID | Requirement | Priority | Source |
|----|------------|----------|--------|
| FR-9.1 | Provide Dockerfile producing minimal container image (<50MB) | P0 | Market research §8 |
| FR-9.2 | Provide docker-compose.yml snippet for integration with ollama-stack | P0 | Market research §8.2 |
| FR-9.3 | Support configuration via environment variables for all critical settings | P0 | Market research §8 |
| FR-9.4 | Support read-only volume mount for provider config file | P0 | Docker best practice |
| FR-9.5 | Include Docker HEALTHCHECK instruction | P0 | Docker best practice |

---

## 8. Non-Functional Requirements

### NFR-1: Performance

| ID | Requirement | Target | Source |
|----|-------------|--------|--------|
| NFR-1.1 | Gateway overhead (latency added to provider response) | <10ms at P50, <50ms at P99 | Bifrost benchmark (11µs) is best-in-class; OllaRouter targets competitive with LiteLLM/Portkey |
| NFR-1.2 | Concurrent request handling | Support 100+ simultaneous streaming connections | Homelab/team use case |
| NFR-1.3 | Memory footprint (idle) | <20MB RSS | Market research §8 (targets <50MB, idle should be much lower) |
| NFR-1.4 | Memory footprint (under load) | <200MB RSS during active streaming | Market research §8 |
| NFR-1.5 | CPU usage (idle) | <0.1 core | Market research §8 (proxy is I/O bound) |
| NFR-1.6 | Startup time | <500ms from container start to ready | Go binary advantage |

### NFR-2: Security

| ID | Requirement | Priority | Source |
|----|-------------|----------|--------|
| NFR-2.1 | API keys must never be logged in plaintext (log as `sk-...xyz`) | P0 | Security best practice |
| NFR-2.2 | Admin UI port must be separate from proxy port and configurable | P1 | Security separation |
| NFR-2.3 | Support TLS/HTTPS on both proxy and admin ports (optional) | P2 | Production readiness |
| NFR-2.4 | No hardcoded secrets in the binary | P0 | Security best practice |
| NFR-2.5 | API key validation on all proxy endpoints when auth enabled | P0 | FR-7.4 |

### NFR-3: Reliability

| ID | Requirement | Target | Source |
|----|-------------|--------|--------|
| NFR-3.1 | Uptime (within Docker with restart policy) | 99.9% (stateless, auto-restart) | Docker best practice |
| NFR-3.2 | Graceful shutdown — complete in-flight requests before exit | Supported | Production readiness |
| NFR-3.3 | Configuration hot-reload without restart | P1 target | Competitor analysis §8 (1LCB/ollama-gateway has this) |
| NFR-3.4 | No data loss on restart (usage stats stored in-memory, optional file persistence) | Supported | FR-4.6 |

### NFR-4: Maintainability

| ID | Requirement | Target | Source |
|----|-------------|--------|--------|
| NFR-4.1 | Language choice | Go (single binary, cross-compile, minimal container) | Competitor analysis §4, §11 |
| NFR-4.2 | Configuration format | YAML with env var interpolation | Competitor analysis §4 |
| NFR-4.3 | API documentation | OpenAPI 3.0 spec for all endpoints | Best practice |
| NFR-4.4 | Container image | Multi-arch (linux/amd64, linux/arm64) | Homelab diversity (Raspberry Pi, Apple Silicon) |

### NFR-5: Compatibility

| ID | Requirement | Target | Source |
|----|-------------|--------|--------|
| NFR-5.1 | Ollama API compatibility | Must pass Ollama client integration test suite | Core differentiator (FR-1.x) |
| NFR-5.2 | Ollama client compatibility | Open WebUI, LangChain Ollama, ollama CLI, continue.dev | Market research §1 |
| NFR-5.3 | NDJSON streaming format | Exact match with Ollama streaming format | Market research §7 |

---

## 9. Risks, Assumptions, Dependencies & Open Questions

### Risks

| # | Risk | Likelihood | Impact | Mitigation |
|---|------|-----------|--------|------------|
| R1 | **Ollama adds native multi-provider support** to their runtime | Medium | High (cannibalizes core value prop) | Monitor Ollama roadmap; differentiate on ChatGPT proxy + cost UI + lightweight design |
| R2 | **LiteLLM adds Ollama API adapter layer** | Medium | Medium (reduces uniqueness on API surface) | LiteLLM is Python+DB-heavy; OllaRouter wins on simplicity and homelab fit |
| R3 | **ChatGPT Plus proxy breaks due to anti-bot measures** | Medium-High | Medium (feature becomes unreliable) | Abstract provider interface; document limitations; keep other providers working independently |
| R4 | **Olla adds cloud provider routing** | Low-Medium | Medium (competes on local inference routing) | Olla targets local inference load balancing; different use case |
| R5 | **Low community adoption** | Medium | High (project lacks contributors/testers) | MIT license, clear documentation, ollama-stack community outreach |
| R6 | **API key leaks via stats dashboard** | Low | High (exposed provider keys) | Dashboard on separate port, auth-protected, keys masked in UI |
| R7 | **Streaming translation bugs** (NDJSON ↔ SSE) | Medium | Medium (broken responses) | Comprehensive streaming unit tests, integration tests with Open WebUI |

### Assumptions

| # | Assumption | Rationale |
|---|-----------|-----------|
| A1 | Users prefer Ollama API over OpenAI API for Ollama-compatible tools | Confirmed by Ollama's 172k+ GitHub stars and Open WebUI adoption |
| A2 | Homelab operators want zero-external-dependency gateways | Confirmed by competitor comparison: LiteLLM (requires PG+Redis) is excluded from many deployments |
| A3 | ChatGPT Plus subscribers want API access without separate billing | Proved by zhaees/chatgpt-proxy existence and 30+ model support |
| A4 | Go is the best language for this use case | Competitor analysis §4, §11: Go gives <20MB binary, <50MB RSS, ms startup |
| A5 | YAML configuration with env var interpolation is sufficient for MVP | Consistent with OllamaProxy, Olla, 1LCB patterns |
| A6 | Users have Docker and docker-compose installed | The project targets ollama-stack users who already use Docker |

### Dependencies

| # | Dependency | Impact | Notes |
|---|-----------|--------|-------|
| D1 | Ollama API specification stability | High — FR-1.x depend on stable API surface | Ollama API is stable and versioned |
| D2 | Provider API availability (OpenAI, Anthropic, Google) | High — FR-2.x depend on provider API stability | All have stable, documented APIs |
| D3 | zhaees/chatgpt-proxy architecture for ChatGPT Plus integration | Medium — FR-2.6 may reference or integrate this project | Must handle cookie refresh, PoW, Cloudflare challenges |
| D4 | Go standard library HTTP client capabilities | Medium — streaming, timeouts, connection pooling | net/http is mature and production-grade |
| D5 | Open WebUI compatibility for testing | Medium — integration test target | Open WebUI is open-source, dockerized |

### Open Questions

| # | Question | Impact | Suggested Resolution |
|---|----------|--------|---------------------|
| Q1 | Should OllaRouter listen on port 11434 (replacing Ollama) or a different port (e.g., 11435)? | Medium — affects docker-compose integration | Research: listen on configurable port, default 11435 alongside Ollama on 11434 |
| Q2 | Should ChatGPT Plus proxy be built-in or a separate companion container? | Medium — affects architecture complexity | Decision: build as pluggable provider interface; can run in-process or as external process |
| Q3 | Should cost tracking use provider-published pricing or custom pricing config? | Medium — affects accuracy | Decision: configurable pricing per provider with sensible defaults from published pricing |
| Q4 | Should usage data be persisted to disk or ephemeral (in-memory only)? | Medium — affects restart behavior | Decision: in-memory for MVP with optional volume-mounted file persistence |
| Q5 | Should the web UI be a separate process or embedded in the Go binary? | High — affects architecture, release artifacts | Decision: embedded web UI (single binary, serve from Go) for simplicity |
| Q6 | What is the minimum viable provider set for MVP? (OpenAI, Anthropic, Ollama — others?) | High — affects scope | Decision: OpenAI + Ollama + ChatGPT Plus for MVP; Anthropic + Google for v1.1 |
| Q7 | Should we support the `/api/embed` (embeddings) endpoint in MVP? | Medium — affects API completeness | Decision: defer to post-MVP (lower usage in chat workflows) |

---

## 10. UX Design Phase Plan

For each feature with UI impact, a dedicated UX design phase must occur **before** implementation begins. The following plan maps features to their UX design requirements.

### Features Requiring Pre-Implementation UX Design

| Feature | UX Phase | Key Design Outputs | Priority |
|---------|----------|-------------------|----------|
| **Provider Configuration Web UI** (FR-5) | UX-1 | Provider add/edit form, provider list, status indicators, YAML config sync | P1 |
| **Stats Dashboard** (FR-6) | UX-2 | Usage charts, cost breakdown, time range filter, request log table, model breakdown | P1 |
| **Unified Model Catalog View** (FR-1.3 / FR-5.3) | UX-3 | Model listing page, search/filter, provider badges, model details panel | P2 |
| **API Key Management UI** (FR-7) | UX-4 | API key list, create/edit form, usage per key, rate limit display | P2 |
| **Health / Provider Status UI** (FR-8.2) | UX-5 | Provider status dashboard, online/offline/error indicators, circuit breaker state | P2 |

### UX Design Phase Schedule

```
Phase UX-1: Provider Configuration UI
  └── Precedes: EPIC-2 (Provider Configuration)
  
Phase UX-2: Stats Dashboard
  └── Precedes: EPIC-4 (Cost Tracking & Dashboard)
  
Phase UX-3: Model Catalog View
  └── Precedes: EPIC-1 (Ollama API Core) — GET /api/tags UI
  
Phase UX-4: API Key Management
  └── Precedes: EPIC-3 (Auth & Security)
  
Phase UX-5: Provider Health Status
  └── Precedes: EPIC-2 (Provider Configuration)
```

---

## 11. UX Goals & Key User Flows

### UX Goals

| Goal | Description | Measure |
|------|-------------|---------|
| **G1: Zero-config onboarding** | User can deploy OllaRouter and route their first request within 2 minutes of reading the docs | Time-to-first-request <2 min |
| **G2: Familiar Ollama experience** | All Ollama endpoints behave identically to real Ollama; clients cannot distinguish | Passes Ollama client integration tests |
| **G3: Transparent cost visibility** | Dashboard shows meaningful cost data within 5 seconds of opening | Data loads in <5s |
| **G4: Effortless provider setup** | Adding a new provider takes <30 seconds via web UI | Task completion time <30s |
| **G5: Clear failure states** | Provider errors, fallback events, and rate limits are visible in both API responses and UI | Error visibility in all states |

### Key User Flows

#### Flow 1: Initial Deployment & First Request (Happy Path)

```
1. User adds ollarouter service to docker-compose.yml
2. User creates providers.yaml with OpenAI API key
3. User runs `docker compose up -d`
4. User checks `curl http://localhost:11435/health` → 200 OK
5. User sends `curl http://localhost:11435/api/chat -d '{"model":"gpt-4o","messages":[{"role":"user","content":"Hello"}]}'`
6. OllaRouter receives request, routes to OpenAI, returns Ollama-format response
7. User configures Open WebUI with OLLAMA_BASE_URL=http://ollarouter:11435
8. User chats in Open WebUI — model "gpt-4o" works through Ollama API
```

#### Flow 2: Adding a Provider via Web UI

```
1. User opens browser to http://localhost:11436 (admin UI port)
2. User sees provider list (empty: "No providers configured")
3. User clicks "Add Provider"
4. Form appears: "Provider Type" dropdown (OpenAI, Anthropic, Ollama, ChatGPT Plus)
5. User selects "OpenAI", enters API key, adds model list "gpt-4o,gpt-4o-mini"
6. User clicks "Save"
7. Provider appears in list with green "Online" status
8. User tests via "Test Connection" button → "Connection successful"
9. User returns to Open WebUI — new models now visible in `/api/tags`
```

#### Flow 3: Viewing Cost Dashboard

```
1. User opens browser to http://localhost:11436/dashboard
2. Dashboard shows: Total Cost (this month), Requests (today), Active Providers
3. Chart shows cost trend over last 7 days
4. Table below shows per-provider breakdown: provider, requests, tokens in/out, cost
5. User selects time range "Last 30 days" from dropdown
6. Data refreshes automatically
7. User clicks on "openai" row → drill-down shows per-model breakdown (gpt-4o, gpt-4o-mini)
```

#### Flow 4: Fallback Scenario

```
1. User configured fallback: gpt-4o → claude-sonnet-4 → local-llama3
2. User sends request with model "gpt-4o"
3. OpenAI returns 429 (rate limited)
4. OllaRouter automatically retries once with backoff → still 429
5. OllaRouter falls back to Claude → success
6. User receives response from Claude, formatted as Ollama NDJSON
7. Response metadata includes `x-ollarouter-provider: anthropic` header
8. User checks dashboard — fallback event is logged
```

#### Flow 5: ChatGPT Plus Integration

```
1. User has ChatGPT Plus subscription ($20/mo)
2. User obtains session cookie (via browser dev tools or helper script)
3. User adds provider in OllaRouter UI: type "ChatGPT Plus", paste cookie
4. OllaRouter validates cookie by calling ChatGPT API
5. New models appear in /api/tags: "chatgpt-gpt-5", "chatgpt-gpt-5.5", "chatgpt-o3", etc.
6. User sends chat request with model "chatgpt-gpt-5"
7. OllaRouter proxies through ChatGPT Plus, returns Ollama-format response
8. Dashboard tracks usage (note: ChatGPT Plus has fixed cost, no per-token tracking)
```

---

## 12. UX Acceptance Criteria

### UX-AC-1: Provider Configuration UI

| # | Criteria | Validation Method |
|---|----------|-------------------|
| 1 | Provider list page shows all configured providers with name, type, status (Online/Offline/Error), and model count | Visual inspection |
| 2 | "Add Provider" button is visible and functional when provider list is empty or populated | Visual inspection |
| 3 | Add Provider form includes: Provider Type dropdown (required), Display Name (required), API Key field (masked), Model list (comma-separated or multi-select) | Form inspection |
| 4 | Form validates required fields before submission with inline error messages | Test empty submission |
| 5 | Successful provider addition shows success toast, provider appears in list immediately | Visual inspection |
| 6 | Provider status indicator updates within 5 seconds of provider health change | Observe status change |
| 7 | "Edit" and "Delete" actions are available per provider in the list | Visual inspection |
| 8 | Delete action shows confirmation dialog before removing provider | Test delete flow |
| 9 | Provider config changes are reflected in runtime without container restart (hot-reload) | curl test after UI change |
| 10 | UI is responsive on tablet (768px) and mobile (375px) viewports | Responsive testing |

### UX-AC-2: Stats Dashboard

| # | Criteria | Validation Method |
|---|----------|-------------------|
| 1 | Dashboard loads within 3 seconds when stats data exists | Performance measurement |
| 2 | Summary cards show: Total Cost (current month), Total Requests, Active Providers | Visual inspection |
| 3 | Cost trend chart shows daily cost for selectable time ranges (7d, 30d, 90d) | Visual inspection |
| 4 | Provider breakdown table shows: Provider Name, Requests, Input Tokens, Output Tokens, Estimated Cost | Visual inspection |
| 5 | Time range selector (dropdown/buttons) changes dashboard data without page reload | Test range switching |
| 6 | Clicking a provider row drills down to per-model breakdown for that provider | Visual inspection |
| 7 | Empty state ("No usage data yet") is shown when no requests have been processed | Visual inspection |
| 8 | Real-time update: dashboard refreshes data every 30 seconds or on manual refresh | Observe data update |

### UX-AC-3: Model Catalog View

| # | Criteria | Validation Method |
|---|----------|-------------------|
| 1 | Model list shows all available models from all configured providers | Compare against provider model configs |
| 2 | Each model entry shows: model name, provider badge, status (available/unavailable) | Visual inspection |
| 3 | Search/filter input filters the model list in real-time as user types | Test search |
| 4 | Provider badge color matches provider type (e.g., OpenAI=green, Anthropic=purple) | Visual inspection |
| 5 | Empty state: "No models configured. Add a provider to see available models." | Visual inspection with no providers |

### UX-AC-4: API Key Management

| # | Criteria | Validation Method |
|---|----------|-------------------|
| 1 | API key list shows all configured keys with name, masked key preview, and status | Visual inspection |
| 2 | "Generate API Key" button creates a new key and shows it once (cannot be viewed again) | Test key generation |
| 3 | Key deletion shows confirmation dialog | Test delete flow |
| 4 | Per-key usage statistics are accessible from the key list | Visual inspection |

### UX-AC-5: Provider Health Status

| # | Criteria | Validation Method |
|---|----------|-------------------|
| 1 | Provider status page shows all providers with real-time health status | Visual inspection |
| 2 | Status indicators: green (Online), yellow (Degraded), red (Offline), grey (Unknown) | Visual inspection |
| 3 | Last health check timestamp is shown per provider | Visual inspection |
| 4 | Provider health is automatically checked every 30 seconds | Observe status updates |

---

## 13. Recommended EPICs

### EPIC-1: Ollama API Core (MVP)

| Field | Value |
|-------|-------|
| **ID** | EPIC-1 |
| **Title** | Ollama API Core — Multi-Provider Request Routing |
| **Priority** | P0 — MVP |
| **Depends On** | None |
| **Features** | FR-1.x (Ollama API mirroring), FR-2.x (Basic provider routing: OpenAI + Ollama), FR-3.x (Fallback/failover) |
| **Non-Functional** | NFR-1.x (Performance), NFR-5.x (Compatibility) |
| **UX Pre-Design** | None (API-only, no UI) |
| **Exit Criteria** | Open WebUI can chat through OllaRouter using OpenAI models and local Ollama models |

**User Story:** "As an Open WebUI user, I want to set OLLAMA_BASE_URL to OllaRouter and chat with GPT-4o through my existing interface, so I don't need to switch tools or manage multiple endpoints."

### EPIC-2: Provider Configuration UI (v1.0)

| Field | Value |
|-------|-------|
| **ID** | EPIC-2 |
| **Title** | Provider Configuration Web UI |
| **Priority** | P1 — v1.0 |
| **Depends On** | EPIC-1 |
| **Features** | FR-5.x (Provider configuration UI), FR-2.x (Provider management endpoints) |
| **Non-Functional** | NFR-2.x (Security — separate admin port) |
| **UX Pre-Design** | UX-1 (Provider Configuration UI), UX-5 (Provider Health Status) |
| **Exit Criteria** | Users can add/edit/delete providers via web UI and see their status |

**User Story:** "As a platform engineer, I want to add providers through a web form instead of editing YAML files, so my non-technical team members can manage LLM access."

### EPIC-3: Auth & Security (v1.0)

| Field | Value |
|-------|-------|
| **ID** | EPIC-3 |
| **Title** | API Key Authentication & Rate Limiting |
| **Priority** | P1 — v1.0 |
| **Depends On** | EPIC-1 |
| **Features** | FR-7.x (API key auth), T4 (Rate limiting) |
| **Non-Functional** | NFR-2.x (Security) |
| **UX Pre-Design** | UX-4 (API Key Management UI) |
| **Exit Criteria** | Gateway rejects unauthenticated requests, rate limits per key, keys manageable via UI |

**User Story:** "As a platform engineer, I want to create API keys for each team member with usage limits, so I can control costs and prevent unauthorized access."

### EPIC-4: Cost Tracking & Dashboard (v1.0)

| Field | Value |
|-------|-------|
| **ID** | EPIC-4 |
| **Title** | Usage Tracking & Stats Dashboard |
| **Priority** | P1 — v1.0 |
| **Depends On** | EPIC-1 |
| **Features** | FR-4.x (Cost tracking), FR-6.x (Stats dashboard) |
| **Non-Functional** | NFR-1.x (Performance — dashboard must not impact proxy) |
| **UX Pre-Design** | UX-2 (Stats Dashboard) |
| **Exit Criteria** | Dashboard shows per-provider token usage, cost, and request history |

**User Story:** "As a team lead, I want to see exactly how much each provider costs me per month, so I can optimize my LLM spending."

### EPIC-5: ChatGPT Plus Provider (v1.1)

| Field | Value |
|-------|-------|
| **ID** | EPIC-5 |
| **Title** | ChatGPT Plus Web Proxy Provider |
| **Priority** | P1 — v1.1 |
| **Depends On** | EPIC-1, EPIC-2 |
| **Features** | FR-2.6 (ChatGPT Plus provider type) |
| **Non-Functional** | NFR-2.x (Security — cookie handling) |
| **UX Pre-Design** | UX-1 (Add Provider form — extended for ChatGPT Plus type) |
| **Exit Criteria** | Users can configure ChatGPT Plus cookie and use ChatGPT models through the Ollama API |

**User Story:** "As a ChatGPT Plus subscriber, I want to use my subscription models through my existing tools, so I don't pay extra per-token API costs."

### EPIC-6: Observability & Monitoring (v1.1)

| Field | Value |
|-------|-------|
| **ID** | EPIC-6 |
| **Title** | Prometheus Metrics, Health Endpoints & Structured Logging |
| **Priority** | P1 — v1.1 |
| **Depends On** | EPIC-1 |
| **Features** | FR-8.x (Monitoring & observability) |
| **Non-Functional** | NFR-1.x (Minimal overhead from metrics collection) |
| **UX Pre-Design** | None (API-level feature, Grafana dashboards are user-responsibility) |
| **Exit Criteria** | Prometheus can scrape /metrics, health endpoints report provider status, logs are structured JSON |

**User Story:** "As a platform engineer, I want Prometheus metrics from OllaRouter, so I can monitor it alongside my other infrastructure."

### EPIC-7: Model Catalog & Extended API (v1.2)

| Field | Value |
|-------|-------|
| **ID** | EPIC-7 |
| **Title** | Enhanced Model Discovery & Extended Ollama Endpoints |
| **Priority** | P2 — v1.2 |
| **Depends On** | EPIC-1, EPIC-2 |
| **Features** | FR-1.4 (/api/version), FR-1.5 (/api/show), FR-1.6 (model catalog UI view) |
| **Non-Functional** | NFR-5.x (Compatibility) |
| **UX Pre-Design** | UX-3 (Model Catalog View) |
| **Exit Criteria** | Users can browse all available models in a web UI, see model details |

**User Story:** "As a developer, I want to see all available models from all providers in one place, so I can pick the best model for each task without memorizing provider-specific model names."

---

## 14. Traceability Links

### Entity Records

| Entity ID | Type | Title | Status |
|-----------|------|-------|--------|
| `c843c293-5a0f-40ab-87e2-5cda57a9e392` | idea | OllaRouter - Ollama-native Multi-Provider LLM Gateway | discovery |
| `4dbc9a4b-717a-4284-83e1-bdddde480d3d` | research | Phase 1 Market Research: OllaRouter | complete |
| `c5aaf9f3-bf1d-4b51-8e6d-ff47c0bed31a` | research | Phase 2 Competitor Analysis: OllaRouter | complete |
| `8b4a1870-aa6b-4e34-b4f7-cbbae43eac35` | epic | OllaRouter USP Definition and Backlog | defined |

### Link Map

```
[c843c293] OllaRouter Idea
    ├── derived_from ◀── [4dbc9a4b] Phase 1 Market Research
    └── derived_from ◀── [c5aaf9f3] Phase 2 Competitor Analysis

[8b4a1870] OllaRouter USP Definition
    └── derived_from ◀── [c843c293] OllaRouter Idea
```

### Requirements to EPIC Mapping

| Requirement Area | EPIC | Features |
|-----------------|------|----------|
| Ollama API + Provider routing + Fallback | EPIC-1 | FR-1.x, FR-2.x, FR-3.x |
| Provider Config UI | EPIC-2 | FR-5.x, UX-1, UX-5 |
| API Key Auth + Rate Limiting | EPIC-3 | FR-7.x, UX-4 |
| Cost Tracking + Dashboard | EPIC-4 | FR-4.x, FR-6.x, UX-2 |
| ChatGPT Plus Provider | EPIC-5 | FR-2.6 |
| Observability | EPIC-6 | FR-8.x |
| Model Catalog + Extended API | EPIC-7 | FR-1.x (extended), UX-3 |

---

## 15. Evidence Sources

### Primary Sources (from Phase 1 & 2)

| # | Source | URL | Evidence Type | Key Finding |
|---|--------|-----|---------------|-------------|
| 1 | Ollama API Reference | https://github.com/ollama/ollama/blob/main/docs/api.md | API Specification | `/api/chat`, `/api/generate`, `/api/tags` endpoints with NDJSON streaming |
| 2 | Ollama Streaming Docs | https://docs.ollama.com/api/streaming | Documentation | NDJSON format (`application/x-ndjson`), stream=false option |
| 3 | LiteLLM GitHub | https://github.com/BerriAI/litellm | Repository | 47.7k stars, 100+ providers, OpenAI format, Python, needs PG+Redis |
| 4 | Portkey AI Gateway | https://github.com/Portkey-AI/gateway | Repository | 11.8k stars, 1600+ models, 45+ providers, TypeScript, OpenAI format |
| 5 | OpenRouter Pricing | https://openrouter.ai/pricing | Documentation | 400+ models, 60+ providers, 5.5% fee, managed SaaS (no self-host) |
| 6 | Bifrost (Maxim AI) | https://github.com/maximhq/bifrost | Repository | 5.1k stars, Go, 11µs overhead, 23+ providers, OpenAI format |
| 7 | any-llm (Mozilla) | https://mozilla-ai.github.io/any-llm/gateway/overview/ | Documentation | FastAPI, budget management, virtual keys, needs PostgreSQL |
| 8 | Ollama Proxy | https://ollama-proxy.readthedocs.io/en/latest/ | Documentation | Python, **Ollama-native API**, multi-provider, routing strategies |
| 9 | Olla (thushan/olla) | https://github.com/thushan/olla | Repository | 224 stars, Go, <50MB, local inference load balancer, no cloud routing |
| 10 | 1LCB/ollama-gateway | https://github.com/1LCB/ollama-gateway | Repository | 16 stars, Go, Ollama-only reverse proxy with rate limiting |
| 11 | LLMOllamaAdapter | https://github.com/pure-san/LiteLLMOllamaAdapter | Repository | 1 star, Python/Flask, Ollama-native API, OpenRouter-only |
| 12 | zhaees/chatgpt-proxy | https://github.com/zhaees/chatgpt-proxy | Repository | 5 stars, cookie-based ChatGPT Plus proxy, 30+ models |
| 13 | ollama-stack docker-compose.yml | C:\Projects\ollama-stack\docker-compose.yml | Local file | Existing stack: ollama (11434) + open-webui (3000), GPU passthrough |

### Phase 3 Checkpoint Sources

| File | Description |
|------|-------------|
| `docs/product/traceability/checkpoints/01-market-research.md` | Phase 1 market research — market size, trends, user needs |
| `docs/product/traceability/checkpoints/02-competitor-analysis.md` | Phase 2 competitor analysis — 10 tools in deep comparison |
| `docs/product/market-research/01-ollarouter-market-research.md` | Cross-reference copy of market research |

### Saved Artifacts (this phase)

| File | Description |
|------|-------------|
| `docs/product/traceability/checkpoints/03-usp-definition.md` | **This document** — complete USP definition, requirements, UX plan, EPICs |
| `docs/product/product-analysis/ollarouter-usp-definition.md` | Product analysis artifact (link to this checkpoint) |
| `docs/product/ux-design/` | Directory for pre-implementation UX design outputs (upcoming phases) |
| `docs/product/backlog/` | Directory for EPIC/feature/story backlog artifacts (upcoming phases) |

---

*End of Phase 3 — USP Definition. Ready for Phase 4 UX Design (Pre-Build) and Phase 5 EPIC → Feature → Story creation.*
