# Architect Validation: Hybrid ChatGPT/Codex Auth

**Checkpoint:** 01-architect-validation  
**Date:** 2026-05-22  
**Agent:** architect (fullstack-solution-architect)  
**Plan artifact:** `docs/plans/2026-05-22-hybrid-chatgpt-auth-plan.md`  
**Jira linkage:** None — no Jira story/feature/epic keys were provided. This is inferred as part of an unregistered EPIC "Hybrid upstream ChatGPT/Codex auth" for the OllaRouter project.

---

## Story/Feature/EPIC Context

| Hierarchy | Name | Status |
|-----------|------|--------|
| EPIC (inferred) | Hybrid upstream ChatGPT/Codex auth | Not created in Jira |
| Feature (inferred) | Hybrid auth subsystem | Not created in Jira |
| Story | Hybrid ChatGPT auth implementation | Plan artifact only |

The plan artifact defines the implementation goal: add a hybrid upstream ChatGPT/Codex auth system to OllaRouter with local callback, remote callback, keychain-first credential storage, encrypted file fallback, admin auth lifecycle endpoints, and legacy session cookie compatibility.

---

## Acceptance Criteria Validation

| # | AC | Code Baseline | Validation | Gaps |
|---|----|---------------|------------|------|
| 1 | Provider config supports structured auth settings for hybrid upstream auth | `config.ProviderConfig` has `SessionCookie` + `UserAgent` fields only; no structured auth settings object | ✅ Reasonable — needs a new `AuthSettings` struct (e.g., `mode`, `client_id`, `redirect_uri`, `keychain_service`) | The exact YAML/struct shape is not defined in the plan |
| 2 | Upstream auth subsystem supports auth sessions, callback validation, and persisted credential status | No upstream auth exists; `internal/auth/` only handles downstream API key auth | ✅ Reasonable — this is a new `internal/auth/upstream/` package | Scope of PKCE flow (local vs remote callback) is undefined |
| 3 | Credential storage prefers OS keychain and falls back to encrypted file storage | No credential storage exists; all credentials live in plain YAML config | ⚠️ **Needs clarification** — no keychain library specified, no encryption algorithm defined | Which keychain API? (macOS, WinCred, secret-service) Which encryption? (AES-GCM, NaCl secretbox) Cross-platform strategy? |
| 4 | Admin routes for auth status, connect, callback, and disconnect | `admin_handlers.go` has provider CRUD, settings, stats, logs | ✅ Reasonable — follows existing pattern in `router.go` `/admin/api/` routes | Route naming convention needs definition |
| 5 | ChatGPT provider can use stored bearer credentials and still fall back to legacy `session_cookie` mode | `chatgpt/adapter.go` uses only `sessionCookie` for auth in `setHeaders()` | ✅ Reasonable — refactor adapter to use a "credential resolver" | Migration path from cookie-only to hybrid needs a clear precedence order |
| 6 | Health/status surfaces missing, connected, expired, and reconnect-required states without exposing secrets | `CheckHealth()` returns "offline" with "no session cookie configured" or "session cookie expired" | ✅ Reasonable — add granular states: `missing`, `connected`, `expired`, `reconnect_required` | Needs to ensure secrets (tokens, cookies) are stripped from error messages |
| 7 | Changes are covered by targeted tests and regression checks | Existing tests: `chatgpt/adapter_test.go` (3 tests), `config/parser_test.go` (6 tests), `router/router_test.go` (2 tests) | ✅ Valid — must extend test patterns | New subsystem needs comprehensive unit tests |

---

## Architecture Decision Summary

### Current Architecture Baseline

```
ollarouter/
├── internal/
│   ├── auth/
│   │   ├── api_key.go          # Downstream API key auth only
│   │   ├── api_key_test.go
│   │   ├── rate_limit.go
│   │   └── rate_limit_test.go
│   ├── config/
│   │   ├── config.go           # ProviderConfig has SessionCookie + UserAgent
│   │   ├── parser.go
│   │   └── parser_test.go
│   ├── provider/
│   │   ├── chatgpt/
│   │   │   ├── adapter.go      # Cookie-only auth, setHeaders() uses __Secure-next-auth.session-token
│   │   │   └── adapter_test.go
│   │   ├── factory.go          # Routes "chatgpt-plus" → chatgpt.NewAdapter()
│   │   └── registry.go
│   ├── router/
│   │   ├── router.go           # App struct has Auth, RateLimit but NOT UpstreamAuth
│   │   ├── admin_handlers.go   # Provider CRUD, settings, stats, logs (no auth lifecycle)
│   │   ├── handler_health.go   # /health and /health/providers
│   │   └── handler_chat.go     # Model resolution, streaming, Ollama + OpenAI adapters
│   └── types/
│       └── provider.go         # ProviderConfig, Provider interface, ProviderState
└── providers.yaml              # chatgpt-plus configured with session_cookie
```

### Proposed Changes

| Component | Change Type | Details |
|-----------|-------------|---------|
| `internal/config/config.go` | Extend | Add `AuthSettings` struct (via `Auth` field on `ProviderConfig`) |
| `internal/types/provider.go` | Extend | Mirror config types for runtime use |
| `internal/auth/upstream/` | **NEW** | Auth manager with PKCE, callback validation, credential persistence |
| `internal/auth/upstream/store.go` | **NEW** | Credential store interface + keychain + encrypted file implementations |
| `internal/provider/chatgpt/adapter.go` | Refactor | Add credential resolver; hybrid bearer-first, cookie-fallback |
| `internal/router/router.go` | Extend | Add `UpstreamAuth` to `App` struct; wire to admin routes |
| `internal/router/admin_handlers.go` | Extend | Add `GET /auth/status`, `POST /auth/connect`, `GET /auth/callback`, `POST /auth/disconnect` |
| `internal/router/handler_health.go` | Extend | Surface auth state per ChatGPT provider without secrets |

### Key Architectural Decisions (Pending ADR)

1. **New package location**: `internal/auth/upstream/` — keeps upstream auth concerns separate from downstream API key auth in `internal/auth/`.
2. **Credential store interface** — `CredentialStore` interface with `Get()`, `Set()`, `Delete()` methods, implemented by keychain and encrypted file backends.
3. **Encrypted file format** — TBD (recommend NaCl secretbox or AES-256-GCM with key derived from machine-specific seed).
4. **Auth mode field** — `ProviderConfig.Auth.AuthMode` will be `"oauth"` or `"session_cookie"` (or `"none"`).
5. **Health states** — Map to: `"missing"` (no credentials), `"connected"` (valid token), `"expired"` (token expired), `"reconnect_required"` (refresh failed).

---

## Implementation Readiness

### ✅ Ready for Implementation (no blocking gaps)

- Config type extensions (`AuthSettings` struct)
- Admin route additions (auth lifecycle endpoints)
- ChatGPT adapter refactoring (credential resolver pattern)
- Health status granularity improvement
- Test coverage extension

### ⚠️ Non-Blocking Gaps (clarify during implementation)

| Gap | Impact | Recommendation |
|-----|--------|----------------|
| Keychain library not specified | Implementation delay for credential store | Evaluate `zalando/go-keyring` (supports macOS, Windows, Linux) or use OS-specific build tags |
| Encryption algorithm not specified | Blocked for file fallback | Use NaCl `secretbox` (simple, audited) or AES-256-GCM with argon2id KDF |
| OAuth callback mechanism undefined | Blocked for auth flow design | Implement local callback server (localhost:callback_port) for MVP; remote webhook as enhancement |
| `AuthSettings` YAML shape undefined | Config parsing cannot start | Define shape as: `auth: { mode: "oauth", client_id: "...", redirect_uri: "...", callback_port: 11437, keychain_service: "ollarouter-chatgpt" }` |
| Credential status persistence location unknown | Health route cannot report accurately | Store status in encrypted file alongside credentials; update on connect/disconnect/expiry |

### Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Keychain integration breaks on Linux (no secret-service) | Medium | High | Detect at runtime; fall through to encrypted file immediately |
| OAuth flow complexity increases scope beyond plan | Medium | Medium | Split into MVP (config-only + encrypted file + session_cookie hybrid) and enhancement (full OAuth with PKCE) |
| Breaking changes to `providers.yaml` format | Low | High | Keep `session_cookie` field as fallback; add `auth` block as optional |
| Secrets leaked in health/logs | Low | High | Add explicit redaction in credential resolver and health handler |
| Circuit breaker interaction with auth state unknown | Medium | Low | Auth failures should trip circuit breaker independently from runtime errors |

---

## TDD Rule Confirmation

The plan states: "All new behavior must be introduced with failing tests first, then minimal implementation, then passing verification."

Confirmed. This maps to the existing test patterns in the codebase (`_test.go` files in each package) and will be enforced per the plan.

---

## Summary

The plan is **implementation-ready** with 5 non-blocking gaps that need clarification during implementation. The architecture is well-scoped for the existing OllaRouter codebase. The most significant new code is the `internal/auth/upstream/` package and the credential store subsystem. Recommend resolving the encryption and keychain choices before starting the credential store workstream.
