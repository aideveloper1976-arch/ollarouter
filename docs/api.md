# API reference

"Key" means `Authorization: Bearer <key>` is required when `api_keys` is configured.

## Ollama-compatible API

| Method | Path | Auth | Purpose |
|---|---|---|---|
| GET | `/api/tags` (also `/api/models`) | public | List all models as `provider/model`, in Ollama's format |
| POST/GET | `/api/show` | public | Model details |
| GET | `/api/ps` | public | Running models (Ollama compatibility) |
| GET | `/api/version` | public | OllaRouter version |
| POST | `/api/chat` | key + rate limit | Chat. NDJSON stream when `"stream": true` |
| POST | `/api/generate` | key + rate limit | Text completion |

Model discovery is public so tools like Open WebUI can list models before the user sets a key.

```bash
curl localhost:11435/api/chat -H "Authorization: Bearer $KEY" -d '{
  "model": "best-quality",
  "messages": [{"role": "user", "content": "Summarise RFC 2119 in one line"}],
  "stream": true
}'
```

Tool definitions (`tools`), tool calls, tool results (`tool_call_id`), images and
`reasoning_content` pass through.

## OpenAI-compatible API

| Method | Path | Auth | Purpose |
|---|---|---|---|
| POST | `/v1/chat/completions` | key + rate limit | Chat completion. SSE stream ending in `data: [DONE]` when `"stream": true` |
| POST | `/chat/completions` | key + rate limit | Same, for clients that leave out `/v1` |

Set `settings.openai_endpoint_public: true` only if a client cannot send a key.

## Health and metrics

| Method | Path | Purpose |
|---|---|---|
| GET | `/health` | Checks all providers in parallel (15s each). `200` if at least one is online, otherwise `503` |
| GET | `/health/providers` | Per-provider status, latency, model count and circuit breaker |
| GET | `/metrics` | Prometheus text: uptime, providers total/online, goroutines, memory, version |

## Admin API

Everything under `/admin/api` except `/me` needs a key when keys are configured.

| Method | Path | Purpose |
|---|---|---|
| GET | `/admin/api/me` | Public stub that always returns `{"authenticated": true}`. Kept for UI compatibility |
| GET / POST | `/admin/api/providers` | List or add providers |
| PUT / DELETE | `/admin/api/providers/{name}` | Update or remove a provider |
| GET | `/admin/api/providers/{name}/test` | Live health check |
| GET | `/admin/api/providers/{name}/balance` | Account balance (DeepSeek, OpenRouter) |
| POST | `/admin/api/providers/discover` | Fetch a provider's model list before saving it |
| GET | `/admin/api/stats/summary` | Requests, tokens, cost, active providers |
| GET | `/admin/api/stats/cost-by-provider` | Cost per provider |
| GET | `/admin/api/stats/cost-by-model` | Requests, tokens and cost per model, grouped by provider |
| GET | `/admin/api/logs?provider=&limit=` | Recent usage records |
| GET / PUT | `/admin/api/settings` | Read or change runtime settings |
| GET | `/admin/api/config/yaml` | Current config as YAML, with keys masked |
| GET | `/admin/api/auth/status` | Upstream OAuth status |
| POST | `/admin/api/auth/connect` | Start the OAuth (PKCE) sign-in for a provider |
| GET / POST | `/admin/api/auth/callback` | OAuth callback |
| POST | `/admin/api/auth/disconnect` | Remove stored upstream credentials |

The admin console is served at `/admin/`. `/` redirects there.
