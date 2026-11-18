# Phase 4 — UX Design (Pre-Build): OllaRouter

**Date:** 2026-05-20  
**Agent:** product-analyst  
**Project:** ollama-stack (C:\Projects\ollama-stack)  
**Phase 1 Reference:** `docs/product/traceability/checkpoints/01-market-research.md`  
**Phase 2 Reference:** `docs/product/traceability/checkpoints/02-competitor-analysis.md`  
**Phase 3 Reference:** `docs/product/traceability/checkpoints/03-usp-definition.md`  
**Design System Source:** ux-intelligence MCP — SaaS Platform, Professional mood, Data-Dense Dashboard style

---

## Table of Contents

1. [Design System Foundation](#1-design-system-foundation)
2. [UI Information Architecture](#2-ui-information-architecture)
3. [Screen Inventory & Layouts](#3-screen-inventory--layouts)
4. [Component Hierarchy](#4-component-hierarchy)
5. [User Flows](#5-user-flows)
    - 5.1 Initial Deployment & First Request
    - 5.2 Adding a Provider via Web UI
    - 5.3 Viewing the Stats Dashboard
    - 5.4 Fallback Configuration & Scenario
    - 5.5 ChatGPT Plus Provider Setup
    - 5.6 API Key Management
6. [Provider Setup UX](#6-provider-setup-ux)
7. [Token/Cost Stats Dashboard UX](#7-token-cost-stats-dashboard-ux)
8. [Model & Fallback Configuration UX](#8-model--fallback-configuration-ux)
9. [Accessibility Plan](#9-accessibility-plan)
10. [Responsive Behavior](#10-responsive-behavior)
11. [UX Acceptance Criteria](#11-ux-acceptance-criteria)
12. [Traceability Links](#12-traceability-links)

---

## 1. Design System Foundation

### 1.1 Design Style

| Attribute | Value |
|-----------|-------|
| **Primary Style** | Data-Dense Dashboard (dashboard pages) + Minimal & Direct (configuration pages) |
| **Pattern** | Minimal & Direct — simple, clear messaging, fast loading |
| **Best For** | Complex data analysis (dashboard), Simple configuration (admin) |
| **Anti-Patterns to Avoid** | Information overload, no clear hierarchy, too sparse, missing information |

### 1.2 Color Palette

| Role | Hex | Usage |
|------|-----|-------|
| **Primary** | `#2563EB` | Primary actions, links, active navigation |
| **Secondary** | `#64748B` | Secondary UI elements, labels, secondary text |
| **CTA / Success** | `#059669` / `#10B981` | Call-to-action buttons, success indicators, online status |
| **Background** | `#F8FAFC` | Page backgrounds, card surfaces |
| **Text** | `#1E293B` | Body text, headings |
| **Accent** | `#7C3AED` | Selected states, highlights, provider badges |
| **Error / Danger** | `#EF4444` | Error states, offline indicators, delete actions |
| **Warning** | `#F59E0B` | Degraded status, rate-limit warnings |
| **Info** | `#3B82F6` | Information banners, tooltips |

### 1.3 Typography

| Attribute | Value |
|-----------|-------|
| **Font Family** | Plus Jakarta Sans (Google Fonts) |
| **Import URL** | `https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700&display=swap` |
| **Scale** | 12px (caption), 14px (small), 16px (body), 18px (lead), 24px (h3), 30px (h2), 36px (h1), 48px (hero) |
| **Weights** | 400 (Regular), 500 (Medium), 600 (Semi-Bold), 700 (Bold) |

### 1.4 Iconography

- **Icon Set:** Lucide Icons (open-source, consistent with shadcn/ui ecosystem)
- **Format:** SVG inline (no emoji as icons)
- **Sizes:** 16px (inline), 20px (small action), 24px (navigation), 32px (status indicators)

### 1.5 Spacing & Layout

| Token | Value | Usage |
|-------|-------|-------|
| `--space-xs` | 4px | Dense table cells |
| `--space-sm` | 8px | Tight spacing |
| `--space-md` | 16px | Default spacing |
| `--space-lg` | 24px | Section spacing |
| `--space-xl` | 32px | Page section gaps |
| `--space-2xl` | 48px | Major section margins |
| **Max content width** | 1280px | Dashboard and admin pages |
| **Sidebar width** | 260px (collapsible to 64px) | Navigation |

### 1.6 Shadows & Elevation

| Level | Shadow | Usage |
|-------|--------|-------|
| 1 | `0 1px 2px rgba(0,0,0,0.05)` | Cards, subtle depth |
| 2 | `0 4px 6px -1px rgba(0,0,0,0.1)` | Dropdowns, modals |
| 3 | `0 10px 15px -3px rgba(0,0,0,0.1)` | Dialogs, slide-overs |

### 1.7 Border Radius

| Token | Value | Usage |
|-------|-------|-------|
| `--radius-sm` | 4px | Inputs, small UI |
| `--radius-md` | 6px | Cards, buttons |
| `--radius-lg` | 8px | Modals, dialogs |
| `--radius-xl` | 12px | Large containers |

### 1.8 Animation

| Property | Duration | Easing |
|----------|----------|--------|
| Hover transitions | 150ms | ease-in-out |
| Modal/dialog enter | 200ms | ease-out |
| Page transitions | 200ms | ease-in-out |
| Data refresh | 300ms | ease |
| Spinner/loading | 800ms loop | linear |
| `prefers-reduced-motion` | Respect all animations | disabled |

---

## 2. UI Information Architecture

### 2.1 Site Map

```
OllaRouter Admin UI (port 11436)
├── /
│   └── Dashboard Landing / Overview
│       ├── Summary Cards Row
│       │   ├── Total Cost (month-to-date)
│       │   ├── Total Requests
│       │   ├── Active Providers
│       │   └── Avg. Latency
│       ├── Cost Trend Chart (last 7/30/90 days)
│       └── Recent Requests Table (last 20 requests)
│
├── /providers
│   ├── Provider List (table: name, type, status, models count, last check)
│   │   ├── Status badges: Online (green) / Degraded (yellow) / Offline (red) / Unknown (grey)
│   │   └── Row actions: Edit, Delete, Test Connection
│   ├── /providers/add (multi-step wizard or inline form)
│   │   ├── Step 1: Provider Type selection (OpenAI, Anthropic, Google, Ollama, ChatGPT Plus, OpenAI-Compatible)
│   │   ├── Step 2: Connection Details (API key, base URL, organization ID — type-specific)
│   │   └── Step 3: Model Configuration (model names, mapping overrides, cost multipliers)
│   │   └── Step 4: Fallback Configuration (fallback chain order, retry policy)
│   └── /providers/:id/edit (pre-filled form, same structure as add)
│
├── /models
│   ├── Model Catalog (searchable, filterable grid/table)
│   │   ├── Columns: Model Name, Provider Badge, Status, Cost/token
│   │   ├── Actions: Quick test, Copy model name
│   │   └── Filter bar: by provider, by status, text search
│   └── /models/:id (detail panel / slide-over)
│       ├── Model details: full name, provider, pricing, context window
│       ├── Usage stats: requests, tokens, cost (time-windowed)
│       └── Fallback chain visualization for this model
│
├── /dashboard (detailed analytics)
│   ├── Time range selector (24h, 7d, 30d, 90d, custom)
│   ├── Cost Breakdown Tab
│   │   ├── Per-provider bar chart
│   │   ├── Per-model pie/treemap chart
│   │   └── Cost trend line chart
│   ├── Token Usage Tab
│   │   ├── Input vs Output stacked area chart
│   │   ├── Per-model token consumption
│   │   └── Daily token volume trend
│   ├── Latency Tab
│   │   ├── P50/P95/P99 latency line chart per provider
│   │   └── Request duration histogram
│   └── Requests Log Tab
│       ├── Infinite-scroll table: timestamp, model, provider, status, latency, tokens, cost
│       ├── Filterable by provider, model, status code, date range
│       └── Expandable row for full request/response details
│
├── /keys
│   ├── API Key List (table: name, masked key, status, usage this month, rate limit, last used)
│   │   └── Row actions: Copy, Revoke, Edit limits, Delete
│   ├── /keys/add (form: key name, rate limit, budget cap, expiration)
│   └── /keys/:id (detail: usage over time, revocation history)
│
├── /settings
│   ├── General Settings
│   │   ├── Admin UI port, Proxy port
│   │   ├── Log level, log format
│   │   ├── Prometheus metrics toggle
│   │   └── Usage data persistence (in-memory vs file)
│   ├── Provider Defaults
│   │   ├── Default retry count, backoff strategy
│   │   ├── Default timeout per provider type
│   │   └── Circuit breaker thresholds
│   └── YAML Config Preview (read-only view of current runtime config)
│
└── /health
    ├── Provider Health Status
    │   ├── Grid of status cards (one per provider)
    │   │   ├── Status dot + label
    │   │   ├── Last checked timestamp
    │   │   ├── Health check latency
    │   │   └── Circuit breaker state (Closed/Open/Half-Open)
    │   └── Auto-refresh toggle (every 30s)
    └── System Health
        ├── Uptime, memory usage, request count
        └── Version info
```

### 2.2 Navigation Structure

```
┌─────────────────────────────────────────────────────────────────┐
│  [Logo] OllaRouter                    [v1.0.0]  [?]  [⚙️]     │  ← Top Bar
├──────────┬──────────────────────────────────────────────────────┤
│          │                                                      │
│  📊      │  ← Sidebar (collapsible)                             │
│  Overview│                                                      │
│          │    Main Content Area                                  │
│  🔌      │                                                      │
│  Providers│                                                      │
│          │                                                      │
│  🤖      │                                                      │
│  Models  │                                                      │
│          │                                                      │
│  📈      │                                                      │
│  Dashboard│                                                      │
│          │                                                      │
│  🔑      │                                                      │
│  API Keys│                                                      │
│          │                                                      │
│  ❤️      │                                                      │
│  Health  │                                                      │
│          │                                                      │
├──────────┤                                                      │
│  🛠️      │                                                      │
│  Settings│                                                      │
└──────────┴──────────────────────────────────────────────────────┘
```

### 2.3 Page-to-EPIC Mapping

| Page / Route | EPIC | Priority | UI Complexity |
|-------------|------|----------|---------------|
| `/` (Overview Dashboard) | EPIC-4 (Cost Tracking & Dashboard) | P1 | Medium |
| `/providers` (Provider List) | EPIC-2 (Provider Configuration UI) | P1 | Low |
| `/providers/add` (Add Provider) | EPIC-2 | P1 | High |
| `/providers/:id/edit` (Edit Provider) | EPIC-2 | P1 | Medium |
| `/models` (Model Catalog) | EPIC-7 (Model Catalog & Extended API) | P2 | Medium |
| `/dashboard` (Detailed Analytics) | EPIC-4 | P1 | High |
| `/keys` (API Key List) | EPIC-3 (Auth & Security) | P1 | Low |
| `/keys/add` (Create API Key) | EPIC-3 | P1 | Low |
| `/health` (Provider Status) | EPIC-2 / EPIC-6 | P1 | Low |
| `/settings` (Configuration) | EPIC-2 / EPIC-6 | P2 | Medium |

---

## 3. Screen Inventory & Layouts

### 3.1 Screen 1: Overview Dashboard (`/`)

```
┌──────────────────────────────────────────────────────────────────────┐
│ [OllaRouter Logo]                              [v1.0.0] [?] [⚙️]   │
├───────┬──────────────────────────────────────────────────────────────┤
│       │  Overview                                                     │
│ Side  │                                                               │
│ bar   │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐        │
│       │  │$ 234.50  │ │  1,847   │ │    3     │ │  342ms   │        │
│       │  │Total Cost│ │ Requests │ │Providers │ │Avg Latency│        │
│       │  │ this mo  │ │ today    │ │ active   │ │          │        │
│       │  └──────────┘ └──────────┘ └──────────┘ └──────────┘        │
│       │                                                               │
│       │  ┌────────────────────────────────────────────────────┐       │
│       │  │  Cost Trend (Last 7 Days)                    [7d▼] │       │
│       │  │  ┌──────────────────────────────────────────────┐ │       │
│       │  │  │   📈 Line chart — daily spend               │ │       │
│       │  │  │   (OpenAI=blue, Anthropic=purple,           │ │       │
│       │  │  │    Ollama=green, ChatGPT=amber)             │ │       │
│       │  │  └──────────────────────────────────────────────┘ │       │
│       │  └────────────────────────────────────────────────────┘       │
│       │                                                               │
│       │  ┌────────────────────────────────────────────────────┐       │
│       │  │  Recent Requests                    [View All →]   │       │
│       │  │  ┌────┬────────────┬──────────┬──────┬──────┬────┐ │       │
│       │  │  │Time│ Model      │Provider   │Status│ Lat. │Cost│ │       │
│       │  │  ├────┼────────────┼──────────┼──────┼──────┼────┤ │       │
│       │  │  │12:30│ gpt-4o    │ OpenAI   │✅ 200│ 1.2s │$0.03│ │       │
│       │  │  │12:28│ claude-sn │ Anthropic│✅ 200│ 2.1s │$0.02│ │       │
│       │  │  │12:25│ gpt-4o (f)│ Anthropic│⚠️ 429│ 3.4s │$0.02│ │       │
│       │  │  │12:20│ llama3    │ Ollama   │✅ 200│ 0.5s │$0.00│ │       │
│       │  │  └────┴────────────┴──────────┴──────┴──────┴────┘ │       │
│       │  └────────────────────────────────────────────────────┘       │
└───────┴──────────────────────────────────────────────────────────────┘
```

### 3.2 Screen 2: Provider List (`/providers`)

```
┌──────────────────────────────────────────────────────────────────────┐
│ [Logo]  / Providers                                    [+ Add New]  │
├──────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  Providers                                                    │    │
│  │                                                               │    │
│  │  ┌─────────┬──────────┬──────────┬──────┬────────┬──────────┐│    │
│  │  │ Name    │ Type     │ Status   │Models│ Last   │ Actions  ││    │
│  │  ├─────────┼──────────┼──────────┼──────┼────────┼──────────┤│    │
│  │  │ OpenAI  │ OpenAI   │ 🟢 Onln  │  12  │ 30s ago│ [Edit][🗑]││    │
│  │  │ Claude  │ Anthropic│ 🟢 Onln  │   8  │ 28s ago│ [Edit][🗑]││    │
│  │  │ My Ollama│ Ollama  │ 🟢 Onln  │   5  │ 15s ago│ [Edit][🗑]││    │
│  │  │ GPT Plus│ChatGPT+  │ 🟡 Degr  │  30  │  2m ago│ [Edit][🗑]││    │
│  │  └─────────┴──────────┴──────────┴──────┴────────┴──────────┘│    │
│  │                                                               │    │
│  │  [Empty State: "No providers configured.                      │    │
│  │   Add your first provider to start routing requests."]        │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                       │
└──────────────────────────────────────────────────────────────────────┘
```

### 3.3 Screen 3: Add Provider Form (`/providers/add`)

```
┌──────────────────────────────────────────────────────────────────────┐
│ [Logo]  / Providers / Add Provider                   [Cancel] [Save] │
├──────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  Step 1: Provider Type                             [1] [2] [3]│    │
│  │                                                               │    │
│  │  Select the provider you want to connect:                     │    │
│  │                                                               │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐         │    │
│  │  │  OpenAI   │ │ Anthropic│ │  Google  │ │  Ollama  │         │    │
│  │  │  API key  │ │ API key  │ │ API key  │ │ auto-disc│         │    │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘         │    │
│  │                                                               │    │
│  │  ┌──────────┐ ┌───────────────────────┐                      │    │
│  │  │ChatGPT + │ │ OpenAI-Compatible     │                      │    │
│  │  │   Cookie  │ │ Custom base URL       │                      │    │
│  │  └──────────┘ └───────────────────────┘                      │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  Step 2: Connection Details                                  │    │
│  │                                                               │    │
│  │  Provider Name *     [My OpenAI Account              ]        │    │
│  │  API Key *           [sk-............................ ] [👁]   │    │
│  │  Organization ID     [org-.......................... ]        │    │
│  │  Base URL (opt)      [https://api.openai.com/v1      ]        │    │
│  │                                                               │    │
│  │  ┌─────────────────────────────────────┐                      │    │
│  │  │ [Test Connection]  → "✅ Connected!" │                      │    │
│  │  └─────────────────────────────────────┘                      │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  Step 3: Model Configuration                                 │    │
│  │                                                               │    │
│  │  Available Models:                                            │    │
│  │  ┌──────────────────────────────────────────────────────┐    │    │
│  │  │ [🔍 Search models...                          ]       │    │    │
│  │  │ ┌──────────────────────────────────────────────────┐  │    │    │
│  │  │ │ ☑ gpt-4o                    → gpt-4o-2024-11-20│  │    │    │
│  │  │ │ ☑ gpt-4o-mini               → gpt-4o-mini-2024-…│  │    │    │
│  │  │ │ ☐ gpt-4-turbo               → gpt-4-turbo-2024…│  │    │    │
│  │  │ │ ☐ o1                        → o1-2024-12-17   │  │    │    │
│  │  │ └──────────────────────────────────────────────────┘  │    │    │
│  │  └──────────────────────────────────────────────────────┘    │    │
│  │                                                               │    │
│  │  Custom Model Name Map:                                       │    │
│  │  [Short Name     ] → [Provider Model Name            ] [➕]  │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  Step 4: Defaults & Fallback                                 │    │
│  │                                                               │    │
│  │  Max Retries       [3    ▼]   Timeout (s)  [30  ▼]           │    │
│  │  Cost per 1K input  [$0.01  ]   Cost per 1K output [$0.03  ] │    │
│  │                                                               │    │
│  │  [Save Provider]                                              │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                       │
└──────────────────────────────────────────────────────────────────────┘
```

### 3.4 Screen 4: Stats Dashboard (`/dashboard`)

```
┌──────────────────────────────────────────────────────────────────────┐
│ [Logo]  / Dashboard                             [24h▼] [Export CSV] │
├──────────────────────────────────────────────────────────────────────┤
│  ┌──────────┬──────────────────────────────────────────────────────┐│
│  │          │  Tab: [Cost] [Tokens] [Latency] [Requests]           ││
│  │          │                                                       ││
│  │          │  ┌────────────────────────────────────────────┐      ││
│  │          │  │ Cost by Provider (Last 7 Days)             │      ││
│  │          │  │ ┌────────────────────────────────────────┐ │      ││
│  │          │  │ │ ████████████████████ OpenAI    $123.45│ │      ││
│  │          │  │ │ ████████████         Anthropic  $67.89│ │      ││
│  │          │  │ │ ██████               Google     $34.56│ │      ││
│  │          │  │ └────────────────────────────────────────┘ │      ││
│  │          │  └────────────────────────────────────────────┘      ││
│  │          │                                                       ││
│  │          │  ┌────────────────────────────┬────────────────────┐  ││
│  │          │  │ Cost Trend                 │ Top Models by Cost │  ││
│  │          │  │ ┌────────────────────────┐ │ ┌────────────────┐ │  ││
│  │          │  │ │ 📈 Line chart         │ │ │ gpt-4o  $89.12│ │  ││
│  │          │  │ │ (multi-series)         │ │ │ claude-sn$45.67│ │  ││
│  │          │  │ └────────────────────────┘ │ │ gemini   $23.45│ │  ││
│  │          │  └────────────────────────────┘ └────────────────┘ │  ││
│  │          │                                                       ││
│  │          │  ┌────────────────────────────────────────────┐      ││
│  │          │  │ Detailed Breakdown                         │      ││
│  │          │  │ ┌──────┬───────┬──────┬──────┬──────┬────┐│      ││
│  │          │  │ │Provdr│ Reqs  │InTok │OutTok│ Cost │%Tot││      ││
│  │          │  │ ├──────┼───────┼──────┼──────┼──────┼────┤│      ││
│  │          │  │ │OpenAI│ 1,234 │ 89K  │ 12K  │$123.4│ 55%││      ││
│  │          │  │ │Anthr │   567 │ 45K  │  8K  │$67.89│ 30%││      ││
│  │          │  │ │Google│   234 │ 23K  │  3K  │$34.56│ 15%││      ││
│  │          │  │ └──────┴───────┴──────┴──────┴──────┴────┘│      ││
│  │          │  └────────────────────────────────────────────┘      ││
│  └──────────┴──────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────────────┘
```

### 3.5 Screen 5: Model Catalog (`/models`)

```
┌──────────────────────────────────────────────────────────────────────┐
│ [Logo]  / Models                                    [🔍 Filter...] │
├──────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  Filter tags: [All] [OpenAI] [Anthropic] [Ollama] [ChatGPT+] [⚙️]  │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  Displaying 52 models (3 providers)         [Grid] [List] ▼  │    │
│  │                                                               │    │
│  │  ┌──────────┬─────────────┬──────────┬─────────┬──────────┐  │    │
│  │  │ Model    │ Provider     │ Status   │ Cost/1K │ Actions  │  │    │
│  │  ├──────────┼─────────────┼──────────┼─────────┼──────────┤  │    │
│  │  │ gpt-4o   │ [OpenAI]    │ 🟢 Ready  │ $0.01i  │ [Copy]   │  │    │
│  │  │          │             │          │ $0.03o  │ [Test]   │  │    │
│  │  ├──────────┼─────────────┼──────────┼─────────┼──────────┤  │    │
│  │  │claude-sn4│[Anthropic]  │ 🟢 Ready  │ $0.03i  │ [Copy]   │  │    │
│  │  ├──────────┼─────────────┼──────────┼─────────┼──────────┤  │    │
│  │  │ llama3   │ [Ollama🟢]  │ 🟢 Ready  │ Free    │ [Copy]   │  │    │
│  │  ├──────────┼─────────────┼──────────┼─────────┼──────────┤  │    │
│  │  │chatgpt-5 │ [ChatGPT+]  │ 🟡 Limit  │ $0.00   │ [Copy]   │  │    │
│  │  └──────────┴─────────────┴──────────┴─────────┴──────────┘  │    │
│  │                                                               │    │
│  │  [Empty State: "No models available.                          │    │
│  │   Add a provider to see available models."]                   │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                       │
└──────────────────────────────────────────────────────────────────────┘
```

### 3.6 Screen 6: API Key Management (`/keys`)

```
┌──────────────────────────────────────────────────────────────────────┐
│ [Logo]  / API Keys                                   [+ New Key]   │
├──────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  API Keys                                                    │    │
│  │                                                               │    │
│  │  ┌──────┬────────────┬──────────┬──────────┬──────────┬────┐ │    │
│  │  │ Name │ Key        │ Status   │ Usage/mo │ Rate Lim │Act │ │    │
│  │  ├──────┼────────────┼──────────┼──────────┼──────────┼────┤ │    │
│  │  │ Prod │ sk-...x3f9 │ 🟢 Active│ $123.45  │ 100/m    │[..]│ │    │
│  │  │ Dev  │ sk-...a2b8 │ 🟢 Active│ $12.34   │ 1000/m   │[..]│ │    │
│  │  │ Old  │ sk-...p4q7 │ 🔴 Revokd│ $0.00    │ -        │[..]│ │    │
│  │  └──────┴────────────┴──────────┴──────────┴──────────┴────┘ │    │
│  │                                                               │    │
│  │  [Empty State: "No API keys configured.                       │    │
│  │   Generate keys to secure your gateway."]                     │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                       │
└──────────────────────────────────────────────────────────────────────┘
```

### 3.7 Screen 7: Provider Health (`/health`)

```
┌──────────────────────────────────────────────────────────────────────┐
│ [Logo]  / Provider Health                    [🔄 Auto-refresh: ON]  │
├──────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  Provider Status                   Last checked: 12:34:56    │    │
│  │                                                               │    │
│  │  ┌──────────────────┐ ┌──────────────────┐                    │    │
│  │  │ 🟢 OpenAI         │ │ 🟢 Anthropic      │                    │    │
│  │  │ Online            │ │ Online            │                    │    │
│  │  │ Latency: 145ms    │ │ Latency: 230ms    │                    │    │
│  │  │ Requests: 1,234   │ │ Requests: 567     │                    │    │
│  │  │ Circuit: Closed   │ │ Circuit: Closed   │                    │    │
│  │  └──────────────────┘ └──────────────────┘                    │    │
│  │                                                               │    │
│  │  ┌──────────────────┐ ┌──────────────────┐                    │    │
│  │  │ 🟢 My Ollama      │ │ 🟡 ChatGPT Plus   │                    │    │
│  │  │ Online            │ │ Degraded          │                    │    │
│  │  │ Latency: 12ms     │ │ Latency: 1,200ms  │                    │    │
│  │  │ Requests: 89      │ │ Requests: 45      │                    │    │
│  │  │ Circuit: Closed   │ │ Circuit: Half-Open│                    │    │
│  │  └──────────────────┘ └──────────────────┘                    │    │
│  │                                                               │    │
│  │  [Empty State: "No providers configured.                      │    │
│  │   Add at least one provider to see health status."]            │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                       │
└──────────────────────────────────────────────────────────────────────┘
```

### 3.8 Screen 8: Settings (`/settings`)

```
┌──────────────────────────────────────────────────────────────────────┐
│ [Logo]  / Settings                                    [Save All]   │
├──────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  General Settings                                             │    │
│  │                                                               │    │
│  │  Proxy Port          [11435          ]                        │    │
│  │  Admin UI Port       [11436          ]                        │    │
│  │  Log Level           [info       ▼    ]                        │    │
│  │  Prometheus Metrics  [✅ Enable /metrics endpoint  ]          │    │
│  │  Usage Persistence   [💾 File-based (every 60s) ▼]           │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  Provider Defaults                                            │    │
│  │                                                               │    │
│  │  Default Max Retries    [3      ▼]                            │    │
│  │  Default Timeout (s)    [30     ▼]                            │    │
│  │  Backoff Strategy       [exponential ▼]                        │    │
│  │  Circuit Breaker:       [5 failures] in [60 seconds]          │    │
│  │  Circuit Recovery:      [30 seconds] cooldown                 │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  YAML Config (read-only)                                      │    │
│  │  ┌──────────────────────────────────────────────────────────┐ │    │
│  │  │ providers:                                               │ │    │
│  │  │   - name: "OpenAI"                                      │ │    │
│  │  │     type: openai                                         │ │    │
│  │  │     api_key: "${OPENAI_API_KEY}"                         │ │    │
│  │  │     models: ["gpt-4o", "gpt-4o-mini"]                    │ │    │
│  │  └──────────────────────────────────────────────────────────┘ │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                       │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 4. Component Hierarchy

### 4.1 Component Tree

```
App
├── TopBar
│   ├── Logo
│   ├── VersionBadge
│   ├── HelpButton (contextual question mark)
│   └── SettingsLink (gear icon)
│
├── Sidebar
│   ├── NavItem (Overview)          → route: /
│   ├── NavItem (Providers)         → route: /providers
│   ├── NavItem (Models)            → route: /models
│   ├── NavItem (Dashboard)         → route: /dashboard
│   ├── NavItem (API Keys)          → route: /keys
│   ├── NavItem (Health)            → route: /health
│   ├── SidebarDivider
│   └── NavItem (Settings)          → route: /settings
│
└── MainContent (uses React Router)
    │
    ├── [route: /] OverviewPage
    │   ├── SummaryCardsRow
    │   │   └── MetricCard (×4) — Total Cost, Requests, Active Providers, Avg Latency
    │   ├── CostTrendChart (line chart — Recharts or Chart.js)
    │   │   ├── TimeRangeSelector (7d, 30d, 90d)
    │   │   └── ChartLegend (per provider color)
    │   └── RecentRequestsTable
    │       ├── TableHeader (sortable columns)
    │       ├── TableRow (×20) — each with status icon, model, provider badge, latency, cost
    │       └── ViewAllLink → /dashboard/requests
    │
    ├── [route: /providers] ProviderListPage
    │   ├── PageHeader (title + AddButton)
    │   ├── ProviderTable
    │   │   ├── TableHeader
    │   │   ├── ProviderRow (×N)
    │   │   │   ├── ProviderName
    │   │   │   ├── ProviderTypeBadge (colored by type)
    │   │   │   ├── StatusBadge (Online/Degraded/Offline/Unknown)
    │   │   │   ├── ModelCount
    │   │   │   ├── LastCheckedTimestamp
    │   │   │   └── RowActions (EditButton, DeleteButton, TestConnectionButton)
    │   │   └── EmptyState
    │   └── DeleteConfirmationDialog
    │
    ├── [route: /providers/add, /providers/:id/edit] ProviderFormPage
    │   ├── WizardStepper (Step 1/2/3/4 indicator)
    │   ├── ProviderTypeSelector (Step 1)
    │   │   └── ProviderTypeCard (×6) — OpenAI, Anthropic, Google, Ollama, ChatGPT+, OpenAI-Compatible
    │   ├── ConnectionForm (Step 2)
    │   │   ├── TextInput (name)
    │   │   ├── PasswordInput (API key — masked, toggle visibility)
    │   │   ├── TextInput (org ID — conditional)
    │   │   ├── TextInput (base URL — conditional, default pre-filled)
    │   │   └── TestConnectionButton → ConnectionStatus
    │   ├── ModelConfigForm (Step 3)
    │   │   ├── SearchInput (filter available models)
    │   │   ├── ModelCheckboxList (select provider models to expose)
    │   │   └── ModelNameMapInput (user-model → provider-model mapping rows)
    │   └── DefaultsForm (Step 4)
    │       ├── SelectInput (max retries)
    │       ├── NumberInput (timeout seconds)
    │       ├── NumberInput (cost per 1K input tokens)
    │       └── NumberInput (cost per 1K output tokens)
    │
    ├── [route: /models] ModelCatalogPage
    │   ├── SearchFilterBar
    │   │   ├── SearchInput
    │   │   ├── ProviderFilterChips
    │   │   └── ViewToggle (grid/table)
    │   ├── ModelTable
    │   │   ├── TableHeader
    │   │   ├── ModelRow (×N)
    │   │   │   ├── ModelName
    │   │   │   ├── ProviderBadge (colored pill)
    │   │   │   ├── StatusIndicator
    │   │   │   ├── CostPerToken
    │   │   │   └── RowActions (CopyNameButton, QuickTestButton)
    │   │   └── EmptyState
    │   └── ModelDetailSlideOver (conditional)
    │       ├── ModelInfo (full name, provider, pricing, context window)
    │       ├── UsageStats (requests, tokens, cost)
    │       └── FallbackChainVisualization
    │
    ├── [route: /dashboard] DashboardPage
    │   ├── GlobalTimeRangeSelector (24h, 7d, 30d, 90d, custom)
    │   ├── DashboardTabs
    │   │   ├── Tab: CostTab
    │   │   │   ├── CostByProviderChart (horizontal bar)
    │   │   │   ├── CostTrendChart (multi-series line)
    │   │   │   ├── TopModelsByCost (sorted horizontal bar)
    │   │   │   └── CostBreakdownTable (provider, requests, tokens in/out, cost, % total)
    │   │   ├── Tab: TokensTab
    │   │   │   ├── InputVsOutputChart (stacked area)
    │   │   │   ├── PerModelTokenChart
    │   │   │   └── DailyTokenVolumeChart
    │   │   ├── Tab: LatencyTab
    │   │   │   ├── LatencyByProviderChart (P50/P95/P99 multi-line)
    │   │   │   └── RequestDurationHistogram
    │   │   └── Tab: RequestsTab
    │   │       ├── RequestLogTable (infinite scroll)
    │   │       │   ├── RequestRow (timestamp, model, provider, status, latency, tokens, cost)
    │   │       │   └── RequestDetailExpandable (full req/res headers + body)
    │   │       └── FilterBar (provider, model, status, date range)
    │   └── EmptyState (no data yet)
    │
    ├── [route: /keys] ApiKeyListPage
    │   ├── PageHeader (title + AddButton)
    │   ├── ApiKeyTable
    │   │   ├── TableHeader
    │   │   ├── ApiKeyRow (×N)
    │   │   │   ├── KeyName
    │   │   │   ├── MaskedKeyPreview (sk-...xxxx)
    │   │   │   ├── StatusBadge (Active/Revoked)
    │   │   │   ├── UsageThisMonth
    │   │   │   ├── RateLimit
    │   │   │   ├── LastUsed
    │   │   │   └── RowActions (CopyButton, EditLimitsButton, RevokeButton, DeleteButton)
    │   │   └── EmptyState
    │   ├── CreateKeyDialog (Modal)
    │   │   ├── TextInput (key name)
    │   │   ├── NumberInput (rate limit per minute)
    │   │   ├── NumberInput (monthly budget cap)
    │   │   ├── DatePicker (optional expiration)
    │   │   └── GenerateButton → KeyRevealBanner (show key once)
    │   └── RevokeConfirmationDialog
    │
    ├── [route: /health] HealthPage
    │   ├── AutoRefreshToggle
    │   ├── LastCheckedTimestamp
    │   └── ProviderStatusGrid
    │       └── ProviderStatusCard (×N)
    │           ├── StatusDot (green/yellow/red/grey)
    │           ├── ProviderName
    │           ├── StatusLabel (Online/Degraded/Offline/Unknown)
    │           ├── LatencyMs
    │           ├── RequestCount
    │           └── CircuitBreakerState (Closed/Open/Half-Open badge)
    │
    └── [route: /settings] SettingsPage
        ├── GeneralSettingsSection
        │   ├── NumberInput (proxy port)
        │   ├── NumberInput (admin port)
        │   ├── SelectInput (log level)
        │   ├── ToggleSwitch (Prometheus metrics)
        │   └── SelectInput (usage persistence mode)
        ├── ProviderDefaultsSection
        │   ├── SelectInput (default max retries)
        │   ├── SelectInput (default timeout)
        │   ├── SelectInput (backoff strategy)
        │   ├── NumberInput (circuit breaker failure threshold)
        │   ├── NumberInput (circuit breaker window seconds)
        │   └── NumberInput (circuit breaker cooldown seconds)
        └── YamlPreviewSection
            └── SyntaxHighlightedCodeBlock (read-only YAML)

Shared / Cross-cutting Components:
├── ToastNotification (success, error, warning, info)
├── Spinner / LoadingSkeleton (for async content)
├── PageHeader (title + optional actions)
├── ConfirmDialog (delete, revoke confirmation)
├── Badge (status, provider type, version)
├── Tooltip (contextual help on icons)
└── ResponsiveContainer (max-width wrapper)
```

### 4.2 Component States

Every interactive component must implement these states:

| State | Visual | Behavior |
|-------|--------|----------|
| **Default** | Normal appearance | Ready for interaction |
| **Hover** | Slightly darker bg, cursor pointer | Mouse hover (150ms transition) |
| **Focus** | Visible ring (2px solid primary, offset 2px) | Keyboard Tab focus |
| **Active** | Pressed state (scale 0.98, darker bg) | MouseDown / TouchStart |
| **Disabled** | 50% opacity, no pointer events | Not interactive |
| **Error** | Red border/message | Validation failure |
| **Loading** | Spinner or skeleton | Data being fetched |
| **Empty** | Illustration + message | No data to display |
| **Success** | Green flash/checkmark | Action completed |

---

## 5. User Flows

### 5.1 Flow 1: Initial Deployment & First Request

```
Actor: Priya (Homelab Operator)
Goal: Deploy OllaRouter and route first request to GPT-4o

START
  │
  1. Priya adds ollarouter service to docker-compose.yml
     │
  2. Sets environment variables:
     │  - OLLAROUTER_PORT=11435
     │  - OLLAROUTER_ADMIN_PORT=11436
     │  - OLLAROUTER_OLLAMA_HOST=http://ollama:11434
     │
  3. Creates providers.yaml with OpenAI API key
     │
  4. Runs: docker compose up -d
     │
  5. Opens browser → http://localhost:11436
     │  ┌──────────────────────────────────────────────┐
     │  │  Sidebar on left, "Overview" page active     │
     │  │  Empty state: "No providers configured.      │
     │  │  Add your first provider to get started."   │
     │  └──────────────────────────────────────────────┘
     │
  6. Clicks "Providers" in sidebar
     │  ┌──────────────────────────────────────────────┐
     │  │  Empty state: "No providers yet.             │
     │  │  [+ Add Provider]"                          │
     │  └──────────────────────────────────────────────┘
     │
  7. Clicks [+ Add Provider]
     │  → Wizard Step 1: Provider Type Selection
     │
  8. Selects "OpenAI" card
     │  → Wizard Step 2: Connection Form
     │
  9. Enters "My OpenAI" as name
     │  Pastes API key → [Test Connection] → ✅ Success
     │
  10. → Step 3: Selects gpt-4o, gpt-4o-mini from model list
     │
  11. → Step 4: Default retry 3, timeout 30s
     │  Clicks [Save Provider]
     │
  12. Provider appears in list with 🟢 Online status
     │
  13. Clicks "Models" in sidebar
     │  ┌──────────────────────────────────────────────┐
     │  │  gpt-4o  [OpenAI]  🟢 Ready                  │
     │  │  gpt-4o-mini  [OpenAI]  🟢 Ready             │
     │  └──────────────────────────────────────────────┘
     │
  14. Sends test via curl:
     │  curl http://localhost:11435/api/chat \
     │    -d '{"model":"gpt-4o","messages":[{"role":"user","content":"Hello"}]}'
     │  → Ollama-format JSON response received
     │
  15. Configures Open WebUI:
     │  OLLAMA_BASE_URL=http://ollarouter:11435
     │  → gpt-4o model now available in Open WebUI chat
     │
END  🎉
```

### 5.2 Flow 2: Adding a Provider via Web UI

```
Actor: Morgan (Platform Engineer)
Goal: Add Anthropic provider for team access

START
  │
  1. Opens browser → http://localhost:11436/providers
     │  ┌────────────────────────────────────────────┐
     │  │  Table: OpenAI 🟢 | Anthropic not yet      │
     │  └────────────────────────────────────────────┘
     │
  2. Clicks [+ Add New]
     │  → Wizard loads at Step 1: Provider Type
     │
  3. Morgan selects "Anthropic" from the icon grid
     │  → Transition to Step 2 with slide animation
     │
  4. Step 2 — Connection Details:
     │  │  Name:            [My Anthropic Account      ]
     │  │  API Key:         [sk-ant-................] [👁]
     │  │  Base URL:        [https://api.anthropic.com  ] (pre-filled)
     │  │
     │  4a. Clicks [Test Connection]
     │  4b. Spinner for ~2s
     │  4c. → 🟢 "✅ Connected! (210ms)"
     │
  5. Step 3 — Model Configuration:
     │  │  ☑ claude-sonnet-4-20250514  → "claude-sonnet-4"
     │  │  ☑ claude-3-haiku           → "claude-haiku"
     │  │
     │  5a. Custom mapping: [claude-sonnet] → [claude-sonnet-4-20250514]
     │
  6. Step 4 — Defaults & Fallback:
     │  │  Max Retries:  [2]
     │  │  Timeout:      [60s]
     │  │  Cost/1K in:   [$0.015]  (pre-filled for Anthropic)
     │  │  Cost/1K out:  [$0.075]  (pre-filled for Anthropic)
     │
  7. Clicks [Save Provider]
     │  → Success toast: "✅ Provider 'Anthropic' added successfully"
     │  → Redirects to /providers
     │  → Anthropic now visible in table with 🟢 Online
     │  → gpt-4o → claude-sonnet-4 fallback from EPIC-1 works automatically
     │
END
```

### 5.3 Flow 3: Viewing the Stats Dashboard

```
Actor: Jordan (Tech Lead)
Goal: Understand monthly LLM spending

START
  │
  1. Opens browser → http://localhost:11436/dashboard
     │  ┌──────────────────────────────────────────────┐
     │  │  Time range: [Last 7 days ▼]                 │
     │  │  Tabs: [Cost] [Tokens] [Latency] [Requests]  │
     │  └──────────────────────────────────────────────┘
     │
  2. Cost tab is active by default
     │  ┌──────────────────────────────────────────────┐
     │  │  Summary: $234.50 this month                   │
     │  │                                                │
     │  │  ┌ Cost by Provider ───────────────────┐     │
     │  │  │ OpenAI    ████████████████  $123.45  │     │
     │  │  │ Anthropic ██████████        $67.89   │     │
     │  │  │ Google    ██████            $34.56   │     │
     │  │  │ Ollama    ▓▓▓▓▓▓            $0.00    │     │
     │  │  └──────────────────────────────────────┘     │
     │  │                                                │
     │  │  ┌ Cost Trend ─────────────────────────┐     │
     │  │  │ 📈 Multi-line chart — daily cost     │     │
     │  │  │ (7 data points, 3 series)            │     │
     │  │  └──────────────────────────────────────┘     │
     │  │                                                │
     │  │  ┌ Top Models by Cost ─────────────────┐     │
     │  │  │ gpt-4o       $89.12 █████████████   │     │
     │  │  │ claude-sn-4  $45.67 ███████         │     │
     │  │  │ gpt-4o-mini  $34.33 █████           │     │
     │  │  └──────────────────────────────────────┘     │
     │  │                                                │
     │  │  ┌ Detailed Breakdown ─────────────────┐     │
     │  │  │ Provider │ Reqs │ Tokens  │ Cost    │     │
     │  │  │ OpenAI   │ 1,234 │ 101,234 │ $123.45 │     │
     │  │  │ Anthropic│ 567   │ 53,456  │ $67.89  │     │
     │  │  └──────────────────────────────────────┘     │
     │  └──────────────────────────────────────────────┘
     │
  3. Clicks on "OpenAI" row in breakdown table
     │  → Per-model drill-down view loads inline
     │  │  gpt-4o: 890 reqs, $89.12
     │  │  gpt-4o-mini: 344 reqs, $34.33
     │
  4. Switches time range to [Last 30 Days]
     │  → All charts refresh with animation (~500ms)
     │
  5. Clicks [Export CSV] → downloads usage-report-2026-05-20.csv
     │
END
```

### 5.4 Flow 4: Fallback Configuration & Scenario

```
Actor: Alex (AI Developer)
Goal: Configure fallback chain so requests never fail

START
  │
  1. Opens browser → http://localhost:11436/providers
     │
  2. Clicks [Edit] on "OpenAI" provider
     │  → Provider form opens with existing values
     │
  3. Navigates to Step 4: Defaults & Fallback
     │
  4. Configures fallback:
     │  ┌──────────────────────────────────────────────┐
     │  │  Fallback Chain:                              │
     │  │  1. [OpenAI/gpt-4o              ] ← autofilled │
     │  │  2. [Anthropic/claude-sonnet-4  ] [⬆⬇⬇]      │
     │  │  3. [Ollama/llama3             ] [⬆⬇⬇]      │
     │  │  [+ Add fallback]                            │
     │  │                                             │
     │  │  Retry Policy:                               │
     │  │  ✓ Retry on 429 (max 3, exp backoff)         │
     │  │  ✓ Retry on 5xx (max 2)                      │
     │  │  □ Retry on timeout                          │
     │  └──────────────────────────────────────────────┘
     │
  5. Saves provider
     │
  6. Sends request with model "gpt-4o"
     │  → OpenAI returns 429 (rate-limited)
     │  → OllaRouter retries once (1s backoff) → still 429
     │  → Falls back to Anthropic claude-sonnet-4
     │  → Response received (from Claude, wrapped in Ollama NDJSON)
     │  → Response header: x-ollarouter-provider: anthropic (fallback)
     │
  7. Checks Dashboard → Requests tab
     │  ┌──────────────────────────────────────────────┐
     │  │  Request Log:                                 │
     │  │  12:30:00  gpt-4o   OpenAI   429  fallback→  │
     │  │  12:30:01  gpt-4o   Anthropic 200  ✓        │
     │  └──────────────────────────────────────────────┘
     │
  8. Expands the request row:
     │  Fallback chain: OpenAI(429) → Anthropic(200)
     │  Total latency: 3.4s
     │
END
```

### 5.5 Flow 5: ChatGPT Plus Provider Setup

```
Actor: Sam (Power User / ChatGPT Plus subscriber)
Goal: Access ChatGPT Plus models via Ollama API

START
  │
  1. Opens browser → http://localhost:11436/providers/add
     │
  2. Selects "ChatGPT Plus" from provider type grid
     │  → Form changes to ChatGPT-specific fields
     │
  3. Step 2 — Connection Details:
     │  │  Name:              [My ChatGPT Plus         ]
     │  │  Session Cookie:    [__Secure-next-auth......] [👁]
     │  │  User Agent:        [Mozilla/5.0 (auto-filled)]
     │  │
     │  │  [Test Connection] → Spinner → "✅ Connected! (1.2s)"
     │  │  ✓ Cookie valid, 32 models detected
     │
  4. Step 3 — Model Configuration:
     │  │  ☑ gpt-5                → "chatgpt-gpt-5"
     │  │  ☑ gpt-5.5             → "chatgpt-gpt-5.5"
     │  │  ☑ o3                  → "chatgpt-o3"
     │  │  ☑ o4-mini             → "chatgpt-o4-mini"
     │  │  ☑ deep-research       → "chatgpt-deep-research"
     │  │  ☑ agent-mode          → "chatgpt-agent"
     │  │  ✓ 30 models available
     │
  5. Step 4 — No cost config needed (fixed subscription)
     │  Pre-filled: Cost/1K = $0.00 (included in subscription)
     │
  6. Saves → Provider appears with 🟡 Degraded status (cookie may expire)
     │
  7. Opens Open WebUI → OLLAMA_BASE_URL points to OllaRouter
     │  → /api/tags now includes "chatgpt-gpt-5", "chatgpt-gpt-5.5", etc.
     │
  8. Sends chat with model "chatgpt-gpt-5"
     │  → Request proxied via ChatGPT Plus
     │  → Response returned as Ollama NDJSON
     │
END
```

### 5.6 Flow 6: API Key Management

```
Actor: Morgan (Platform Engineer)
Goal: Create an API key for the marketing team with budget cap

START
  │
  1. Opens browser → http://localhost:11436/keys
     │  ┌──────────────────────────────────────────────┐
     │  │  Empty: "No API keys yet. Generate your      │
     │  │  first key to secure the gateway."           │
     │  └──────────────────────────────────────────────┘
     │
  2. Clicks [+ New Key]
     │  → Modal dialog opens
     │
  3. Fills form:
     │  │  Key Name:        [Marketing Team Key          ]
     │  │  Rate Limit:      [50          ] requests/min
     │  │  Monthly Budget:  [$100.00     ]
     │  │  Expires:         [2026-06-20  ] (optional)
     │  │
     │  3a. Clicks [Generate Key]
     │
  4. ✅ Key generated — shown once:
     │  ┌──────────────────────────────────────────────┐
     │  │  🎉 Key generated! Copy it now —             │
     │  │  you won't be able to see it again.          │
     │  │                                              │
     │  │  [or-xxxxxxxxxxxxxxxxxxxxxxxxxxxx] [📋 Copy] │
     │  └──────────────────────────────────────────────┘
     │
  5. Key appears in list:
     │  Marketing Team Key | or-...x3f9 | 🟢 Active | $0.00 | 50/m
     │
  6. Shares key with marketing team
     │  → Marketing uses it in their HTTP client header:
     │  │  Authorization: Bearer or-xxxxxxxxxxxxxxxxxx
     │
  7. Morgan checks back a week later:
     │  Marketing Team Key | or-...x3f9 | 🟢 Active | $34.50 | 48/m
     │
END
```

---

## 6. Provider Setup UX

### 6.1 Provider Type Cards

Each provider type is represented by a clickable card (120×140px) in a responsive grid:

```
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│ [OpenAI logo]    │  │ [Anthropic logo] │  │  [Google logo]   │
│                  │  │                  │  │                  │
│     OpenAI       │  │    Anthropic     │  │  Google Gemini   │
│  API Key-based   │  │  API Key-based   │  │  API Key-based   │
└──────────────────┘  └──────────────────┘  └──────────────────┘

┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│ [Ollama logo]    │  │ [ChatGPT icon]   │  │  [Custom icon]   │
│                  │  │                  │  │                  │
│  Local Ollama    │  │  ChatGPT Plus    │  │ OpenAI-Compatible │
│  Auto-discovery  │  │  Cookie-based    │  │   Custom URL     │
└──────────────────┘  └──────────────────┘  └──────────────────┘
```

**Interaction:**
- Click card → card highlights (2px primary border, slight scale up)
- Next button at bottom-right activates
- Hover: card lifts (shadow level 2, translateY(-2px))

### 6.2 Provider Form — Conditional Fields

| Provider Type | Step 2 Fields Shown | Step 3 Default Models |
|---------------|--------------------|----------------------|
| **OpenAI** | API Key (required), Org ID (opt), Base URL (pre-filled) | gpt-4o, gpt-4o-mini, gpt-4-turbo, o1, o3 |
| **Anthropic** | API Key (required), Base URL (pre-filled) | claude-sonnet-4, claude-haiku, claude-opus |
| **Google Gemini** | API Key (required) | gemini-2.5-pro, gemini-2.5-flash |
| **Ollama** | No API key (local). Poll interval for auto-discovery | Auto-discovered from local Ollama |
| **ChatGPT Plus** | Session Cookie (required), User Agent (auto-filled) | Auto-detected from account (30+ models) |
| **OpenAI-Compatible** | API Key (required), Base URL (required, no default) | User-entered model names |

### 6.3 Test Connection UX

```
┌─────────────────────────────────────────────────┐
│  [Test Connection]                               │
│                                                   │
│  States:                                          │
│  ┌─────────────────────────────────────────────┐  │
│  │ [Test Connection]  ← Default (idle)        │  │
│  ├─────────────────────────────────────────────┤  │
│  │ [ ⟳ Testing... ]  ← Loading (disable btn) │  │
│  ├─────────────────────────────────────────────┤  │
│  │ ✅ Connected! (210ms)  ← Success (green)   │  │
│  │   12 models available                        │  │
│  ├─────────────────────────────────────────────┤  │
│  │ ❌ Connection failed: 401 Unauthorized      │  │
│  │   [Check your API key and try again]        │  │
│  ├─────────────────────────────────────────────┤  │
│  │ ⚠️ Connected but degraded (1.2s latency)    │  │
│  └─────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────┘
```

### 6.4 Provider Status Indicators

| Status | Dot Color | Background | Icon | Meaning |
|--------|-----------|------------|------|---------|
| **Online** | `#10B981` (green) | `rgba(16,185,129,0.1)` | ✅ | Last health check passed, circuit closed |
| **Degraded** | `#F59E0B` (amber) | `rgba(245,158,11,0.1)` | ⚠️ | High latency (>1s) or circuit half-open |
| **Offline** | `#EF4444` (red) | `rgba(239,68,68,0.1)` | ❌ | Health check failed or circuit open |
| **Unknown** | `#94A3B8` (grey) | `rgba(148,163,184,0.1)` | ❓ | Not yet checked or initial state |

---

## 7. Token/Cost Stats Dashboard UX

### 7.1 Summary Cards

Each metric card shows:
- **Icon** (32px, color-coded by metric type)
- **Value** (36px, semi-bold, primary text)
- **Label** (14px, secondary text)
- **Trend indicator** (small up/down arrow with percentage vs previous period)

```
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│ 💰              │  │ 🔄               │  │ 🔌              │  │ ⏱               │
│ $234.50         │  │ 1,847            │  │ 3               │  │ 342ms           │
│ Total Cost      │  │ Requests         │  │ Active Providers│  │ Avg Latency     │
│ this month      │  │ today            │  │                 │  │                 │
│ ↑ 12.3% vs prev │  │ ↑ 8.1% vs prev  │  │ —               │  │ ↓ 5.2% vs prev │
└─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘
```

### 7.2 Chart Specifications

| Chart | Type | Data | Library Recommendation |
|-------|------|------|----------------------|
| Cost Trend | Multi-series line chart (2-4 lines) | Daily cost per provider over time range | Recharts `<LineChart>` |
| Cost by Provider | Horizontal bar chart | Total cost per provider | Recharts `<BarChart>` layout="horizontal" |
| Top Models by Cost | Horizontal bar chart | Cost per model (top 10) | Recharts `<BarChart>` |
| Input vs Output Tokens | Stacked area chart | Daily input + output tokens | Recharts `<AreaChart>` stack |
| Latency by Provider | Multi-series line (P50/P95/P99) | Daily latency percentiles | Recharts `<LineChart>` |
| Request Duration Histogram | Bar chart | Bucketed request durations | Recharts `<BarChart>` |

**Chart Design Guidelines:**
- All charts use consistent color mapping per provider:
  - OpenAI: `#2563EB` (blue)
  - Anthropic: `#7C3AED` (purple)
  - Google: `#059669` (green)
  - Ollama: `#F59E0B` (amber)
  - ChatGPT+: `#EF4444` (red)
- Hover on data points shows tooltip with exact values
- Charts are responsive (auto-resize on window change)
- Empty state charts show "No data available" overlay
- Chart legends are interactive (click to toggle series)

### 7.3 Request Log Table

| Column | Width | Alignment | Sortable | Filterable |
|--------|-------|-----------|----------|------------|
| Timestamp | 150px | left | ✅ | Date range |
| Model | 180px | left | ✅ | ✅ Text search |
| Provider | 120px | left | ✅ | ✅ Multi-select |
| Status | 80px | center | ✅ | ✅ Multi-select |
| Latency | 80px | right | ✅ | Range slider |
| Input Tokens | 100px | right | ✅ | No |
| Output Tokens | 100px | right | ✅ | No |
| Cost | 80px | right | ✅ | No |

**Expandable Row Behavior:**
- Click row → expand with slide-down animation
- Shows: request headers, response headers, fallback chain, raw response preview (truncated)
- "View Full" link opens raw response in a modal

### 7.4 Data Refresh Strategy

| Data Type | Refresh Interval | Mechanism |
|-----------|-----------------|-----------|
| Summary cards | Every 30s | Polling (incremental) |
| Cost/Token charts | Every 60s | Polling (full refresh) |
| Request log table | On page load + manual | Manual "Refresh" button |
| Latency charts | Every 60s | Polling (full refresh) |
| Provider health | Every 30s | Polling (in background) |

**Visual Feedback During Refresh:**
- Charts show subtle shimmer animation during data load
- Tables show skeleton rows (3 placeholder rows with pulse animation)
- Error state: "Failed to load data. [Retry]" link
- Stale indicator: "Last updated: 45s ago" timestamp above charts

---

## 8. Model & Fallback Configuration UX

### 8.1 Fallback Chain Configuration

The fallback chain is configured at the **provider level** (Step 4 of Add/Edit Provider) and visualized as a ordered list with drag-to-reorder:

```
┌────────────────────────────────────────────────────────────┐
│  Fallback Chain for: gpt-4o                                │
│                                                             │
│  Priority order (drag to reorder):                          │
│                                                             │
│  1. ┌─────────────────────────────────────────────────┐    │
│     │ ⬡  [OpenAI  ▼] [gpt-4o  ▼]  ⚡primary          │    │
│     └─────────────────────────────────────────────────┘    │
│      ⬇                                                   │
│  2. ┌─────────────────────────────────────────────────┐    │
│     │ ⬡  [Anthropic  ▼] [claude-sonnet-4  ▼]         │    │
│     └─────────────────────────────────────────────────┘    │
│      ⬇                                                   │
│  3. ┌─────────────────────────────────────────────────┐    │
│     │ ⬡  [Ollama  ▼] [llama3  ▼]                     │    │
│     └─────────────────────────────────────────────────┘    │
│                                                             │
│  [+ Add Fallback Target]                                    │
│                                                             │
│  ┌────────────────────────────────────────────────────────┐│
│  │  Retry Policy for Primary:                              ││
│  │  ☑ Retry on 429 (max [3  ] times, exp backoff)        ││
│  │  ☑ Retry on 5xx (max [2  ] times)                     ││
│  │  ☐ Retry on timeout                                    ││
│  └────────────────────────────────────────────────────────┘│
│                                                             │
│  [Save Chain]                                               │
└────────────────────────────────────────────────────────────┘
```

**Interaction:**
- Drag handles (⬡) on each row enable drag-and-drop reordering
- Provider dropdown shows all configured providers
- Model dropdown shows models available for the selected provider
- "primary" badge on slot 1 indicates first-choice
- "⚡fallback" badge on slots 2+ indicates fallback role
- "Add Fallback Target" button appends a new empty row
- Remove button (🗑) on each row deletes that fallback

### 8.2 Model Catalog — Filter & Search

```
┌────────────────────────────────────────────────────────────┐
│  [🔍 Search by model name...                  ] [Clear]  │
│                                                             │
│  Provider: [All ▾]  Status: [All ▾]  Sort: [A-Z ▾]        │
│                                                             │
│  Active filters: [OpenAI ✕] [Ready ✕] [Clear all]         │
│                                                             │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Found 8 models (filtered from 52)                  │    │
│  │                                                     │    │
│  │  ┌──────────┬─────────────┬──────────┬──────────┐  │    │
│  │  │ Model    │ Provider     │ Status   │ Cost/1K  │  │    │
│  │  ├──────────┼─────────────┼──────────┼──────────┤  │    │
│  │  │ gpt-4o   │ [OpenAI]    │ 🟢 Ready  │ $0.01/$0.03│  │    │
│  │  │ gpt-4o-mi│ [OpenAI]    │ 🟢 Ready  │ $0.0015/..│  │    │
│  │  └──────────┴─────────────┴──────────┴──────────┘  │    │
│  └────────────────────────────────────────────────────┘    │
└────────────────────────────────────────────────────────────┘
```

### 8.3 Model Detail Slide-Over

Clicking a model name opens a slide-over panel from the right:

```
┌────────────────────────────────────────────────────────────┐
│  [← Back]  Model Details                     [Close ✕]    │
│                                                             │
│  gpt-4o                                                     │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Provider:    [OpenAI] 🟢                            │  │
│  │  Full Name:   gpt-4o-2024-11-20                      │  │
│  │  Context:     128,000 tokens                          │  │
│  │  Pricing:     $0.01 / 1K in  ·  $0.03 / 1K out      │  │
│  │  Modality:    Text + Image                            │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Usage (Last 7 Days):                                       │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Requests:    890 (52% of total)                     │  │
│  │  Tokens In:   112,345                                 │  │
│  │  Tokens Out:  23,456                                  │  │
│  │  Cost:        $89.12 (38% of total)                  │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Fallback Chain:                                            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  1. gpt-4o → [OpenAI]                              │  │
│  │  2. claude-sonnet-4 → [Anthropic]                   │  │
│  │  3. llama3 → [Ollama]                               │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  [Quick Test]  [Copy Model Name]                            │
└────────────────────────────────────────────────────────────┘
```

---

## 9. Accessibility Plan

### 9.1 Compliance Target

| Standard | Level | Target |
|----------|-------|--------|
| WCAG 2.1 | AA | All UI features |
| WCAG 2.1 | AAA | Text contrast, keyboard navigation |
| Section 508 | — | All UI features |

### 9.2 Color & Contrast

| Requirement | Target | Verification Method |
|-------------|--------|---------------------|
| Normal text contrast | ≥ 4.5:1 (AA) | `ux_check_accessibility` tool on all text/background pairs |
| Large text (≥18px bold or ≥24px) | ≥ 3:1 (AA) | Same as above |
| UI component contrast (borders, icons) | ≥ 3:1 (AA) | Same as above |
| Focus indicators | 2px solid #2563EB ring with 2px offset | Visual inspection + keyboard tab test |
| No color-only information | All status indicators include icon + text label | Audit all status badges |
| `prefers-color-scheme: dark` | Full dark mode support with maintained contrast | CSS custom properties + media query |

**Color Palette Contrast Verification (light mode):**

| Foreground | Background | Ratio | WCAG AA | WCAG AAA |
|-----------|-----------|-------|---------|----------|
| `#1E293B` (text) | `#F8FAFC` (bg) | 13.2:1 | ✅ Pass | ✅ Pass |
| `#2563EB` (primary) | `#FFFFFF` (white) | 5.8:1 | ✅ Pass | ✅ Pass (large text) |
| `#64748B` (secondary) | `#F8FAFC` (bg) | 4.7:1 | ✅ Pass | ❌ Fail (normal text) |
| `#10B981` (success) | `#F8FAFC` (bg) | 3.2:1 | ❌ Fail | ❌ Fail |

**Mitigation for low-contrast elements:**
- Success text/icons on white backgrounds to use `#059669` (darker green, 4.7:1 ratio)
- Secondary text to use `#475569` (darker grey, 5.8:1 ratio)
- Paired icon + label approach ensures information is never conveyed by color alone

### 9.3 Keyboard Navigation

| Requirement | Implementation |
|-------------|---------------|
| All interactive elements reachable via Tab | Native HTML elements + `tabindex` where needed |
| Visible focus ring on all focusable elements | `focus-visible:ring-2 focus-visible:ring-primary` |
| Logical tab order | Follow visual layout (sidebar → main content → actions) |
| Skip-to-content link | First focusable element on every page |
| Modals trap focus | Focus stays in modal, Esc closes, focus returns to trigger |
| Dropdowns/menus | Arrow keys navigate items, Enter selects, Esc closes |
| Tables | Arrow keys navigate cells in data tables |
| No keyboard traps | All interactive elements can be Tab'ed away from |
| Draggable lists | Keyboard alternative: move up/down buttons for fallback reorder |

### 9.4 Screen Reader Support

| Requirement | Implementation |
|-------------|---------------|
| Semantic HTML | Use `<nav>`, `<main>`, `<section>`, `<article>`, `<table>` elements |
| ARIA landmarks | `role="navigation"`, `role="main"`, `role="complementary"` |
| Dynamic content announcements | `aria-live="polite"` regions for toasts, status changes |
| Form validation | `aria-describedby` linking to error messages, `aria-invalid` on error fields |
| Chart alternatives | Data tables alongside charts (hidden on desktop, visible to screen readers) |
| Status icons | `aria-label` on status dots: "OpenAI provider — Online" |
| Loading states | `aria-busy="true"` on containers, `aria-label="Loading..."` on spinners |
| Empty states | `role="status"` with descriptive text: "No providers configured yet" |
| Modal dialogs | `role="dialog"`, `aria-modal="true"`, `aria-labelledby` referencing title |
| Table sort | `aria-sort="ascending/descending"` on sortable column headers |
| Toggle switches | `role="switch"`, `aria-checked` for on/off toggles |

### 9.5 Motion & Animation

| Requirement | Implementation |
|-------------|---------------|
| `prefers-reduced-motion` | Respect via `@media (prefers-reduced-motion: reduce)` — disable all non-essential animations |
| Essential animations only | Loading spinners, progress bars, skeleton screens (duration > 0) |
| Non-essential animations off | Page transitions, card hover lifts, slide-over panels, toast entrances |
| No flashing content | No strobing/blinking effects (WCAG 2.3.1 — 3 flashes/sec max) |

### 9.6 Touch Targets

| Requirement | Target | Rationale |
|-------------|--------|-----------|
| Minimum touch target | 44×44px | WCAG 2.5.5 (AA) for pointer inputs |
| Small interactive elements | 24×24px minimum with adequate spacing | Table row actions, icon buttons |
| Form inputs | Minimum 40px height | Comfortable tap target on mobile |
| Button padding | Minimum 8px horizontal, 12px vertical | Ensures minimum target size |

### 9.7 Internationalization (i18n) Readiness

| Requirement | Implementation |
|-------------|---------------|
| All user-facing strings externalized | String constants file/module in `locales/en.json` |
| Number formatting | Use `Intl.NumberFormat` for token/currency formatting |
| Date/time formatting | Use `Intl.DateTimeFormat` with locale-aware patterns |
| Text direction | CSS logical properties (`margin-inline-start`, `padding-inline-end`) |
| RTL support | Layout tested with dir="rtl" attribute (post-MVP) |

---

## 10. Responsive Behavior

### 10.1 Breakpoints

| Breakpoint | Viewport | Layout | Device |
|-----------|----------|--------|--------|
| `xs` | < 640px | Single column, stacked | Mobile phones |
| `sm` | 640-767px | Single column, wider touch targets | Large phones |
| `md` | 768-1023px | Two-column, sidebar collapses to icons | Tablets |
| `lg` | 1024-1279px | Full sidebar + two-column content | Small desktop |
| `xl` | 1280px+ | Full sidebar + multi-column content | Desktop |

### 10.2 Sidebar Behavior

| Breakpoint | Sidebar State | Width | Nav Item Display |
|-----------|--------------|-------|------------------|
| `< 768px` | Hidden (off-screen drawer) | 0 → 260px opened | Icon + label |
| 768-1023px | Collapsed (icon-only) | 64px | Icon only, label on hover |
| ≥ 1024px | Expanded (default) | 260px | Icon + label |

**Mobile Drawer:**
- Trigger: Hamburger button in top bar
- Overlay covers content when open
- Swipe right to open, swipe left to close
- `aria-expanded` on hamburger button

### 10.3 Dashboard Responsive Adaptation

| Component | `xs` (<640px) | `sm` (640-767px) | `md` (768-1023px) | `lg+` (≥1024px) |
|-----------|---------------|-------------------|--------------------|------------------|
| **Summary cards** | 2-column grid, stacked | 2×2 grid | 4-column row | 4-column row |
| **Cost trend chart** | Full width, small | Full width | 2/3 width + side panel | Full width |
| **Request table** | Horizontal scroll wrapper | Horizontal scroll | Normal table | Normal table |
| **Provider list table** | Card view (1 per row) | Card view | Normal table | Normal table |
| **Model catalog** | Single column list | 2-col grid | 3-col grid | Table view |
| **Health cards** | 1 per row | 2 per row | 2 per row | 4 per row |
| **Add provider wizard** | Full screen modal | Full screen modal | Centered modal | Centered modal |

### 10.4 Responsive Component Patterns

| Component | Mobile (<768px) | Desktop (≥768px) |
|-----------|----------------|-------------------|
| **Data tables** | Horizontal scroll container with sticky first column | Normal table layout |
| **Modal dialogs** | Full-screen, 100% width/height | Centered, max 640px width |
| **Slide-over panels** | Full width (100vw) | Right 480px panel |
| **Wizard steps** | Vertical layout, full buttons | Horizontal stepper, compact |
| **Multi-select** | Native `<select multiple>` | Checkbox dropdown |
| **Date picker** | Native `<input type="date">` | Calendar widget |
| **Filter bars** | Collapsible accordion | Inline row |

### 10.5 Print Styles

- Hide sidebar, top bar, navigation
- Show main content full width
- Data tables print with all columns visible
- Charts print as static images (fallback to data tables)
- URL and timestamp printed in footer
- `@media print` stylesheet included

---

## 11. UX Acceptance Criteria

### 11.1 Global UX Acceptance Criteria

| # | Criteria | Validation Method | WCAG Ref |
|---|----------|-------------------|----------|
| UX-AC-01 | All pages load and display correctly in Chrome, Firefox, Safari, Edge (latest 2 versions) | Cross-browser visual inspection | — |
| UX-AC-02 | All interactive elements are reachable and operable via keyboard alone | Full keyboard tab-through of every page | 2.1.1 |
| UX-AC-03 | Visible focus indicator on all focusable elements (2px ring, 2px offset) | Tab through all pages, inspect focus ring | 2.4.7 |
| UX-AC-04 | All text meets WCAG AA contrast ratio (≥4.5:1 normal, ≥3:1 large) | Automated color contrast audit | 1.4.3 |
| UX-AC-05 | No information conveyed by color alone — all status indicators include icon + text label | Audit all badges, status dots | 1.4.1 |
| UX-AC-06 | All images/icons have meaningful `alt` text or `aria-label` | Screen reader audit | 1.1.1 |
| UX-AC-07 | Page title updates on route change (document.title) | Visual inspection of browser tab | 2.4.2 |
| UX-AC-08 | Skip-to-content link is first focusable element on every page | Keyboard tab test on page load | 2.4.1 |
| UX-AC-09 | `prefers-reduced-motion` is respected — all non-essential animations disabled | Enable OS reduce-motion setting, verify | 2.3.3 |
| UX-AC-10 | All touch targets are minimum 44×44px | Inspector measurement on interactive elements | 2.5.5 |
| UX-AC-11 | Page is responsive at 320px, 640px, 768px, 1024px, 1440px | Browser DevTools responsive mode | 1.4.10 |
| UX-AC-12 | All forms show inline validation errors with `aria-describedby` linking | Submit empty forms, inspect error messages | 3.3.1 |
| UX-AC-13 | All async operations show loading state (skeleton/spinner) within 200ms | Throttle network, observe loading states | 2.2.1 |
| UX-AC-14 | Empty states display helpful message + action button (if applicable) | Remove all providers/keys, check each page | — |
| UX-AC-15 | Error states display descriptive message + retry action | Simulate API failure, inspect error UI | 3.3.2 |
| UX-AC-16 | All interactive hover states have smooth 150-200ms CSS transition | Hover test on buttons, links, cards | — |
| UX-AC-17 | Sidebar navigation correctly highlights active route | Click each nav item, verify active state | — |
| UX-AC-18 | Breadcrumb or back navigation available on detail/edit pages | Navigate to /providers/add, verify back button | 2.4.8 |

### 11.2 Provider Configuration UI Acceptance Criteria

| # | Criteria | Validation Method |
|---|----------|-------------------|
| UX-AC-P1 | Provider list page loads within 2s with 10+ providers | Performance measurement |
| UX-AC-P2 | Provider list shows all columns: name, type, status, models, last check, actions | Visual inspection |
| UX-AC-P3 | "Add Provider" button is visible and functional in all states (empty + populated) | Visual inspection |
| UX-AC-P4 | Provider type selector grid displays 6 provider types with icons | Visual inspection |
| UX-AC-P5 | Clicking a provider type card highlights it and advances to next step | Click test |
| UX-AC-P6 | Connection form shows type-specific fields (Ollama: no API key; ChatGPT: cookie field) | Visual inspection of each type |
| UX-AC-P7 | API key field masks input by default, toggle button reveals it | Click 👁 toggle |
| UX-AC-P8 | "Test Connection" button calls provider and shows success/failure result within 5s | Test with valid + invalid keys |
| UX-AC-P9 | Model configuration step shows provider's available models with search/filter | Visual inspection |
| UX-AC-P10 | Custom model name mapping rows are addable and removable | Test add/remove mapping |
| UX-AC-P11 | Fallback configuration step allows drag-and-drop reordering (with keyboard alternative) | Drag test + keyboard test |
| UX-AC-P12 | Save button shows success toast and redirects to provider list | Test full flow |
| UX-AC-P13 | Edit provider form is pre-filled with existing values | Test edit flow |
| UX-AC-P14 | Delete provider shows confirmation dialog with provider name | Test delete flow |
| UX-AC-P15 | Provider config changes are reflected in API behavior without container restart | curl request after UI change |
| UX-AC-P16 | Provider form validates all required fields before submission with inline messages | Submit empty form |

### 11.3 Stats Dashboard Acceptance Criteria

| # | Criteria | Validation Method |
|---|----------|-------------------|
| UX-AC-D1 | Dashboard loads within 3s when stats data exists for 30+ days | Performance measurement |
| UX-AC-D2 | Summary cards show: Total Cost (month), Requests (today), Active Providers, Avg Latency | Visual inspection |
| UX-AC-D3 | Cost trend chart displays as multi-series line chart with provider color coding | Visual inspection |
| UX-AC-D4 | Time range selector changes all chart data without page reload | Test 7d → 30d → 90d |
| UX-AC-D5 | Cost by Provider horizontal bar chart sorts providers by cost descending | Visual inspection |
| UX-AC-D6 | Top Models by Cost shows correct top-10 models with cost values | Visual inspection |
| UX-AC-D7 | Detailed Breakdown table shows: provider, requests, input tokens, output tokens, cost, % total | Visual inspection |
| UX-AC-D8 | Clicking a provider row in breakdown drills down to per-model view | Click test |
| UX-AC-D9 | Tokens tab shows Input vs Output stacked area chart with correct aggregation | Visual inspection |
| UX-AC-D10 | Latency tab shows P50/P95/P99 multi-line chart per provider | Visual inspection |
| UX-AC-D11 | Requests tab shows infinite-scroll log with all columns sortable | Test scroll + column click |
| UX-AC-D12 | Request log rows are expandable to show full details (headers, body preview, fallback chain) | Click expand |
| UX-AC-D13 | Request log rows can be filtered by provider, model, status, and date range | Test each filter |
| UX-AC-D14 | Dashboard data refreshes automatically every 30s with visual indication | Wait 30s, observe |
| UX-AC-D15 | "No usage data yet" empty state shown when no requests processed | Clear stats, check page |
| UX-AC-D16 | Export CSV button downloads formatted CSV file with correct data | Test download |
| UX-AC-D17 | Chart tooltips show exact values on hover with provider color match | Hover over data points |

### 11.4 Model Catalog Acceptance Criteria

| # | Criteria | Validation Method |
|---|----------|-------------------|
| UX-AC-M1 | Model list shows all models from all configured providers (union) | Compare against provider configs |
| UX-AC-M2 | Each model row shows: name, provider badge, status, cost per token | Visual inspection |
| UX-AC-M3 | Search input filters model list in real-time as user types | Type partial model name |
| UX-AC-M4 | Provider filter chips toggle model visibility per provider | Click each provider chip |
| UX-AC-M5 | Status filter (All/Ready/Limited/Unavailable) works correctly | Test each filter option |
| UX-AC-M6 | Provider badges are color-coded by provider type | Visual inspection |
| UX-AC-M7 | Clicking model name opens detail slide-over with full info, usage, fallback chain | Click test |
| UX-AC-M8 | "Copy Model Name" button copies model name to clipboard | Click Copy, paste test |
| UX-AC-M9 | "Quick Test" button sends a test request and shows response preview | Click Quick Test |
| UX-AC-M10 | Empty state shown when no providers configured | Remove all providers, check page |
| UX-AC-M11 | Model count updates correctly when providers are added/removed | Add provider → check count |

### 11.5 API Key Management Acceptance Criteria

| # | Criteria | Validation Method |
|---|----------|-------------------|
| UX-AC-K1 | API key list shows all configured keys with name, masked key, status, usage | Visual inspection |
| UX-AC-K2 | "New Key" button opens modal dialog with form fields | Click test |
| UX-AC-K3 | Key generation modal validates: name (required), rate limit (number), budget (number) | Submit invalid data |
| UX-AC-K4 | Generated key is shown once in a highlighted banner with copy button | Test generate flow |
| UX-AC-K5 | Generated key has "or-" prefix format and minimum 32 chars | Visual inspection |
| UX-AC-K6 | Key list shows real-time usage data (requests, cost) for each key | Visual inspection |
| UX-AC-K7 | Revoke key shows confirmation dialog, key status changes to "Revoked" | Test revoke flow |
| UX-AC-K8 | Revoked keys remain in list but are clearly marked and non-functional | Test API call with revoked key |
| UX-AC-K9 | Copy button copies key to clipboard with success tooltip | Click Copy |
| UX-AC-K10 | Empty state shown when no keys configured | Remove all keys, check page |

### 11.6 Provider Health Status Acceptance Criteria

| # | Criteria | Validation Method |
|---|----------|-------------------|
| UX-AC-H1 | Health page shows all configured providers as status cards | Visual inspection |
| UX-AC-H2 | Status indicators: green (Online), yellow (Degraded), red (Offline), grey (Unknown) | Visual inspection |
| UX-AC-H3 | Each status card shows: provider name, status label, latency, request count, circuit state | Visual inspection |
| UX-AC-H4 | Auto-refresh toggle controls automatic polling (default: ON every 30s) | Toggle on/off, observe |
| UX-AC-H5 | Last checked timestamp updates on each refresh cycle | Wait 30s, observe timestamp |
| UX-AC-H6 | Adding a provider immediately adds its status card to health page | Add provider, check health page |
| UX-AC-H7 | Removing a provider removes its status card from health page | Remove provider, check health page |
| UX-AC-H8 | Empty state shown when no providers configured | Remove all providers, check page |

### 11.7 Accessibility-Specific Acceptance Criteria

| # | Criteria | Validation Method |
|---|----------|-------------------|
| UX-AC-A1 | Axe-core scan reports 0 critical or serious violations on every page | Run `test_automation_check_accessibility` on each route |
| UX-AC-A2 | Tab order follows visual layout: sidebar → main content → actions | Full keyboard walkthrough |
| UX-AC-A3 | All form inputs have associated `<label>` elements | HTML inspection |
| UX-AC-A4 | All error messages are linked to inputs via `aria-describedby` | Inspect invalid form state |
| UX-AC-A5 | All status changes are announced via `aria-live` region | Screen reader test |
| UX-AC-A6 | All modals trap focus and close on Esc key | Modal interaction test |
| UX-AC-A7 | All icon buttons have `aria-label` describing the action | Button audit |
| UX-AC-A8 | Zoom to 200% — no content cut off or overlapping | Browser zoom test |
| UX-AC-A9 | All data tables have proper `<th>` scope and caption | Table markup audit |
| UX-AC-A10 | Custom drag-and-drop has keyboard alternative (move up/down buttons) | Keyboard reorder test |

### 11.8 Responsive Acceptance Criteria

| # | Criteria | Validation Method |
|---|----------|-------------------|
| UX-AC-R1 | All pages render correctly at 375px (mobile) without horizontal scroll | DevTools 375px viewport |
| UX-AC-R2 | Sidebar collapses to icon-only at 768px, expands at 1024px | Resize browser, observe sidebar |
| UX-AC-R3 | Summary cards stack to 2-column grid below 768px | Resize dashboard page |
| UX-AC-R4 | Data tables show horizontal scroll on viewports < 768px | Resize provider list page |
| UX-AC-R5 | Add Provider wizard is full-screen on mobile, modal on desktop | Resize between breakpoints |
| UX-AC-R6 | All touch targets ≥ 44×44px on touch devices | Touch device test |
| UX-AC-R7 | Form inputs are full-width on mobile, constrained on desktop | Resize form pages |
| UX-AC-R8 | Charts resize proportionally on viewport change | Resize dashboard, observe charts |

### 11.9 Performance Acceptance Criteria

| # | Criteria | Validation Method |
|---|----------|-------------------|
| UX-AC-PF1 | Dashboard initial page load (with data) ≤ 3s | Chrome DevTools Network tab |
| UX-AC-PF2 | Provider list page load (10+ providers) ≤ 2s | Performance measurement |
| UX-AC-PF3 | Chart data refresh ≤ 1s (API response time for aggregated stats) | Network tab timing |
| UX-AC-PF4 | Form submission response (save provider) ≤ 2s | Network tab timing |
| UX-AC-PF5 | Search/filter in model catalog responds within 200ms of last keystroke | Type test |
| UX-AC-PF6 | Page transitions feel instant (≤ 300ms) | Navigation timing |

---

## 12. Traceability Links

### 12.1 Entity Records

| Entity ID | Type | Title | Status |
|-----------|------|-------|--------|
| `8b4a1870-aa6b-4e34-b4f7-cbbae43eac35` | epic | OllaRouter USP Definition and Backlog | defined |
| *(new)* | epic | EPIC-4: Cost Tracking & Stats Dashboard UX Design | design |
| *(new)* | ux_review | Phase 4 UX Design (Pre-Build) OllaRouter | complete |

### 12.2 Link Map

```
[8b4a1870] OllaRouter USP Definition
    └── documents ◀── [this document] Phase 4 UX Design
        ├── documents ◀── Overview Dashboard Screen (Screen 1)
        ├── documents ◀── Provider List Screen (Screen 2)
        ├── documents ◀── Add Provider Form Screen (Screen 3)
        ├── documents ◀── Stats Dashboard Screen (Screen 4)
        ├── documents ◀── Model Catalog Screen (Screen 5)
        ├── documents ◀── API Key Management Screen (Screen 6)
        ├── documents ◀── Health Status Screen (Screen 7)
        └── documents ◀── Settings Screen (Screen 8)
```

### 12.3 Feature-to-Screen Mapping

| EPIC | Feature / FR | Primary Screens | UX Phase |
|------|-------------|-----------------|----------|
| EPIC-1 | FR-1.x (Ollama API Core) | N/A (API only) | — |
| EPIC-2 | FR-5.x (Provider Configuration UI) | Screen 2, Screen 3, Screen 8 | UX-1 |
| EPIC-2 | FR-8.2 (Provider Health) | Screen 7 | UX-5 |
| EPIC-3 | FR-7.x (API Key Auth) | Screen 6 | UX-4 |
| EPIC-4 | FR-4.x (Cost Tracking) | Screen 1, Screen 4 | UX-2 |
| EPIC-4 | FR-6.x (Stats Dashboard) | Screen 1, Screen 4 | UX-2 |
| EPIC-6 | FR-8.x (Health/Metrics) | Screen 7 | UX-5 |
| EPIC-7 | FR-1.3 (Model Catalog UI) | Screen 5 | UX-3 |

### 12.4 Component-to-EPIC Mapping

| Component | EPIC | Priority | UI Impact |
|-----------|------|----------|-----------|
| SummaryCardsRow | EPIC-4 | P1 | New |
| CostTrendChart | EPIC-4 | P1 | New |
| RecentRequestsTable | EPIC-4 | P1 | New |
| ProviderTable | EPIC-2 | P1 | New |
| ProviderForm (Wizard) | EPIC-2 | P1 | New |
| ProviderTypeSelector | EPIC-2 | P1 | New |
| TestConnectionButton | EPIC-2 | P1 | New |
| StatusBadge | EPIC-2 | P1 | New |
| ModelTable | EPIC-7 | P2 | New |
| ModelDetailSlideOver | EPIC-7 | P2 | New |
| FallbackChainConfig | EPIC-2 | P1 | New |
| ApiKeyTable | EPIC-3 | P1 | New |
| CreateKeyDialog | EPIC-3 | P1 | New |
| HealthCards | EPIC-2/6 | P1 | New |
| SettingsForm | EPIC-2/6 | P2 | New |
| YamlPreview | EPIC-2/6 | P2 | New |
| Sidebar (shared) | All | P0 | New |
| TopBar (shared) | All | P0 | New |
| ToastNotification (shared) | All | P1 | New |
| ConfirmDialog (shared) | All | P1 | New |

---

## 13. Design Tokens Summary (CSS Custom Properties)

```css
:root {
  /* Colors */
  --color-primary: #2563EB;
  --color-primary-hover: #1D4ED8;
  --color-secondary: #64748B;
  --color-accent: #7C3AED;
  --color-success: #10B981;
  --color-success-dark: #059669;
  --color-error: #EF4444;
  --color-warning: #F59E0B;
  --color-info: #3B82F6;
  --color-bg: #F8FAFC;
  --color-bg-card: #FFFFFF;
  --color-bg-sidebar: #1E293B;
  --color-text: #1E293B;
  --color-text-secondary: #475569;
  --color-text-inverse: #F8FAFC;
  --color-border: #E2E8F0;

  /* Provider colors (consistent across charts & badges) */
  --color-provider-openai: #2563EB;
  --color-provider-anthropic: #7C3AED;
  --color-provider-google: #059669;
  --color-provider-ollama: #F59E0B;
  --color-provider-chatgpt: #EF4444;

  /* Typography */
  --font-family: 'Plus Jakarta Sans', system-ui, -apple-system, sans-serif;
  --font-size-xs: 0.75rem;    /* 12px */
  --font-size-sm: 0.875rem;   /* 14px */
  --font-size-base: 1rem;     /* 16px */
  --font-size-lg: 1.125rem;   /* 18px */
  --font-size-xl: 1.5rem;     /* 24px */
  --font-size-2xl: 1.875rem;  /* 30px */
  --font-size-3xl: 2.25rem;   /* 36px */

  /* Spacing */
  --space-xs: 0.25rem;  /* 4px */
  --space-sm: 0.5rem;   /* 8px */
  --space-md: 1rem;     /* 16px */
  --space-lg: 1.5rem;   /* 24px */
  --space-xl: 2rem;     /* 32px */
  --space-2xl: 3rem;    /* 48px */

  /* Layout */
  --sidebar-width: 260px;
  --sidebar-collapsed-width: 64px;
  --topbar-height: 56px;
  --max-content-width: 1280px;

  /* Shadows */
  --shadow-sm: 0 1px 2px rgba(0,0,0,0.05);
  --shadow-md: 0 4px 6px -1px rgba(0,0,0,0.1);
  --shadow-lg: 0 10px 15px -3px rgba(0,0,0,0.1);

  /* Border Radius */
  --radius-sm: 4px;
  --radius-md: 6px;
  --radius-lg: 8px;
  --radius-xl: 12px;

  /* Transitions */
  --transition-fast: 150ms ease-in-out;
  --transition-normal: 200ms ease-in-out;
  --transition-slow: 300ms ease;
}
```

---

## 14. Artifact Summary

| Artifact | Path | Description |
|----------|------|-------------|
| Phase 4 UX Design (this file) | `docs/product/traceability/checkpoints/04-ux-design.md` | Complete pre-implementation UX design for OllaRouter |
| UX Design artifact | `docs/product/ux-design/04-ollarouter-ux-design.md` | Symlink/copy of this document for UX directory |
| Phase 1 Market Research | `docs/product/traceability/checkpoints/01-market-research.md` | Market size, trends, user needs, gap analysis |
| Phase 2 Competitor Analysis | `docs/product/traceability/checkpoints/02-competitor-analysis.md` | 10-competitor deep comparison matrix |
| Phase 3 USP Definition | `docs/product/traceability/checkpoints/03-usp-definition.md` | USP, requirements, EPICs, personas |
| Design System Source | ux-intelligence MCP | SaaS Platform, Professional mood, Data-Dense Dashboard style |

---

## 15. Next Steps

After this UX design phase, the following phases are ready to proceed:

1. **EPIC-2: Provider Configuration UI** — Has UX-1 (Provider Config UI) and UX-5 (Health Status) design complete. Ready for story breakdown and architecture review.
2. **EPIC-4: Cost Tracking & Dashboard** — Has UX-2 (Stats Dashboard) design complete. Ready for story breakdown and architecture review.
3. **EPIC-3: Auth & Security** — Has UX-4 (API Key Management) design complete. Ready for story breakdown.
4. **EPIC-7: Model Catalog** — Has UX-3 (Model Catalog) design complete. Ready for story breakdown (P2 — defer to v1.2).

**Recommended build order for UI:**
1. Sidebar + TopBar + shared component library (Toast, ConfirmDialog, Badge)
2. Provider List page (card/table with empty state)
3. Add/Edit Provider Wizard (all 4 steps)
4. Overview Dashboard (summary cards + recent requests)
5. Stats Dashboard (all 4 tabs with charts)
6. Model Catalog (table + detail slide-over)
7. API Key Management (list + create dialog)
8. Health Status (provider status cards)
9. Settings (general + defaults + YAML preview)

---

*End of Phase 4 — UX Design (Pre-Build). Ready for Phase 5 EPIC → Feature → Story creation.*

