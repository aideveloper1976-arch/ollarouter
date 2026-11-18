# Security

## Reporting a vulnerability

Please report vulnerabilities privately through
[GitHub security advisories](https://github.com/aideveloper1976-arch/ollarouter/security/advisories/new),
not in public issues. I aim to reply within 7 days.

## Security model

OllaRouter is built to run on your own machine or a trusted network, in front of provider
accounts you pay for. Anyone who can call its chat endpoints can spend your provider credit, so
treat its API keys like the provider keys behind them.

| Area | How it works |
|---|---|
| Client auth | Bearer API keys from `api_keys`. Held in memory as SHA-256 hashes and masked in logs and in `/admin/api/config/yaml` |
| **Auth off when no keys** | With an empty `api_keys` list, chat endpoints and the admin API are **open**. Always set at least one key unless OllaRouter only listens on localhost |
| Protected routes | `/api/chat`, `/api/generate`, `/v1/chat/completions`, `/chat/completions`, `/admin/api/*` (except `/admin/api/me`) |
| Public routes | Model discovery (`/api/tags`, `/api/show`, `/api/ps`, `/api/version`), `/health*`, `/metrics`, and the admin UI's static files (which hold no data) |
| Rate limiting | Token bucket per bearer token, 60 requests/minute |
| Provider secrets | Read from the environment via `${VAR}`. `providers.yaml` and `.env` are git-ignored. Config written by the admin API is saved with mode `0600` |
| Upstream OAuth | PKCE (S256) with a random `state`. Tokens are stored with AES-256-GCM in `upstream_auth_credential_path` |
| Container | Distroless `static-debian12:nonroot`, static binary, no shell |
| Admin CORS | `/admin` allows any origin without credentials. The bearer key is still required |
| Supply chain | Three Go dependencies. Dependabot, gitleaks and CodeQL run in CI |

## Known gaps

- `monthly_budget_usd` is not enforced, and the per-key `rate_limit` value is not applied.
  Every key gets the default bucket.
- The upstream credential key is truncated or zero-padded to 32 bytes, not passed through a key
  derivation function. Use 32 random bytes (`openssl rand -base64 24` gives 32 characters).
- The rate limiter keys on the raw `Authorization` header, so it does not stop someone sending
  many different invalid keys. Auth still rejects them.
- The bundled Open WebUI in `docker-compose.yml` has its own login turned off
  (`WEBUI_AUTH=False`). Don't expose port 3000 beyond your machine.
- The `chatgpt-plus` provider uses an unofficial ChatGPT backend. Use your own account only.

## Before you expose it beyond localhost

1. Set strong `api_keys` (`openssl rand -hex 32`) and keep `openai_endpoint_public: false`.
2. Put it behind TLS (a reverse proxy such as Caddy or nginx).
3. Restrict `/admin` and `/metrics` at the proxy if others can reach the host.
4. Use provider keys with spending limits set at the provider.
