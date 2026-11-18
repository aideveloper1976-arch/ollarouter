# Documentation

| Page | Read it for |
|---|---|
| [Getting started](getting-started.md) | Running from source or Docker Compose, connecting clients, running tests |
| [Configuration](configuration.md) | Every setting, provider type, virtual models, environment variables |
| [API reference](api.md) | Ollama, OpenAI, health, metrics and admin endpoints |
| [Architecture](architecture.md) | Request flow, components, design decisions, deployment |
| [Guardrails](guardrails.md) | How broken tool calls from local models are repaired |
| [Verification](verification.md) | What is tested, how, and what is not yet verified |

## SDLC evidence

OllaRouter was built with [Agentic SDLC](https://github.com/aideveloper1976-arch/agentic-sdlc).
These folders are the unedited output of those runs. Only one private Confluence link was
redacted. They are kept as a record of the process, so some details (versions, counts, plans)
describe the project at that time rather than now.

| Folder | Contents |
|---|---|
| [`product/market-research`](product/market-research) | Market research |
| [`product/product-analysis`](product/product-analysis) | USP definition |
| [`product/evidence`](product/evidence) | LiteLLM comparison |
| [`product/ux-design`](product/ux-design) | Admin console UX design |
| [`product/backlog`](product/backlog) | Epics, features and stories |
| [`product/architecture`](product/architecture) | System, adapter contract and Docker architecture |
| [`product/implementation-plan`](product/implementation-plan) | Master implementation plan |
| [`product/traceability`](product/traceability) | Gate checkpoints 1–18 and the full traceability report |
| [`09_jira_pack`](09_jira_pack) | Browser UI regression test script |
| [`../ollarouter/docs`](../ollarouter/docs) | Second run: guardrails and upstream OAuth plan and checkpoints |
