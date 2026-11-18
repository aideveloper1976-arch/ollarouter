# Hybrid ChatGPT Auth Implementation Plan

> Goal: add a hybrid upstream ChatGPT/Codex auth system for OllaRouter with local callback, remote callback, keychain-first credential storage, encrypted file fallback, admin auth lifecycle endpoints, and legacy session cookie compatibility.

## Acceptance Criteria

- Provider config supports structured auth settings for hybrid upstream auth.
- Upstream auth subsystem supports auth sessions, callback validation, and persisted credential status.
- Credential storage prefers OS keychain and falls back to encrypted file storage.
- Admin routes exist for auth status, connect, callback, and disconnect.
- ChatGPT provider can use stored bearer credentials and still fall back to legacy `session_cookie` mode.
- Health/status surfaces missing, connected, expired, and reconnect-required states without exposing secrets.
- Changes are covered by targeted tests and regression checks.

## Planned Workstreams

1. Config and type model updates in `internal/config` and `internal/auth/upstream`.
2. Credential store implementations for keychain and encrypted file fallback.
3. Upstream auth manager for state, PKCE, callback validation, and credential persistence.
4. Admin router/handler lifecycle endpoints for connect, callback, status, and disconnect.
5. ChatGPT adapter migration to credential resolver with legacy cookie fallback.
6. Verification gates: self-verification, regression tests, code review, security review, traceability update.

## TDD Rule

All new behavior must be introduced with failing tests first, then minimal implementation, then passing verification.
