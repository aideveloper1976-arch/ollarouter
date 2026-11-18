# LiteLLM vs OllaRouter Feature Comparison

## LiteLLM (BerriAI) — Current Feature Set
- Open-source AI Gateway (Python SDK + Proxy Server)
- 100+ LLM providers via OpenAI-compatible format
- Retry/fallback across deployments (Router)
- Cost tracking & spend management per project
- Virtual keys, rate limiting, guardrails
- Admin dashboard UI for monitoring
- Observability callbacks (MLflow, Langfuse, Lunary, etc.)
- Docker deployment (requires PostgreSQL in production)
- ChatGPT Plus/Pro subscription via **OAuth device flow** → `/backend-api/codex`
- Ollama as a provider (via `ollama_chat/` → `/api/chat`)

## Key Gaps in LiteLLM That OllaRouter Fills

| Feature | LiteLLM | OllaRouter (Planned) |
|---------|---------|---------------------|
| **API Surface** | OpenAI `/v1/chat/completions` only | Ollama-native `/api/chat`, `/api/generate`, `/api/tags` |
| **Model Listing** | OpenAI-format model list | Ollama `GET /api/tags` with `provider/model` format |
| **ChatGPT Plus Auth** | OAuth device flow only (Codex endpoint) | Session token / access token via `chatgpt.com/backend-api/conversation` |
| **SSE→NDJSON Transform** | No (uses OpenAI SSE format) | Required for ChatGPT Plus web proxy stream conversion |
| **Provider Config UI** | No (admin dashboard for spend only) | Yes — add/edit providers, API keys, endpoints |
| **Token/Cost Dashboard** | In admin UI (requires DB) | Lightweight built-in dashboard with daily/weekly/monthly aggregations |
| **External DB** | PostgreSQL required for prod | None — config-driven, in-memory stats |
| **Fallback Array** | Per-deployment retry | Per-virtual-model ordered fallback chain with silent failover |
| **Local-First** | Enterprise-focused | Designed for `docker compose up` alongside Ollama |
| **deployment size** | ~300MB+ image + DB | Target: minimal Python image, no external services |

## LiteLLM Strengths (not competing on)
- Largest provider count (100+)
- Enterprise features (SSO/SAML, audit logs, multi-tenant)
- Mature SDK and ecosystem integrations
- Production-proven at scale

## OllaRouter Positioning
OllaRouter is **not** a LiteLLM replacement. It fills a specific niche: **Ollama-native API gateway for local-first development** — targeting developers who want to use Ollama-compatible tools (Continue.dev, Page Assist, Open WebUI, LlamaIndex) with cloud provider fallback, without switching to OpenAI's API format.

Sources:
- https://github.com/BerriAI/litellm
- https://docs.litellm.ai/docs/providers/chatgpt
- https://docs.litellm.ai/docs/providers/ollama
- https://github.com/pure-san/LiteLLMOllamaAdapter
