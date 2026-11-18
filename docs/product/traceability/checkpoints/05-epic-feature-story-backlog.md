# Phase 5 — EPIC → Feature → Story Backlog: OllaRouter

**Date:** 2026-05-20  
**Agent:** story-writer  
**Project:** ollama-stack (C:\Projects\ollama-stack)  
**Phase 1 Reference:** `docs/product/traceability/checkpoints/01-market-research.md`  
**Phase 2 Reference:** `docs/product/traceability/checkpoints/02-competitor-analysis.md`  
**Phase 3 Reference:** `docs/product/traceability/checkpoints/03-usp-definition.md`  
**Phase 4 Reference:** `docs/product/traceability/checkpoints/04-ux-design.md`  
**Jira Project:** ACP (AML/CTF/PF)

---

## Summary

The OllaRouter backlog has been structured into **7 EPICs**, **24 Features**, and **56 Stories**:

| EPIC | Title | Jira Key | Priority |
|------|-------|----------|----------|
| EPIC-1 | Ollama API Core — Multi-Provider Request Routing | ACP-266 | P0 (MVP) |
| EPIC-2 | Provider Configuration Web UI | ACP-267 | P1 (v1.0) |
| EPIC-3 | API Key Authentication & Rate Limiting | ACP-268 | P1 (v1.0) |
| EPIC-4 | Usage Tracking & Stats Dashboard | ACP-269 | P1 (v1.0) |
| EPIC-5 | ChatGPT Plus Web Proxy Provider | ACP-270 | P1 (v1.1) |
| EPIC-6 | Prometheus Metrics, Health Endpoints & Structured Logging | ACP-271 | P1 (v1.1) |
| EPIC-7 | Enhanced Model Discovery & Extended Ollama Endpoints | ACP-272 | P2 (v1.2) |

### Jira Items Created

**31 Jira issues created in project ACP:**
- 7 Epics (ACP-266 through ACP-272)
- 24 Features/Tasks (ACP-273 through ACP-296)

### Local Backlog File

`docs/product/backlog/ollarouter-backlog.md` — Complete backlog with:
- Full EPIC definitions with business outcomes and success metrics
- Feature objectives with scope boundaries
- 56 Stories with GIVEN/WHEN/THEN acceptance criteria, edge cases, backend/frontend notes, test notes, security notes, and traceability links
- Open questions with suggested resolutions
- Traceability matrix linking all entities to evidence

---

## Key Architecture Decisions (from research)

| Decision | Rationale | Source |
|----------|-----------|--------|
| **Go language** for single binary | <50MB container, ms startup, cross-compile | Competitor analysis §4, §11 |
| **Ollama API surface** (not OpenAI) | Core differentiator — no competitor does this | Market research §4, USP §5 |
| **NDJSON streaming** | Ollama protocol — must match exactly | Market research §7 |
| **Zero external DB** | No PostgreSQL/Redis — stateless design | Market research §8, USP §5 |
| **Embedded web UI** | Single binary — admin UI built into Go binary | USP §9 (Q5) |
| **YAML config + env vars** | Docker-native pattern, consistent with competitors | Market research §5 |
| **Default port 11435** | Alongside Ollama (11434), not replacing it | USP §9 (Q1) |

---

## Traceability Links

```
[c843c293] OllaRouter Idea
    ├── derived_from ◀── [4dbc9a4b] Phase 1 Market Research
    └── derived_from ◀── [c5aaf9f3] Phase 2 Competitor Analysis

[8b4a1870] OllaRouter USP Definition
    └── derived_from ◀── [c843c293] OllaRouter Idea

[EPIC-1] b500772a Ollama API Core
    └── derived_from ◀── [8b4a1870] USP Definition
    ├── decomposes_to ▸ FEAT-1.1 (ACP-273)
    ├── decomposes_to ▸ FEAT-1.2 (ACP-274)
    ├── decomposes_to ▸ FEAT-1.3 (ACP-275)
    ├── decomposes_to ▸ FEAT-1.4 (ACP-276)
    ├── decomposes_to ▸ FEAT-1.5 (ACP-277)
    ├── decomposes_to ▸ FEAT-1.6 (ACP-278)
    └── decomposes_to ▸ FEAT-1.7 (ACP-279)

[EPIC-2] 5987bb40 Provider Configuration Web UI
    └── derived_from ◀── [8b4a1870] USP Definition
    ├── decomposes_to ▸ FEAT-2.1 (ACP-280)
    ├── decomposes_to ▸ FEAT-2.2 (ACP-281)
    ├── decomposes_to ▸ FEAT-2.3 (ACP-282)
    └── decomposes_to ▸ FEAT-2.4 (ACP-283)

[EPIC-3] 66fffa17 API Key Auth
    └── derived_from ◀── [8b4a1870] USP Definition
    ├── decomposes_to ▸ FEAT-3.1 (ACP-284)
    ├── decomposes_to ▸ FEAT-3.2 (ACP-285)
    └── decomposes_to ▸ FEAT-3.3 (ACP-286)

[EPIC-4] daac4b32 Cost Tracking
    └── derived_from ◀── [8b4a1870] USP Definition
    ├── decomposes_to ▸ FEAT-4.1 (ACP-287)
    ├── decomposes_to ▸ FEAT-4.2 (ACP-288)
    └── decomposes_to ▸ FEAT-4.3 (ACP-289)

[EPIC-5] 41cc67fa ChatGPT Plus
    └── derived_from ◀── [8b4a1870] USP Definition
    ├── decomposes_to ▸ FEAT-5.1 (ACP-290)
    └── decomposes_to ▸ FEAT-5.2 (ACP-291)

[EPIC-6] 96cb8b8c Observability
    └── derived_from ◀── [8b4a1870] USP Definition
    ├── decomposes_to ▸ FEAT-6.1 (ACP-292)
    ├── decomposes_to ▸ FEAT-6.2 (ACP-293)
    └── decomposes_to ▸ FEAT-6.3 (ACP-294)

[EPIC-7] cc2223e4 Model Discovery
    └── derived_from ◀── [8b4a1870] USP Definition
    ├── decomposes_to ▸ FEAT-7.1 (ACP-295)
    └── decomposes_to ▸ FEAT-7.2 (ACP-296)
```

---

## Artifacts

| File | Description |
|------|-------------|
| `docs/product/traceability/checkpoints/05-epic-feature-story-backlog.md` | **This file** — Phase 5 checkpoint |
| `docs/product/backlog/ollarouter-backlog.md` | Complete backlog with all EPICs, Features, Stories, and acceptance criteria |
| Jira ACP-266 through ACP-296 | 31 Jira issues created (7 Epics + 24 Features) |

---

## Next Phase

**Phase 6: Architecture Review** — Review data model, API contracts, component integration, and security for the OllaRouter Go implementation.

---

*End of Phase 5 — EPIC/Feature/Story Backlog. Ready for Phase 6 Architecture Review.*
