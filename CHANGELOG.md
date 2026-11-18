# Changelog

All notable changes are listed here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.22.0] - 2026-11-18

First public release.

### Security
- `/v1/chat/completions` and `/chat/completions` now need an API key when `api_keys` are
  configured, with the same rate limiting as `/api/chat`. **Behaviour change:** set
  `settings.openai_endpoint_public: true` to restore the old open behaviour for clients that
  cannot send a key.

### Fixed
- Possible crash from concurrent map writes in the fallback engine's circuit breakers.
- `circuit_breaker_threshold` and `circuit_breaker_cooldown_seconds` are now applied.
- `default_max_retries: 0` now disables retries instead of falling back to 3.
- The admin console sends an API key, so it works when keys are configured.
- The admin console shows live circuit-breaker state and the real version.
- `/api/version`, `/health` and `/metrics` report the same version.
- `OLLAROUTER_PORT` now sets the listen port, matching the Compose file and `-healthcheck`.

### Added
- End-to-end smoke test (`scripts/smoke-test.sh`) with a mock OpenAI-compatible provider.
- CI: gofmt, vet, race-enabled tests, UI build, smoke test, Docker build, gitleaks, CodeQL,
  Dependabot.
- Documentation: getting started, configuration, API reference, architecture, guardrails,
  verification status, security model.

### Changed
- Go module path is now `github.com/aideveloper1976-arch/ollarouter`.
- Code formatted with `gofmt`. Debug-only files removed.

## [0.21.0] - 2026-07-06

- DeepSeek balance support end to end, llama.cpp host networking fix.

## Earlier

- Tool-call guardrails for local models, upstream OAuth (ChatGPT Plus) with an encrypted
  credential store, admin console, fallback chains and cost tracking. See `docs/product` and
  `ollarouter/docs` for the build records.
