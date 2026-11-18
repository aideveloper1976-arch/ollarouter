# UI Test Script: OllaRouter Admin Browser UI Regression

**Script ID:** TS-OLLAROUTER-001
**Test Date:** 2026-05-20
**Phase:** Phase 11.1 — Browser UI Regression (Rerun after UX fixes)
**Service URL:** http://localhost:11435
**Default API Key:** `or_admin_default_key_change_me`

---

## Objective
Validate the OllaRouter Admin UI SPA renders correctly, auth is enforced, health endpoints are public, accessibility meets standards, color contrast passes AA, and the UI is responsive across viewports.

---

## Prerequisites
- OllaRouter service running on port 11435 (`docker compose up -d ollarouter`)
- Service must be healthy (`curl http://localhost:11435/health` returns 200)
- Browser with JavaScript enabled (Chromium/Playwright recommended)

---

## Test 1: Login Page Loads (Not Raw JSON)

| Step | Action | Expected Result | Evidence |
|------|--------|----------------|----------|
| 1.1 | Navigate to `http://localhost:11435/admin/` | HTTP 200, Content-Type: `text/html` | curl: `-D -` shows headers |
| 1.2 | Verify page contains `<!DOCTYPE html>` | HTML document structure present | Screenshot |
| 1.3 | Verify heading "OllaRouter Admin" exists | `<h1>` with text "OllaRouter Admin" | Screenshot |
| 1.4 | Verify API key input field `#or-token-input` | Visible, placeholder "Enter API key..." | Screenshot |
| 1.5 | Verify Login button `#or-login-btn` | Visible, text "Login" | Screenshot |
| 1.6 | **Negative test:** Navigate to `/admin/` without auth | **Must NOT return raw JSON 401** | Ensure HTML, not `{"error":"authentication required"}` |

## Test 2: Login Flow

| Step | Action | Expected Result | Evidence |
|------|--------|----------------|----------|
| 2.1 | Navigate to `/admin/` | Login overlay rendered | Screenshot |
| 2.2 | Type `or_admin_default_key_change_me` into `#or-token-input` | Input accepted | — |
| 2.3 | Click `#or-login-btn` | Button text changes to "Verifying..." | — |
| 2.4 | Wait 3-5 seconds | Page reloads (auth verified successfully) | — |
| 2.5 | Verify `#root` is visible after reload | SPA rendered | Screenshot |
| 2.6 | **Negative test:** Type empty string | Error "Please enter an API key" displayed | — |
| 2.7 | **Negative test:** Type `invalid_token` | Error "Invalid API key" displayed | — |

## Test 3: SPA Navigation Pages

| Step | Action | Expected Result | Evidence |
|------|--------|----------------|----------|
| 3.1 | After login, check page body text | Contains "Dashboard" | Assert text |
| 3.2 | Check page body text | Contains "Providers" | Assert text |
| 3.3 | Check page body text | Contains "API Keys" | Assert text |
| 3.4 | Check page body text | Contains "Models" | Assert text |
| 3.5 | Check page body text | Contains "Settings" | Assert text |
| 3.6 | Check page body text | Contains "OllaRouter" | Assert text |

## Test 4: Auth Enforcement

| Step | Action | Expected Result | Evidence |
|------|--------|----------------|----------|
| 4.1 | `GET /admin/api/providers` without auth header | HTTP 401, `{"error":"authentication required"}` | curl |
| 4.2 | `GET /admin/api/providers` with valid Bearer token | HTTP 200, JSON with providers list | curl |
| 4.3 | `GET /admin/api/providers` with invalid Bearer token | HTTP 401, `{"error":"invalid API key"}` | curl |
| 4.4 | `GET /admin/api/stats/summary` without auth | HTTP 401 | curl |
| 4.5 | `GET /admin/api/stats/summary` with valid token | HTTP 200 | curl |
| 4.6 | `GET /admin/api/settings` without auth | HTTP 401 | curl |
| 4.7 | `GET /admin/api/settings` with valid token | HTTP 200 | curl |
| 4.8 | `GET /admin/api/api-keys` without auth | HTTP 401 | curl |
| 4.9 | `GET /admin/api/api-keys` with valid token | HTTP 200 | curl |
| 4.10 | `GET /api/version` without auth | HTTP 401 | curl |
| 4.11 | `GET /api/tags` without auth | HTTP 401 | curl |

## Test 5: Public Health Endpoints

| Step | Action | Expected Result | Evidence |
|------|--------|----------------|----------|
| 5.1 | `GET /health` without auth | HTTP 200, JSON with status/version/uptime | curl |
| 5.2 | `GET /health/providers` without auth | HTTP 200, JSON with provider details | curl |

## Test 6: Accessibility

| Step | Action | Expected Result | Evidence |
|------|--------|----------------|----------|
| 6.1 | Run axe-core scan on `/admin/` | No critical or serious violations | Scan report |
| 6.2 | Check for landmark region violations | ≤1 moderate violation acceptable | — |
| 6.3 | Verify 30+ checks pass | Minimum 30 passing checks | — |

## Test 7: Color Contrast

| Step | Element | Ratio Requirement | Measurement |
|------|---------|-------------------|-------------|
| 7.1 | Subtitle text (#94A3B8 on #0F172A) | WCAG AA ≥ 4.5:1 | 6.96:1 ✅ |
| 7.2 | Input text (#F1F5F9 on #0F172A) | WCAG AA ≥ 4.5:1 | 16.3:1 ✅ |
| 7.3 | Heading (#38BDF8 on #0F172A) | WCAG AA ≥ 3:1 (large text) | 8.33:1 ✅ |
| 7.4 | Button text (#FFFFFF on #075985) | WCAG AA ≥ 4.5:1 | 7.56:1 ✅ |
| 7.5 | Verify CSS overrides injected in bootstrap | bg-sky-600, hover:bg-sky-500, text-slate-500 | ✅ |

## Test 8: Responsive Behavior

| Step | Viewport | Action | Expected Result | Evidence |
|------|----------|--------|----------------|----------|
| 8.1 | Desktop (1280px) | Load `/admin/` | Full login form centered | Screenshot |
| 8.2 | Tablet (768px) | Load `/admin/` | Login form ≤92vw, centered | Screenshot |
| 8.3 | Mobile (375px) | Load `/admin/` | Login form ≤92vw, no horizontal scroll | Screenshot |

---

## Evidence Linkage

All screenshots saved to `docs/product/evidence/tests/`:
- `ollarouter-admin-login-page-*.png` — Login page
- `ollarouter-dashboard-after-login-*.png` — Dashboard after login
- `ollarouter-dashboard-desktop-*.png` — Desktop view
- `ollarouter-login-mobile-375px-*.png` — Mobile view
- `ollarouter-login-tablet-768px-*.png` — Tablet view
- Step screenshots: `step-*-*.png`

Full evidence record: `docs/product/evidence/ACP-UI-REG-001-OllaRouter_Phase_11_Browser_UI_Regression_Rerun.json`

Checkpoint: `docs/product/traceability/checkpoints/11.1-browser-ui-regression-rerun.md`

---

## Results Summary

| Test | Status |
|------|--------|
| 1. Login Page Loads (not raw JSON) | ✅ PASS |
| 2. Login Flow | ✅ PASS |
| 3. SPA Navigation Pages | ✅ PASS |
| 4. Auth Enforcement | ✅ PASS |
| 5. Public Health Endpoints | ✅ PASS |
| 6. Accessibility | ✅ PASS (1 moderate) |
| 7. Color Contrast | ✅ PASS (AA/AAA) |
| 8. Responsive Behavior | ✅ PASS |
| **Overall** | **✅ PASS** |
