# Verification status

This page says what has been checked and how, so the README's claims can be audited.

## Checked automatically in CI on every push

| Check | Command | What it proves |
|---|---|---|
| Formatting and vet | `gofmt -l .` must be empty, `go vet ./...` | Code is formatted and passes static checks |
| Unit and integration tests | `go test -race ./...` | 190 test functions in 25 files: config parsing, interpolation and the example config, auth and rate limiting, OAuth/PKCE and the encrypted store, resolver, fallback, retry and circuit breaker, the OpenAI, Anthropic, OpenAI-compatible, Ollama and ChatGPT adapters against `httptest` servers (the Gemini adapter has no unit tests yet), stream translators, guardrails, sanitiser, admin routes and auth on the OpenAI endpoint. The race detector is on |
| Admin UI build | `npm ci && npm run build` in `webui/` | TypeScript type-checks and the production bundle builds |
| End-to-end smoke test | `scripts/smoke-test.sh` | 15 checks against the real binary and a mock provider (below) |
| Docker image | `docker build -f ollarouter/Dockerfile .` | The 3-stage image builds from a clean checkout |
| Secret scan | gitleaks over the full git history | No credentials committed |
| Static analysis | CodeQL for Go and JavaScript/TypeScript | No known vulnerability patterns flagged |

### Smoke test checks

The script builds the binary, starts `scripts/mock_provider.py` (an OpenAI-compatible mock that
echoes messages, as JSON or SSE) and a second provider pointing at a closed port, then checks:

1. `/api/version` reports the version injected at build time
2. `/api/tags` lists models as `provider/model`
3. `/api/chat` without a key → `401`
4. `/v1/chat/completions` without a key → `401`
5. Ollama chat to a direct `provider/model`
6. Ollama generate
7. A virtual model fails over from the dead provider to the working one
8. Ollama NDJSON streaming through that fallback chain
9. OpenAI chat completion
10. OpenAI SSE streaming ending in `data: [DONE]`
11. Cost is recorded from the provider's configured pricing
12. Admin API without a key → `401`
13. `/health` reports one provider online and one offline
14. Prometheus `/metrics`
15. The embedded admin UI is served

## Checked by hand for this release (18 Nov 2026)

- All of the above run locally on macOS (Go 1.27 toolchain, `go 1.23` module), all passing.
- The admin console was loaded in headless Chromium against the smoke-test setup. That produced
  the screenshots in `docs/assets/`. It prompted once for the API key, then dashboard, providers
  and usage pages loaded data, and the circuit-breaker column showed failures on the dead provider.

## Bugs found and fixed while preparing this release

| Issue | Fix | Test |
|---|---|---|
| Concurrent requests could crash the process with a concurrent map write in the fallback engine's circuit-breaker map | Map guarded by a mutex | `TestGetOrCreateBreakerConcurrent` (fails under `-race` before the fix) |
| `circuit_breaker_threshold` and `circuit_breaker_cooldown_seconds` were ignored | Wired from config into the fallback engine | `TestSetCircuitBreakerConfig` |
| `/v1/chat/completions` skipped API-key auth even when keys were configured | Same auth and rate limiting as `/api/chat`, with an explicit `openai_endpoint_public` opt-out | `TestOpenAIEndpointRequiresAuthWhenKeysConfigured`, `TestOpenAIEndpointPublicOptOut`, smoke check 4 |
| The admin UI never sent an API key, so it couldn't load anything once keys were configured | UI asks for a key once, keeps it in `sessionStorage` and sends it as a bearer token | Browser check above |
| Admin "circuit breaker" column always showed `closed (0)` | Shows the live breaker from the fallback engine | Browser check above |
| `default_max_retries: 0` was treated as "use the default of 3" | `0` now means fail over at once | Smoke check 7 (failover in ~40 ms instead of ~7 s) |
| `/api/version`, `/health` and `/metrics` reported different versions | One build-time version everywhere | Smoke check 1 |
| `OLLAROUTER_PORT` in Docker Compose only changed the healthcheck, not the listen port | `OLLAROUTER_PORT` now overrides `settings.port` | Run by hand |

## Not verified in this release

- **Real upstream providers.** Adapters are tested against `httptest` servers that copy each
  provider's wire format. This release was not run against live OpenAI, Anthropic, Gemini,
  DeepSeek, OpenRouter, Groq or ChatGPT accounts. The Gemini adapter has no unit tests.
- **Docker Compose with Open WebUI.** The image build is checked in CI. The full Compose stack was
  not started for this release.
- **Load.** No load or soak test has been run. The race fix above was found by inspection and
  confirmed with the race detector, not under load.
