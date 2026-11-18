# Configuration

OllaRouter reads one YAML file. It looks in this order:

1. `-config <path>`
2. `OLLAROUTER_CONFIG_PATH`
3. `providers.yaml` in the working directory
4. If that fails: `/app/config/providers.yaml`, `/app/providers.yaml.example`, `./providers.yaml`,
   `./providers.yaml.example`

Any `${VAR}` or `${VAR:-default}` in the file is replaced from the environment before parsing.
Keep secrets in the environment and commit only `providers.yaml.example`. `providers.yaml` and
`.env` are git-ignored.

The admin console can add, edit and delete providers and change settings at runtime. Those
changes are written back to the same file with mode `0600`.

## `settings`

| Key | Default | Notes |
|---|---|---|
| `port` | `11435` | Overridden by `OLLAROUTER_PORT` if set |
| `log_level` | `info` | `debug`, `info`, `warn`, `error` |
| `log_format` | `json` | `json` or `text` |
| `prometheus_enabled` | `true` | Serves `GET /metrics` |
| `rate_limiting_enabled` | `true` | Token bucket per bearer token: 60 requests/minute, refilling at 1/second |
| `default_max_retries` | `3` | Retries per provider before failing over. `0` = fail over at once |
| `default_timeout_seconds` | `30` | Default upstream timeout |
| `usage_persistence` | `memory` | `memory` or `file` |
| `usage_file_path` | `/data/usage.json` | Used when `usage_persistence: file`. Saved every minute |
| `circuit_breaker_threshold` | `5` | Consecutive failures before a provider's breaker opens |
| `circuit_breaker_cooldown_seconds` | `30` | How long before an open breaker lets a trial request through |
| `openai_endpoint_public` | `false` | `true` skips API-key auth on `/v1/chat/completions`. Only for clients that cannot send a key |
| `upstream_auth_credential_path` | `/data/upstream-auth.enc` | Encrypted store for provider OAuth tokens |
| `upstream_auth_encryption_key` | empty | Required (16+ bytes, use 32 random bytes) for providers with an `auth` block |

## `api_keys`

```yaml
api_keys:
  - name: admin-key
    key: "${OLLAROUTER_API_KEY}"
    rate_limit: 100            # stored; not applied yet (see limitations)
    monthly_budget_usd: 100    # stored and shown; not enforced
```

If the list is empty, **authentication is off**: chat endpoints and the admin API are open.
If it has at least one key, `/api/chat`, `/api/generate`, `/v1/chat/completions` and
`/admin/api/*` need `Authorization: Bearer <key>`. Keys are kept in memory as SHA-256 hashes.

## `providers`

```yaml
providers:
  - name: deepseek                # used in model names: deepseek/deepseek-chat
    type: deepseek
    base_url: https://api.deepseek.com
    api_key: "${DEEPSEEK_API_KEY}"
    models:
      - name: deepseek-chat
        max_tokens: 8192
    pricing:                      # USD per 1,000 tokens, used for cost tracking
      input_per_1k: 0.00027
      output_per_1k: 0.0011
    max_retries: 3
    timeout_seconds: 60
    headers:                      # optional extra request headers
      X-Custom: value
```

| `type` | Adapter | Notes |
|---|---|---|
| `openai` | OpenAI | Optional `organization_id` |
| `anthropic` | Anthropic Messages API | Optional `anthropic_version` |
| `google` | Gemini `generateContent` / `streamGenerateContent` | |
| `deepseek` | OpenAI-compatible | Balance via `GET /user/balance` |
| `openrouter` | OpenAI-compatible | Balance via OpenRouter credits API |
| `openai-compatible` | OpenAI-compatible | Groq, Together, llama.cpp server, vLLM, LM Studio… Guardrails on by default |
| `ollama` | Ollama native | `auto_discover: true` reads models from `/api/tags`. Guardrails on by default |
| `chatgpt-plus` | ChatGPT Codex backend | Needs an `auth` block (OAuth with PKCE) or a `session_cookie`. Unofficial |

`auto_discover: true` fetches the model list from the provider instead of listing `models` by hand.

### Guardrails per provider

```yaml
    guardrails:
      enabled: true
      tool_call_validation: true
      rescue_malformed_tool_calls: true
      mixed_output_policy: tool_call_first   # or text_first
      max_retries: 3
```

See [guardrails.md](guardrails.md).

## `virtual_models`

```yaml
virtual_models:
  - name: fast-chat
    fallback_chain:
      - openai-main/gpt-4o-mini
      - groq/llama3-8b-8192
  - name: local-first
    fallback_chain:
      - local-ollama/*            # first model the provider reports
      - deepseek/deepseek-chat
```

Entries are tried in order. A provider is skipped if it isn't registered or its circuit breaker
is open. Repeated entries are skipped, so a chain can't loop. Retryable errors (unavailable,
timeout, rate-limited) are retried with exponential backoff (1s base, 30s cap, ±25% jitter) before
moving on. Other errors move to the next entry straight away.

## Environment variables

| Variable | Effect |
|---|---|
| `OLLAROUTER_CONFIG_PATH` | Config file path |
| `OLLAROUTER_PORT` | Listen port, overriding `settings.port`. Also used by `-healthcheck` |
| Anything referenced as `${VAR}` | Substituted into the config |
