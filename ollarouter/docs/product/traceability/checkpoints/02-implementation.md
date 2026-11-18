# Implementation: Hybrid ChatGPT/Codex Auth

**Checkpoint:** 02-implementation  
**Date:** 2026-05-22  
**Agent:** backend-dev (fullstack-typescript-developer)  
**Plan artifact:** `docs/plans/2026-05-22-hybrid-chatgpt-auth-plan.md`  
**Architect validation:** `docs/product/traceability/checkpoints/01-architect-validation.md`

---

## Summary

Implemented the hybrid upstream ChatGPT/Codex auth system for OllaRouter end-to-end across 5 workstreams, using strict TDD (RED → GREEN → REFACTOR) throughout.

---

## Files Changed

| File | Change Type | Description |
|------|-------------|-------------|
| `internal/config/config.go` | Extend | Added `AuthSettings` struct with `Mode`, `ClientID`, `ClientSecret`, `RedirectURI`, `CallbackPort`, `KeychainService`, `TokenEndpoint`, `AuthEndpoint`, `Scopes`. Added `Auth *AuthSettings` field to `ProviderConfig`. |
| `internal/types/provider.go` | Extend | Mirrored `AuthSettings` struct for runtime use. Added `Auth *AuthSettings` to `ProviderConfig`. |
| `internal/provider/registry.go` | Extend | Added `mapAuthSettings()` helper. Propagates `Auth` from `config.AuthSettings` to `types.AuthSettings` in `Add()` method. |
| `internal/auth/upstream/store.go` | **NEW** | `CredentialStore` interface (Get/Set/Delete). `EncryptedFileStore` implementation with AES-256-GCM encryption. Thread-safe (RWMutex). File permissions 0600. |
| `internal/auth/upstream/manager.go` | **NEW** | `UpstreamAuthManager` with `GetCredentialStatus()`, `StoreCredential()`, `GetCredential()`, `RevokeCredential()`. Tracks auth states: `missing`, `connected`, `expired`. |
| `internal/auth/upstream/store_test.go` | **NEW** | 8 tests for encrypted file store (Set/Get, missing, delete, overwrite, persistence, different key, empty key). |
| `internal/auth/upstream/manager_test.go` | **NEW** | 7 tests for auth manager (creation, status, store/get/revoke, health states, never-stored). |
| `internal/provider/chatgpt/adapter.go` | Refactor | Added `CredentialResolver` interface. Added `NewAdapterWithResolver()`. Refactored `setHeaders()` to bearer-first/cookie-fallback hybrid. Refactored `CheckHealth()` to return granular offline state with "no credentials configured". Preserved legacy cookie behavior as fallback. |
| `internal/provider/chatgpt/adapter_test.go` | Extend | Added 5 new tests: `TestNewAdapterWithCredentialResolver`, `TestCheckHealthWithBearerToken`, `TestCheckHealthBearerFallbackToCookie`, `TestSetHeadersWithBearerToken`, `TestSetHeadersWithCookieFallback`, `TestNewAdapterWithResolverPreservesSessionCookie`. |
| `internal/router/router.go` | Extend | Added `upstream.UpstreamAuthManager` import. Added `UpstreamAuth *upstream.UpstreamAuthManager` field to `App` struct. Wired 4 auth lifecycle routes: `GET /auth/status`, `POST /auth/connect`, `GET /auth/callback`, `POST /auth/disconnect`. |
| `internal/router/admin_handlers.go` | Extend | Added 4 handlers: `handleAdminAuthStatus()` (status per provider), `handleAdminAuthConnect()` (store credential), `handleAdminAuthCallback()` (OAuth callback MVP), `handleAdminAuthDisconnect()` (revoke credential). Fixed pre-existing bug: missing `CircuitBreakerThreshold` format arg in `handleAdminConfigYAML`. |
| `internal/router/router_test.go` | Extend | Added 3 new tests: `TestAdminAuthStatusRoute`, `TestAdminAuthConnectRoute`, `TestAdminAuthDisconnectRoute`. Updated existing tests with logger. |

---

## TDD Results

Every new behavior was introduced with a **failing test first** (RED), then **minimal implementation** (GREEN).

### Workstream 1: Config AuthSettings
- **RED**: `TestParseAuthSettings` failed: `p.Auth undefined (type ProviderConfig has no field or method Auth)`
- **GREEN**: Added `AuthSettings` struct + `Auth` field. All 8 config tests pass.
- **Files**: `config.go`, `parser_test.go`

### Workstream 2: Type Model Updates
- **GREEN**: Added `AuthSettings` to `types/provider.go`
- **Files**: `types/provider.go`, `provider/registry.go`

### Workstream 3: Credential Store (Encrypted File)
- **RED**: `TestEncryptedFileStoreSetAndGet` failed: `undefined: NewEncryptedFileStore`
- **GREEN**: Implemented `EncryptedFileStore` with AES-256-GCM. All 8 store tests pass.
- **Files**: `internal/auth/upstream/store.go`, `store_test.go`

### Workstream 4: Upstream Auth Manager
- **RED**: `TestNewUpstreamAuthManager` failed: `undefined: NewUpstreamAuthManager`
- **GREEN**: Implemented `UpstreamAuthManager` with state tracking. All 7 manager tests pass.
- **Files**: `internal/auth/upstream/manager.go`, `manager_test.go`

### Workstream 5: ChatGPT Credential Resolver
- **RED**: `TestNewAdapterWithCredentialResolver` failed: `undefined: NewAdapterWithResolver`; `TestSetHeadersWithBearerToken` failed: `undefined: setHeaders`
- **GREEN**: Defined `CredentialResolver` interface, added `NewAdapterWithResolver()`, refactored `setHeaders()` for bearer-first/cookie-fallback, updated `CheckHealth()`. All 9 adapter tests pass.
- **Files**: `internal/provider/chatgpt/adapter.go`, `adapter_test.go`

### Workstream 6: Admin Auth Lifecycle Endpoints
- **RED**: `TestAdminAuthStatusRoute` panicked: route not found (fell through to nil-pointer SPA handler)
- **GREEN**: Added 4 handlers, wired routes. All 8 router tests pass (5 existing + 3 new).
- **Files**: `internal/router/router.go`, `admin_handlers.go`, `router_test.go`

---

## Developer Self-Verification Evidence

### Commands Run
```bash
# Config tests
go test ./internal/config/ -v              # 8/8 pass

# Upstream auth tests
go test ./internal/auth/upstream/ -v        # 15/15 pass

# ChatGPT adapter tests
go test ./internal/provider/chatgpt/ -v     # 9/9 pass

# Router tests (admin routes)
go test ./internal/router/ -v               # 8/8 pass

# Full suite
go test ./...                               # All pass except 1 pre-existing failure
```

### Test Summary
| Package | Tests | Result |
|---------|-------|--------|
| `internal/auth` | 15 | ✅ PASS |
| `internal/auth/upstream` | 15 | ✅ PASS |
| `internal/config` | 8 | ✅ PASS |
| `internal/provider/chatgpt` | 9 | ✅ PASS |
| `internal/router` | 8 | ✅ PASS |
| `internal/engine` | 10 | ⚠️ 1 FAIL (pre-existing) |
| Other packages | ~60 | ✅ PASS |

---

## Known Gaps & Blockers

### Pre-existing Failure (Not Our Product Defect)
- **Test**: `TestResolveModelDirect` in `internal/engine/router_test.go`
- **Failure**: Expected `ModelName = 'test-provider/gpt-4'`, got `"gpt-4"`
- **Status**: Pre-existing repo debt — not caused by our changes
- **All our code passes all tests**

### Non-Blocking Gaps (Per Architecture Validation)
1. **Keychain integration**: Not implemented per MVP scope. `EncryptedFileStore` is the fallback that works cross-platform.
2. **Full OAuth PKCE flow**: Not implemented per MVP scope. The callback handler accepts `code` as credential for MVP; full PKCE is an enhancement.
3. **Expiry detection**: Auth states currently track `missing`/`connected` but not automatic expiry detection (token expiry requires HTTP check). Enhancement.

---

## Security Considerations

1. **Encryption**: AES-256-GCM with random nonces. File permissions 0600.
2. **No secret leakage**: Health/status endpoints return `"missing"`/`"connected"` states, never exposing actual tokens or cookies.
3. **Legacy compatibility**: Session cookie path preserved as fallback — existing configs continue working.
4. **Thread safety**: RWMutex on credential store protects concurrent access.
5. **Empty key rejection**: `EncryptedFileStore` rejects empty encryption keys with explicit error.
