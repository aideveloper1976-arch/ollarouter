# OllaRouter — UX Design Artifact

> This artifact is a copy/link to the canonical UX Design checkpoint.
>
> **Canonical document:** `docs/product/traceability/checkpoints/04-ux-design.md`
>
> **Purpose:** This file exists so that the `docs/product/ux-design/` directory contains
> the pre-implementation UX design output for OllaRouter. All detailed content (screens,
> component hierarchy, user flows, accessibility plan, responsive behavior, UX acceptance
> criteria) is in the canonical checkpoint.

## Quick Reference

| Attribute | Value |
|-----------|-------|
| Product | OllaRouter — Ollama-native Multi-Provider LLM Gateway |
| Phase | 4 — UX Design (Pre-Build) |
| Date | 2026-05-20 |
| Agent | product-analyst |
| Status | Complete |

## UX Design Outputs

| Output | Description | Location in Canonical Doc |
|--------|-------------|--------------------------|
| Design System Foundation | Colors, typography, spacing, shadows, animations | §1 |
| UI Information Architecture | Site map, navigation structure, page-to-EPIC mapping | §2 |
| Screen Inventory & Layouts | 8 screen mockups with ASCII art | §3 |
| Component Hierarchy | Full component tree with states, shared components | §4 |
| User Flows | 6 complete user journeys (deploy, provider setup, dashboard, fallback, ChatGPT, API keys) | §5 |
| Provider Setup UX | Provider type cards, conditional form fields, test connection, status indicators | §6 |
| Token/Cost Stats Dashboard UX | Summary cards, chart specs, request log table, data refresh strategy | §7 |
| Model & Fallback Configuration UX | Fallback chain config, model catalog filter/search, model detail slide-over | §8 |
| Accessibility Plan | WCAG 2.1 AA/AAA compliance, color contrast, keyboard nav, screen reader, motion | §9 |
| Responsive Behavior | Breakpoints, sidebar behavior, dashboard adaptation, component patterns, print styles | §10 |
| UX Acceptance Criteria | 17 global + ~80 feature-specific criteria across all 8 screens | §11 |
| Design Tokens | CSS custom properties for colors, typography, spacing, shadows | §13 |

## Design System Source

- **Style:** Data-Dense Dashboard + Minimal & Direct
- **Colors:** Professional blues (`#2563EB` primary) on `#F8FAFC` background
- **Typography:** Plus Jakarta Sans (Google Fonts)
- **Icons:** Lucide SVG icons
- **Framework:** shadcn/ui + Tailwind CSS
- **Anti-patterns avoided:** Information overload, no clear hierarchy

## Traceability

| Phase | Document |
|-------|----------|
| Phase 1 | `docs/product/traceability/checkpoints/01-market-research.md` |
| Phase 2 | `docs/product/traceability/checkpoints/02-competitor-analysis.md` |
| Phase 3 | `docs/product/traceability/checkpoints/03-usp-definition.md` |
| **Phase 4** | **`docs/product/traceability/checkpoints/04-ux-design.md` ← this** |

## Next Steps

1. Phase 5: EPIC → Feature → Story creation in Jira project
2. Phase 6: Architecture Review — data model, API contracts, component integration
3. Phase 7: Implementation Plan — build order, files to create/modify, dependencies

