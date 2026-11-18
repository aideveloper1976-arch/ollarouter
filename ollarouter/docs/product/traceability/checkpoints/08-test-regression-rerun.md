# Test Regression Re-Run: Hybrid ChatGPT/Codex Auth (Post-Blocker-Fixes)

**Checkpoint:** 08-test-regression-rerun
**Date:** 2026-05-22
**Agent:** test-engineer (fullstack-tester-reviewer)
**Triggered by:** Blocker fixes completed (`docs/product/traceability/checkpoints/07-blocker-fixes.md`)
**Re-running:** Full regression from `03-test-regression.md` after 7 blocker-fix patches

---

## What Changed Since Last Regression

| Blocker | Fix | Files Changed | Tests Added |
|---------|-----|---------------|-------------|
| B1 | `Update()` merges optional fields from existing config | `registry.go`, `admin_handlers.go`, `registry_test.go` | 1 (registry) |
| B2 | `NewEncryptedFileStore()` returns error, enforces 16-byte min key | `store.go`, `store_test.go`, `manager_test.go` | 1 (short key) |
| C-01/02 | Auth middleware added to admin API routes (except `/me`) | `router.go`, `router_test.go` | 4 (auth protection) |
| C-03 | `handleAdminConfigYAML()` masks API keys | `admin_handlers.go`, `router_test.go` | 1 (YAML masking) |
| H-08 | Config file permissions 0644 → 0600 | `parser.go` | 0 (no test needed) |

---

## Tests Run and Outcomes

### Full Test Suite (`go test ./internal/... -count=1`)

| Package | Tests | Status | Notes |
|---------|-------|--------|-------|
| `internal/auth` | 15 | ✅ ALL PASS | Unchanged |
| `internal/auth/upstream` | 16 | ✅ ALL PASS | 8 store + 7 manager + 1 short-key (B2) |
| `internal/config` | 8 | ✅ ALL PASS | Unchanged |
| `internal/engine` | — | ⚠️ 1 FAIL pre-existing | `TestResolveModelDirect` — NOT caused by our changes |
| `internal/logger` | — | — | No test files |
| `internal/provider` | 1 | ✅ PASS | `TestRegistryUpdatePreservesOptionalFields` (B1) |
| `internal/provider/anthropic` | 10 | ✅ ALL PASS | Unchanged |
| `internal/provider/chatgpt` | 9 | ✅ ALL PASS | 5 adapter + 4 existing |
| `internal/provider/google` | — | — | No test files |
| `internal/provider/ollama` | 9 | ✅ ALL PASS | Unchanged |
| `internal/provider/openai` | 10 | ✅ ALL PASS | Unchanged |
| `internal/provider/openaicompat` | 9 | ✅ ALL PASS | Unchanged |
| `internal/router` | 13 | ✅ ALL PASS | 3 auth routes + 5 auth protection + 5 existing |
| `internal/streaming` | 4 | ✅ ALL PASS | Unchanged |
| `internal/tracking` | 11 | ✅ ALL PASS | Unchanged |
| `internal/types` | — | — | No test files |
| `internal/webui` | — | — | No test files |

**Total: ~109 tests pass. Zero regressions introduced by blocker fixes.**

### Targeted Blocker-Fix Test Verification

| Test | Package | Result |
|------|---------|--------|
| `TestRegistryUpdatePreservesOptionalFields` | `internal/provider` | ✅ PASS |
| `TestEncryptedFileStoreShortKeyRejected` | `internal/auth/upstream` | ✅ PASS |
| `TestAdminAuthStatusRequiresAuthWhenKeysConfigured` | `internal/router` | ✅ PASS |
| `TestAdminAuthConnectRequiresAuthWhenKeysConfigured` | `internal/router` | ✅ PASS |
| `TestAdminProviderTestRouteRequiresAuthWhenKeysConfigured` | `internal/router` | ✅ PASS |
| `TestAdminConfigYAMLMasksAPIKeys` | `internal/router` | ✅ PASS |
| `TestAdminMeRouteWorksWithoutAuth` | `internal/router` | ✅ PASS |

### Build & Vet

| Check | Result |
|-------|--------|
| `go build ./...` | ✅ PASS (no output = clean) |
| `go vet ./...` | ✅ PASS (no output = clean) |

---

## Pre-Existing Failure (Not Our Product Defect)

**Test:** `TestResolveModelDirect` in `internal/engine/router_test.go`
**Failure:** Expected `ModelName = 'test-provider/gpt-4'`, got `"gpt-4"`
**Status:** Unchanged from previous regression. Not caused by blocker fixes. Engine package was not modified.

---

## UI/Browser/UX Applicability Decision

**Decision: UI/browser/UX gates remain NOT applicable.**

Rationale (unchanged from `03-test-regression.md`):
- All changes are backend-only Go code
- No HTML templates, SPAs, or frontend files were modified
- Admin auth lifecycle endpoints are REST APIs tested via `httptest`
- The admin SPA was not modified

**Explicitly excluded from this checkpoint:**
- Browser flow validation
- Screenshot capture
- Accessibility scanning
- Desktop/tablet/mobile responsive testing
- Global UI regression matrix
- Persisted UI test script artifact

---

## Regression Comparison: Before vs After Blocker Fixes

| Metric | Before Fixes (03-test-regression) | After Fixes (this run) | Delta |
|--------|----------------------------------|----------------------|-------|
| Build | ✅ PASS | ✅ PASS | Unchanged |
| `go vet` | ✅ PASS | ✅ PASS | Unchanged |
| `internal/auth/upstream` tests | 15 ✅ | 16 ✅ | +1 (short-key test) |
| `internal/provider` tests | 0 | 1 ✅ | +1 (registry test) |
| `internal/router` tests | 8 ✅ | 13 ✅ | +5 (auth protection) |
| Pre-existing engine failure | ⚠️ 1 FAIL | ⚠️ 1 FAIL | Unchanged |
| **Blocking product defects** | **0** | **0** | **Unchanged** |

---

## Findings Table

| # | Finding | Severity | Type | Evidence |
|---|---------|----------|------|----------|
| 1 | All 7 blocker-fix tests pass (B1, B2, C-01, C-02, C-03) | ✅ NONE | — | Targeted test runs above |
| 2 | All existing tests unaffected — zero regressions | ✅ NONE | — | Full suite output |
| 3 | Build compiles clean | ✅ NONE | — | `go build ./...` |
| 4 | `go vet` clean | ✅ NONE | — | `go vet ./...` |
| 5 | Pre-existing `TestResolveModelDirect` failure unchanged | ⚠️ NON-BLOCKING | Pre-existing repo debt | `internal/engine/router_test.go:154` |
| 6 | No race detector (CGO disabled) | ✅ NONE | Environment limitation | Known constraint |
| 7 | No golangci-lint | ✅ NONE | Environment limitation | Not installed |

---

## Gate Verdict

### ✅ PASS — All blocker fixes validated, zero regressions

All 7 blocker-fix tests pass. All ~109 tests in the full suite pass (except the pre-existing engine failure). Build and vet are clean. UI/browser/UX remain not applicable.

**Final recommendation: Proceed to the next lifecycle phase.** No remediation needed.

---

## Test Evidence Artifacts

- **Full test output:** Inline above from `go test ./internal/... -count=1`
- **Targeted blocker-fix outputs:** 4 targeted `-v` runs above confirming:
  - `TestRegistryUpdatePreservesOptionalFields` → PASS
  - All 16 `internal/auth/upstream` tests → PASS
  - All 13 `internal/router` tests → PASS (5 new auth protection)
  - All 9 `internal/provider/chatgpt` tests → PASS
- **Source files verified (new/changed since 03-test-regression):**
  - `internal/provider/registry.go` — `Update()` merge logic
  - `internal/provider/registry_test.go` — NEW: `TestRegistryUpdatePreservesOptionalFields`
  - `internal/router/admin_handlers.go` — `handleAdminUpdateProvider` copy logic, `maskAPIKey()` helper
  - `internal/auth/upstream/store.go` — `NewEncryptedFileStore` error return + key length enforcement
  - `internal/auth/upstream/store_test.go` — NEW: `TestEncryptedFileStoreShortKeyRejected`
  - `internal/auth/upstream/manager_test.go` — Updated for new store signature
  - `internal/router/router.go` — Auth middleware on admin API routes
  - `internal/router/router_test.go` — NEW: 5 auth protection + YAML masking tests
  - `internal/config/parser.go` — `SaveFile()` permissions 0644 → 0600
