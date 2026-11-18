# OllaRouter — EPIC / Feature / Story Backlog

**Generated:** 2026-05-20  
**Phase:** 5 — EPIC → Feature → Story  
**Project:** ollama-stack (C:\Projects\ollama-stack)  
**Jira Project:** ACP (AML/CTF/PF)

---

## Backlog Summary

| EPIC | Title | Priority | Jira Key | Features | Stories |
|------|-------|----------|----------|----------|---------|
| EPIC-1 | Ollama API Core — Multi-Provider Request Routing | P0 (MVP) | ACP-266 | 7 | 14 |
| EPIC-2 | Provider Configuration Web UI | P1 (v1.0) | ACP-267 | 4 | 10 |
| EPIC-3 | API Key Authentication & Rate Limiting | P1 (v1.0) | ACP-268 | 3 | 7 |
| EPIC-4 | Usage Tracking & Stats Dashboard | P1 (v1.0) | ACP-269 | 3 | 9 |
| EPIC-5 | ChatGPT Plus Web Proxy Provider | P1 (v1.1) | ACP-270 | 2 | 5 |
| EPIC-6 | Prometheus Metrics, Health Endpoints & Structured Logging | P1 (v1.1) | ACP-271 | 3 | 6 |
| EPIC-7 | Enhanced Model Discovery & Extended Ollama Endpoints | P2 (v1.2) | ACP-272 | 2 | 5 |

**Total: 7 EPICs, 24 Features, 56 Stories**

---

## EPIC-1: Ollama API Core — Multi-Provider Request Routing

**Jira:** ACP-266  
**Priority:** P0 — MVP  
**Depends On:** None  
**Business Outcome:** First Ollama-native multi-provider gateway — users can route requests from any Ollama-compatible client to OpenAI, Anthropic, local Ollama, and OpenAI-compatible providers.  
**Exit Criteria:** Open WebUI can chat through OllaRouter using OpenAI models and local Ollama models.  
**User Story:** "As an Open WebUI user, I want to set OLLAMA_BASE_URL to OllaRouter and chat with GPT-4o through my existing interface, so I don't need to switch tools or manage multiple endpoints."  
**Evidence:** docs/product/traceability/checkpoints/01-market-research.md, docs/product/traceability/checkpoints/02-competitor-analysis.md

### FEAT-1.1: POST /api/chat Endpoint with NDJSON Streaming

**Jira:** ACP-273  
**Objective:** Implement the Ollama-compatible POST /api/chat endpoint with NDJSON streaming

#### Stories

##### STORY-1.1.1: Request Parsing & Model Routing for /api/chat
**Goal:** Parse Ollama-format chat requests and route to the correct provider backend.

**Description:** Implement request handler for POST /api/chat that parses Ollama-format JSON body (model, messages, stream, options), extracts the model name, looks up the configured provider for that model, and routes the request accordingly.

**Acceptance Criteria:**
- GIVEN a valid POST /api/chat request with model "gpt-4o" WHEN the request is received THEN OllaRouter parses the Ollama-format body and routes it to the configured OpenAI provider
- GIVEN a request with stream=true WHEN processed THEN the response Content-Type is "application/x-ndjson"
- GIVEN a request with stream=false WHEN processed THEN the response Content-Type is "application/json"
- GIVEN an unknown model name WHEN a request is received THEN OllaRouter returns 404 with {"error":"model \"xyz\" not found"}
- GIVEN a request with invalid JSON body WHEN received THEN OllaRouter returns 400 with {"error":"invalid request body"}

**Edge Cases:**
- Model name with provider prefix (e.g., "openai/gpt-4o", "anthropic/claude-sonnet-4")
- Messages array with single message vs multi-turn conversation
- Missing required fields (model, messages)
- Options field with temperature, top_p, max_tokens, etc.
- Very long conversation history (>100 messages)

**Backend Notes:**
- Implement in Go using net/http or chi router
- Parse Ollama chat request format as defined in Ollama API spec
- Extract model name, strip provider prefix if present
- Pass raw messages to downstream provider translator

**Frontend Notes:** N/A (API-only)

**Data Impact:** None (stateless proxy)

**Test Notes:**
- Unit tests for request parsing with valid/invalid bodies
- Integration tests routing to mock providers
- Streaming response format verification

**Security Notes:** Model names validated against allowlist of configured models

**Traceability:** EPIC-1 (ACP-266) → FEAT-1.1 (ACP-273)

##### STORY-1.1.2: NDJSON Streaming Response for /api/chat
**Goal:** Stream Ollama-format NDJSON responses for streaming chat requests.

**Description:** Implement NDJSON streaming for POST /api/chat with stream=true. Each line is a JSON object following Ollama streaming format: {"model":"gpt-4o","created_at":"...","message":{"role":"assistant","content":"Hello"},"done":false}. Final line has done:true with full statistics.

**Acceptance Criteria:**
- GIVEN a streaming chat request WHEN the provider returns SSE THEN OllaRouter translates each SSE data event to an NDJSON line
- GIVEN a streaming chat request WHEN complete THEN the last line has "done":true with "total_duration", "load_duration", "prompt_eval_count", "eval_count" statistics
- GIVEN a streaming chat request WHEN the connection drops THEN OllaRouter closes the response cleanly
- GIVEN a streaming request with a provider error mid-stream WHEN the error occurs THEN OllaRouter sends an error NDJSON line and closes

**Edge Cases:**
- Provider returns empty content chunks
- Provider returns multiple content delta chunks per SSE event
- Very long responses (>10K tokens)
- Client disconnects mid-stream (context cancellation)

**Backend Notes:**
- Translate OpenAI SSE (data: {...} events) to Ollama NDJSON (one JSON per line)
- Use Go's http.Flusher for streaming
- Handle context cancellation for client disconnect
- Set Transfer-Encoding: chunked

**Data Impact:** None

**Test Notes:**
- Test with mock SSE provider returning various patterns
- Verify each NDJSON line is valid JSON
- Test client disconnect handling

**Traceability:** EPIC-1 (ACP-266) → FEAT-1.1 (ACP-273)

### FEAT-1.2: POST /api/generate Endpoint with NDJSON Streaming

**Jira:** ACP-274  
**Objective:** Implement the Ollama-compatible POST /api/generate endpoint

#### Stories

##### STORY-1.2.1: Request Parsing & Model Routing for /api/generate
**Goal:** Parse Ollama-format generate requests and route to providers.

**Description:** Implement POST /api/generate handler that parses Ollama generate format (prompt, model, stream, options, images, system, template) and routes to the correct provider. Translate generate format to provider's chat/completions format when needed.

**Acceptance Criteria:**
- GIVEN a valid POST /api/generate request WHEN received THEN it is parsed and routed to the correct provider
- GIVEN a request with images (base64) WHEN the provider supports vision THEN images are included in the provider request
- GIVEN a request with stream=true WHEN processed THEN NDJSON streaming response is returned
- GIVEN a non-streaming generate request WHEN complete THEN single JSON response is returned with "response" field

**Edge Cases:**
- Generate without prompt (should return 400)
- Provider doesn't support generate (completions) — must translate to chat format
- Multiple images in a single request
- Very long prompts (>100K tokens)

**Traceability:** EPIC-1 (ACP-266) → FEAT-1.2 (ACP-274)

##### STORY-1.2.2: Generate Format Translation
**Goal:** Translate between Ollama generate format and provider API formats.

**Description:** Implement bidirectional translation between Ollama generate format and OpenAI/Anthropic API formats. Map prompt to system+user messages, map images to vision API format, map options to provider parameters.

**Acceptance Criteria:**
- GIVEN an Ollama generate request with system prompt WHEN translated THEN system message is set on the provider request
- GIVEN an Ollama generate request with images WHEN translated THEN images are included in vision API format
- GIVEN a provider response WHEN translated back to Ollama format THEN the "response" field contains the generated text
- GIVEN Ollama options (temperature, top_p, top_k, etc.) WHEN available THEN they are mapped to provider equivalent parameters

**Edge Cases:**
- Options not supported by target provider (silently dropped)
- Provider returns multiple choices — use first one
- Provider returns tool_calls in chat format — simplified in generate response

**Traceability:** EPIC-1 (ACP-266) → FEAT-1.2 (ACP-274)

### FEAT-1.3: GET /api/tags with Aggregated Model Listing

**Jira:** ACP-275  
**Objective:** Implement GET /api/tags returning unified model list from all providers

#### Stories

##### STORY-1.3.1: Local Ollama Model Discovery
**Goal:** Auto-discover models from the local Ollama instance.

**Description:** Periodically query the local Ollama /api/tags endpoint to discover available local models. Cache the results and merge into the aggregated model list.

**Acceptance Criteria:**
- GIVEN a running Ollama instance WHEN OllaRouter starts THEN it queries Ollama's /api/tags and caches the model list
- GIVEN Ollama models change WHEN OllaRouter polls (every 60s) THEN the cached list is updated
- GIVEN Ollama is unreachable WHEN polled THEN OllaRouter continues with last known model list and marks Ollama as degraded

**Edge Cases:**
- Ollama not configured (no local models expected)
- Ollama starts after OllaRouter — auto-detection on retry
- 200+ local models

**Traceability:** EPIC-1 (ACP-266) → FEAT-1.3 (ACP-275)

##### STORY-1.3.2: Aggregated Model List Response
**Goal:** Return unified /api/tags response merging local and cloud models.

**Description:** GET /api/tags handler that merges the local Ollama model list with configured cloud provider models. Returns standard Ollama format: {"models":[{"name":"provider/model","modified_at":"...","size":0,...}]}. Cloud models show size=0.

**Acceptance Criteria:**
- GIVEN configured OpenAI provider with gpt-4o and gpt-4o-mini WHEN /api/tags is called THEN response includes "openai/gpt-4o" and "openai/gpt-4o-mini"
- GIVEN local Ollama with "llama3" model WHEN /api/tags is called THEN response includes "ollama/llama3" (or local model name)
- GIVEN both local and cloud models WHEN /api/tags is called THEN response contains the union of all models
- GIVEN no providers configured WHEN /api/tags is called THEN response returns empty models array

**Edge Cases:**
- Model name collision (local and cloud have same name) — prefix with provider
- Very large model count (>100 models)
- Cache staleness — models shown that are no longer available

**Traceability:** EPIC-1 (ACP-266) → FEAT-1.3 (ACP-275)

### FEAT-1.4: Provider Routing Engine

**Jira:** ACP-276  
**Objective:** Core provider routing engine with YAML config

#### Stories

##### STORY-1.4.1: YAML Provider Configuration Parser
**Goal:** Parse providers.yaml with env var interpolation.

**Description:** Implement YAML configuration file parser that supports provider definitions with name, type, api_key (env var ref), base_url, models list, and model name mappings. Support ${ENV_VAR} interpolation for API keys and sensitive values.

**Acceptance Criteria:**
- GIVEN a providers.yaml with 3 providers WHEN parsed THEN all 3 are loaded into runtime config
- GIVEN ${OPENAI_API_KEY} in YAML WHEN env var is set THEN it is interpolated with the env var value
- GIVEN ${MISSING_VAR} in YAML WHEN env var is not set THEN an error is logged and startup continues with other providers
- GIVEN invalid YAML syntax WHEN parsed THEN an error is returned with file/line info

**Edge Cases:**
- Empty providers.yaml (no providers configured)
- Duplicate provider names — last one wins with warning
- YAML anchors and aliases (advanced YAML features)
- Very large config file (>50 providers)

**Traceability:** EPIC-1 (ACP-266) → FEAT-1.4 (ACP-276)

##### STORY-1.4.2: OpenAI Provider Adapter
**Goal:** Route requests to OpenAI via compatible API.

**Description:** Implement OpenAI provider adapter that translates Ollama requests to OpenAI API format (/v1/chat/completions). Handle streaming and non-streaming. Support organization ID and custom base URL.

**Acceptance Criteria:**
- GIVEN a chat request for "gpt-4o" WHEN routed to OpenAI THEN the request is sent to https://api.openai.com/v1/chat/completions
- GIVEN streaming mode WHEN using OpenAI THEN SSE responses are translated to NDJSON
- GIVEN organization ID configured WHEN request is sent THEN OpenAI-Organization header is included
- GIVEN custom base URL configured WHEN request is sent THEN the custom URL is used instead of default

**Edge Cases:**
- OpenAI API returns error with structured error body
- OpenAI rate limit (429) — passed to FEAT-1.5 for retry
- OpenAI model not found (404) — returned as Ollama error

**Traceability:** EPIC-1 (ACP-266) → FEAT-1.4 (ACP-276)

##### STORY-1.4.3: Anthropic Provider Adapter
**Goal:** Route requests to Anthropic via Messages API.

**Description:** Implement Anthropic provider adapter that translates Ollama requests to Anthropic Messages API format. Handle streaming with SSE translation.

**Acceptance Criteria:**
- GIVEN a chat request for "claude-sonnet-4" WHEN routed to Anthropic THEN the request is sent to https://api.anthropic.com/v1/messages
- GIVEN streaming mode WHEN using Anthropic THEN SSE events (content_block_delta, message_delta) are translated to NDJSON
- GIVEN Anthropic API version header WHEN required THEN anthropic-version header is included

**Edge Cases:**
- Anthropic requires specific message roles (user/assistant) — validate
- Anthropic content block types (text, tool_use, tool_result)
- Anthropic rate limit headers

**Traceability:** EPIC-1 (ACP-266) → FEAT-1.4 (ACP-276)

##### STORY-1.4.4: OpenAI-Compatible Provider Adapter
**Goal:** Route requests to any OpenAI-compatible API endpoint.

**Description:** Implement generic OpenAI-compatible provider adapter that works with any API following the OpenAI chat completions format (Azure, Together AI, Groq, vLLM, etc.). Support custom base URL configuration.

**Acceptance Criteria:**
- GIVEN a configured OpenAI-compatible provider with custom URL WHEN routed THEN the request goes to the custom URL
- GIVEN Azure OpenAI with deployment name WHEN configured THEN the deployment name is used in the URL path
- GIVEN a provider that doesn't support streaming WHEN stream=true THEN fall back to non-streaming and buffer the response

**Edge Cases:**
- Provider requires different auth header format
- Provider returns non-standard error format
- Provider has different model naming conventions

**Traceability:** EPIC-1 (ACP-266) → FEAT-1.4 (ACP-276)

### FEAT-1.5: Fallback/Failover on 429/5xx

**Jira:** ACP-277  
**Objective:** Automatic failover and retry with circuit breaker

#### Stories

##### STORY-1.5.1: Retry with Exponential Backoff
**Goal:** Auto-retry on 429 and 5xx with configurable backoff.

**Description:** Implement retry logic for provider requests. Retry on 429 (rate limited) with exponential backoff (base delay 1s, multiply by 2 each retry, configurable max retries). Retry on 5xx with configurable max retries and shorter delay.

**Acceptance Criteria:**
- GIVEN a request that returns 429 WHEN retries are configured THEN OllaRouter waits 1s, retries, then 2s, then 4s (max 3 retries default)
- GIVEN a request that returns 503 WHEN retries are configured THEN OllaRouter retries with 500ms delay (max 2 retries default)
- GIVEN all retries exhausted WHEN the provider still returns error THEN fallback chain is triggered
- GIVEN max retries set to 0 WHEN a provider error occurs THEN no retry is attempted

**Edge Cases:**
- Provider returns 429 with Retry-After header — use that value
- Interleaving 429 and 5xx responses during retry sequence
- Network timeout during retry — counted as failed attempt

**Traceability:** EPIC-1 (ACP-266) → FEAT-1.5 (ACP-277)

##### STORY-1.5.2: Fallback Chain Execution
**Goal:** Execute per-model fallback chain on provider failure.

**Description:** Implement fallback chain execution that tries providers in configured order. On failure (all retries exhausted), move to next provider in the chain. Add x-ollarouter-provider response header showing which provider served the request.

**Acceptance Criteria:**
- GIVEN a fallback chain [OpenAI → Anthropic → Ollama] WHEN OpenAI fails AFTER retries THEN the request is sent to Anthropic
- GIVEN Anthropic also fails WHEN all providers in the chain are exhausted THEN the last error is returned to the client
- GIVEN a successful fallback WHEN the response is returned THEN x-ollarouter-provider header shows the fallback provider name
- GIVEN no fallback configured WHEN the primary provider fails THEN error is returned immediately

**Edge Cases:**
- Circular fallback chains (detect and prevent infinite loops)
- Very long chains (>10 providers)
- Provider in chain becomes unavailable mid-chain
- Streaming fallback — must restart stream from new provider

**Traceability:** EPIC-1 (ACP-266) → FEAT-1.5 (ACP-277)

##### STORY-1.5.3: Circuit Breaker Pattern
**Goal:** Temporarily disable failing providers to prevent cascading failures.

**Description:** Implement circuit breaker per provider. Track consecutive failures. After N failures (configurable, default 5) in a sliding window, open the circuit — all requests fail fast. After cooldown period (30s default), allow one test request (half-open). If it succeeds, close the circuit. If it fails, stay open.

**Acceptance Criteria:**
- GIVEN a provider fails 5 consecutive times WHEN the circuit is open THEN requests to that provider fail immediately with 503
- GIVEN an open circuit WHEN the cooldown period expires THEN one test request is allowed (half-open state)
- GIVEN the test request succeeds WHEN the circuit transitions to half-open THEN it closes and normal operation resumes
- GIVEN the test request fails WHEN half-open THEN the circuit stays open and cooldown restarts

**Edge Cases:**
- Circuit breaker state persisted on hot-reload
- All providers in circuit breaker open state — return 503 to client
- Provider recovers during retry backoff — reset failure counter

**Traceability:** EPIC-1 (ACP-266) → FEAT-1.5 (ACP-277)

### FEAT-1.6: Local Ollama Passthrough Provider

**Jira:** ACP-278  
**Objective:** Local Ollama as first-class connected provider

#### Stories

##### STORY-1.6.1: Ollama Provider Implementation
**Goal:** Proxy requests to local Ollama instance transparently.

**Description:** Implement the Ollama provider type that proxies requests to the configured Ollama host (default: http://ollama:11434). Forward /api/chat, /api/generate, /api/tags requests to the local Ollama. Return responses unchanged (already in Ollama format).

**Acceptance Criteria:**
- GIVEN a request for an Ollama model WHEN routed to the Ollama provider THEN the request is forwarded to the Ollama host
- GIVEN Ollama streaming response WHEN proxied THEN NDJSON stream is forwarded as-is
- GIVEN Ollama is unavailable WHEN a request comes in THEN the circuit breaker marks it as failed and fallback may trigger
- GIVEN Ollama health check WHEN performed THEN OllaRouter queries Ollama's /api/tags to verify connectivity

**Edge Cases:**
- Ollama host connection refused — return 502 with Ollama error format
- Ollama returns non-standard response — pass through anyway
- GPU memory exhaustion on Ollama host — returned as Ollama error

**Traceability:** EPIC-1 (ACP-266) → FEAT-1.6 (ACP-278)

### FEAT-1.7: Error Wrapping in Ollama Format

**Jira:** ACP-279  
**Objective:** Wrap all provider errors in Ollama-compatible format

#### Stories

##### STORY-1.7.1: Provider Error Translation
**Goal:** Translate provider errors to Ollama error format.

**Description:** Intercept all provider error responses (4xx, 5xx, network errors) and translate to {"error":"descriptive message"} format. Include provider name and status code in the error message for debugging. Ensure Content-Type is application/json.

**Acceptance Criteria:**
- GIVEN an OpenAI 401 error WHEN received THEN OllaRouter returns {"error":"OpenAI authentication failed (401)"}
- GIVEN a network timeout (30s default) WHEN exceeded THEN OllaRouter returns {"error":"provider \"OpenAI\" request timed out"}
- GIVEN a provider connection refused WHEN attempted THEN OllaRouter returns {"error":"provider \"Ollama\" is unreachable"}
- GIVEN all fallbacks exhausted WHEN none succeed THEN OllaRouter returns {"error":"all providers in fallback chain failed"}

**Edge Cases:**
- Provider returns HTML error page — wrap in clean JSON error
- Provider returns valid JSON error with different format — normalize
- Error during streaming — send error NDJSON line and close
- Error in circuit breaker open state — immediate 503

**Traceability:** EPIC-1 (ACP-266) → FEAT-1.7 (ACP-279)

---

## EPIC-2: Provider Configuration Web UI

**Jira:** ACP-267  
**Priority:** P1 — v1.0  
**Depends On:** EPIC-1  
**Business Outcome:** Users can add/edit/delete providers via web browser — no YAML editing required.  
**Exit Criteria:** Users can add, edit, delete providers via web UI and see their health status.  
**User Story:** "As a platform engineer, I want to add providers through a web form instead of editing YAML files, so my non-technical team members can manage LLM access."  
**UX Reference:** docs/product/traceability/checkpoints/04-ux-design.md

### FEAT-2.1: Provider List Page

**Jira:** ACP-280

#### Stories

##### STORY-2.1.1: Provider Table with Status Display
**Goal:** Display all configured providers in a table with status.

**Acceptance Criteria:**
- GIVEN 3 configured providers WHEN the provider page loads THEN all 3 are shown in the table
- GIVEN a provider with Online status WHEN displayed THEN a green dot and "Online" label is shown
- GIVEN a provider with Offline status WHEN displayed THEN a red dot and "Offline" label is shown
- GIVEN a provider with Degraded status WHEN displayed THEN an amber dot and "Degraded" label is shown
- GIVEN no providers configured WHEN the page loads THEN empty state with "No providers configured. Add your first provider to get started." and an "Add Provider" CTA button is shown

**Edge Cases:** 50+ providers (pagination), very long provider names (truncation with tooltip)

**Traceability:** EPIC-2 (ACP-267) → FEAT-2.1 (ACP-280)

##### STORY-2.1.2: Provider Row Actions (Edit, Delete, Test Connection)
**Goal:** Provide per-provider row actions.

**Acceptance Criteria:**
- GIVEN a provider row WHEN "Edit" is clicked THEN the edit provider form opens with pre-filled values
- GIVEN a provider row WHEN "Delete" is clicked THEN a confirmation dialog appears with the provider name
- GIVEN the delete confirmation is accepted WHEN clicked THEN the provider is removed from runtime config and the row disappears
- GIVEN "Test Connection" is clicked WHEN the provider responds THEN success/failure result is shown inline

**Traceability:** EPIC-2 (ACP-267) → FEAT-2.1 (ACP-280)

### FEAT-2.2: Add/Edit Provider Wizard

**Jira:** ACP-281

#### Stories

##### STORY-2.2.1: Provider Type Selection (Step 1)
**Goal:** Select provider type from visual grid.

**Acceptance Criteria:**
- GIVEN the Add Provider wizard WHEN Step 1 loads THEN 6 provider type cards are displayed: OpenAI, Anthropic, Google, Ollama, ChatGPT Plus, OpenAI-Compatible
- GIVEN a provider type card WHEN clicked THEN it highlights with a 2px primary border
- GIVEN a type is selected WHEN "Next" is clicked THEN the wizard advances to Step 2
- GIVEN no type is selected WHEN "Next" is clicked THEN the button remains disabled

**Traceability:** EPIC-2 (ACP-267) → FEAT-2.1 (ACP-281)

##### STORY-2.2.2: Connection Details Form (Step 2)
**Goal:** Enter provider-specific connection details.

**Acceptance Criteria:**
- GIVEN "OpenAI" type is selected WHEN Step 2 loads THEN fields shown: Provider Name, API Key (masked), Organization ID (optional), Base URL (pre-filled with https://api.openai.com/v1)
- GIVEN "Ollama" type is selected WHEN Step 2 loads THEN only Provider Name and Base URL fields are shown (no API key)
- GIVEN "ChatGPT Plus" type is selected WHEN Step 2 loads THEN fields shown: Provider Name, Session Cookie (masked), User Agent (auto-filled)
- GIVEN API key field WHEN the 👁 toggle is clicked THEN the key is shown in plaintext
- GIVEN "Test Connection" is clicked WHEN the provider responds THEN success (green checkmark + latency) or failure (red X + error message) is shown
- GIVEN required fields are empty WHEN "Next" is clicked THEN inline validation errors appear with aria-describedby

**Traceability:** EPIC-2 (ACP-267) → FEAT-2.1 (ACP-281)

##### STORY-2.2.3: Model Configuration (Step 3)
**Goal:** Select and map models for the provider.

**Acceptance Criteria:**
- GIVEN Step 3 loads WHEN provider has auto-detected models THEN available models are shown in a searchable checkbox list
- GIVEN a model search query WHEN typed THEN the list filters in real-time
- GIVEN default models are available (e.g., gpt-4o, gpt-4o-mini for OpenAI) WHEN the step loads THEN they are pre-selected
- GIVEN custom model name mapping WHEN "Short Name → Provider Model Name" rows are added THEN the mapping is saved with the provider

**Traceability:** EPIC-2 (ACP-267) → FEAT-2.1 (ACP-281)

##### STORY-2.2.4: Defaults & Save (Step 4)
**Goal:** Configure provider defaults and save.

**Acceptance Criteria:**
- GIVEN Step 4 loads THEN fields shown: Max Retries (default 3), Timeout seconds (default 30), Cost per 1K input tokens (pre-filled with provider default), Cost per 1K output tokens (pre-filled)
- GIVEN "Save Provider" is clicked WHEN all steps are valid THEN a success toast appears and redirects to provider list
- GIVEN the provider is saved WHEN the runtime config is updated THEN the new provider is available immediately without container restart

**Traceability:** EPIC-2 (ACP-267) → FEAT-2.1 (ACP-281)

### FEAT-2.3: Provider Health Status Display

**Jira:** ACP-282

#### Stories

##### STORY-2.3.1: Provider Health Status Cards
**Goal:** Display real-time provider health in grid.

**Acceptance Criteria:**
- GIVEN 4 providers configured WHEN the health page loads THEN 4 status cards are displayed in a grid
- GIVEN a provider is Online WHEN shown THEN card has green dot with "Online" label, latency, request count, circuit breaker state
- GIVEN a provider is Degraded WHEN shown THEN card has amber dot with "Degraded" label
- GIVEN a provider is Offline WHEN shown THEN card has red dot with "Offline" label
- GIVEN auto-refresh is ON WHEN 30s elapses THEN all cards update with fresh data
- GIVEN no providers configured WHEN page loads THEN empty state with helpful message

**Traceability:** EPIC-2 (ACP-267) → FEAT-2.3 (ACP-282)

### FEAT-2.4: Settings Page

**Jira:** ACP-283

#### Stories

##### STORY-2.4.1: General Settings & Provider Defaults
**Goal:** Configure OllaRouter global settings.

**Acceptance Criteria:**
- GIVEN the Settings page WHEN General Settings section loads THEN fields shown: Proxy Port, Admin UI Port, Log Level dropdown, Prometheus Metrics toggle, Usage Persistence dropdown
- GIVEN Provider Defaults section WHEN loaded THEN fields shown: Default Max Retries, Default Timeout, Backoff Strategy, Circuit Breaker thresholds
- GIVEN "Save All" is clicked WHEN settings are valid THEN a success toast appears and settings are applied immediately

**Traceability:** EPIC-2 (ACP-267) → FEAT-2.4 (ACP-283)

##### STORY-2.4.2: Read-Only YAML Config Preview
**Goal:** Show current runtime configuration as YAML.

**Acceptance Criteria:**
- GIVEN 3 providers configured WHEN the YAML preview section loads THEN all 3 providers are shown in syntax-highlighted YAML
- GIVEN API keys in the config WHEN displayed THEN they are masked (sk-...xxxx)
- GIVEN the config changes via UI WHEN the page is refreshed THEN the YAML reflects the new state

**Traceability:** EPIC-2 (ACP-267) → FEAT-2.4 (ACP-283)

---

## EPIC-3: API Key Authentication & Rate Limiting

**Jira:** ACP-268  
**Priority:** P1 — v1.0  
**Depends On:** EPIC-1  
**Exit Criteria:** Gateway rejects unauthenticated requests, rate limits per key, keys manageable via UI.

### FEAT-3.1: API Key Authentication Middleware

**Jira:** ACP-284

#### Stories

##### STORY-3.1.1: Bearer Token Validation
**Goal:** Validate API keys from Authorization header.

**Acceptance Criteria:**
- GIVEN auth is enabled WHEN a request includes a valid API key in Authorization: Bearer header THEN the request is allowed through
- GIVEN auth is enabled WHEN a request has no Authorization header THEN 401 is returned with {"error":"authentication required"}
- GIVEN auth is enabled WHEN a request has an invalid API key THEN 401 is returned with {"error":"invalid API key"}
- GIVEN auth is disabled (empty key list) WHEN any request comes in THEN all requests are allowed

**Edge Cases:** Key rotation (old + new keys both valid during transition), 100+ keys in list

**Traceability:** EPIC-3 (ACP-268) → FEAT-3.1 (ACP-284)

##### STORY-3.1.2: Secure Key Configuration
**Goal:** Configure API keys without exposing in plaintext.

**Acceptance Criteria:**
- GIVEN API keys are configured via YAML with ${ENV_VAR} WHEN loaded THEN keys are never stored in plaintext in logs
- GIVEN a request is logged WHEN the API key is present THEN it is masked as "sk-...xxxx"
- GIVEN the same key is used by multiple clients WHEN tracked THEN usage is aggregated by key ID

**Traceability:** EPIC-3 (ACP-268) → FEAT-3.1 (ACP-284)

### FEAT-3.2: Per-Key Rate Limiting

**Jira:** ACP-285

#### Stories

##### STORY-3.2.1: Token Bucket Rate Limiter
**Goal:** Implement per-key rate limiting.

**Acceptance Criteria:**
- GIVEN a key with 60 req/min limit WHEN 61 requests arrive in the same minute THEN the 61st request returns 429 with Retry-After header
- GIVEN a rate-limited request WHEN the Retry-After time passes THEN subsequent requests are allowed
- GIVEN a key with no rate limit configured WHEN requests arrive THEN no rate limiting is applied

**Edge Cases:** Burst allowance (allow short bursts above average), distributed rate limiting (in-memory is sufficient for single instance)

**Traceability:** EPIC-3 (ACP-268) → FEAT-3.2 (ACP-285)

##### STORY-3.2.2: Budget Cap Enforcement
**Goal:** Enforce monthly budget caps per API key.

**Acceptance Criteria:**
- GIVEN a key with $100/month budget WHEN cumulative cost exceeds $100 THEN subsequent requests return 429 with {"error":"monthly budget exceeded"}
- GIVEN a budget cap is reached WHEN the calendar month resets THEN requests are allowed again

**Traceability:** EPIC-3 (ACP-268) → FEAT-3.2 (ACP-285)

### FEAT-3.3: API Key Management UI

**Jira:** ACP-286

#### Stories

##### STORY-3.3.1: API Key List Page
**Goal:** Display, create, and revoke API keys.

**Acceptance Criteria:**
- GIVEN the API Keys page loads WHEN keys exist THEN they are shown in a table with name, masked key, status, usage, rate limit, last used
- GIVEN "New Key" is clicked WHEN the dialog opens THEN fields shown: Key Name, Rate Limit, Monthly Budget, Expiration Date
- GIVEN "Generate Key" is clicked WHEN created THEN the key is shown once with "Copy" button and warning to save it
- GIVEN "Revoke" is clicked WHEN confirmed THEN the key status changes to "Revoked" and it stops working

**Traceability:** EPIC-3 (ACP-268) → FEAT-3.3 (ACP-286)

---

## EPIC-4: Usage Tracking & Stats Dashboard

**Jira:** ACP-269  
**Priority:** P1 — v1.0  
**Depends On:** EPIC-1  
**Exit Criteria:** Dashboard shows per-provider token usage, cost, and request history.

### FEAT-4.1: Token and Cost Tracking Engine

**Jira:** ACP-287

#### Stories

##### STORY-4.1.1: Request Token Counting
**Goal:** Count input and output tokens per request.

**Acceptance Criteria:**
- GIVEN a completed request WHEN the provider returns token usage THEN input and output token counts are recorded
- GIVEN a provider that doesn't return token counts WHEN the request completes THEN tokens are estimated using a configurable ratio
- GIVEN a streaming request WHEN completed THEN accumulated token counts from the final message are used

**Traceability:** EPIC-4 (ACP-269) → FEAT-4.1 (ACP-287)

##### STORY-4.1.2: Cost Calculation Engine
**Goal:** Calculate estimated cost per request.

**Acceptance Criteria:**
- GIVEN input token count and configured input price WHEN a request completes THEN input cost = tokens × (price_per_1K / 1000)
- GIVEN output token count and configured output price WHEN a request completes THEN output cost = tokens × (price_per_1K / 1000)
- GIVEN no pricing configured for a provider WHEN a request completes THEN cost is recorded as $0.00 with a warning

**Traceability:** EPIC-4 (ACP-269) → FEAT-4.1 (ACP-287)

##### STORY-4.1.3: Usage Data Persistence
**Goal:** Store usage data in-memory with optional file persistence.

**Acceptance Criteria:**
- GIVEN usage data is accumulated WHEN stored in-memory THEN it is accessible via the stats API for dashboard queries
- GIVEN file persistence is enabled WHEN data is recorded THEN it is written to a JSON file every 60s
- GIVEN OllaRouter restarts WHEN file persistence was enabled THEN usage data is loaded from the file on startup
- GIVEN in-memory only mode WHEN OllaRouter restarts THEN usage data resets to zero

**Traceability:** EPIC-4 (ACP-269) → FEAT-4.1 (ACP-287)

### FEAT-4.2: Stats Dashboard Web UI

**Jira:** ACP-288

#### Stories

##### STORY-4.2.1: Summary Cards and Cost Charts
**Goal:** Display cost summary and trend charts.

**Acceptance Criteria:**
- GIVEN usage data exists WHEN the dashboard loads THEN 4 summary cards show: Total Cost (month), Total Requests (today), Active Providers, Avg Latency
- GIVEN cost data exists WHEN the cost tab is active THEN a multi-series line chart shows daily cost per provider over the selected time range
- GIVEN cost data exists WHEN displayed THEN a horizontal bar chart shows cost by provider sorted descending
- GIVEN cost data exists WHEN displayed THEN a breakdown table shows provider, requests, tokens, cost, and % of total
- GIVEN a provider row is clicked WHEN in the breakdown table THEN a drill-down shows per-model cost for that provider
- GIVEN no usage data exists WHEN the dashboard loads THEN "No usage data yet" empty state is shown

**Traceability:** EPIC-4 (ACP-269) → FEAT-4.2 (ACP-288)

##### STORY-4.2.2: Time Range Selector and CSV Export
**Goal:** Filter by time range and export data.

**Acceptance Criteria:**
- GIVEN the time range selector WHEN changed to 7d, 30d, or 90d THEN all charts and tables update without page reload
- GIVEN "Export CSV" is clicked WHEN data exists THEN a CSV file is downloaded with provider breakdown data
- GIVEN the dashboard WHEN loaded THEN data refreshes automatically every 30s with a subtle animation

**Traceability:** EPIC-4 (ACP-269) → FEAT-4.2 (ACP-288)

### FEAT-4.3: Request Log with Filtering

**Jira:** ACP-289

#### Stories

##### STORY-4.3.1: Infinite-Scroll Request Log Table
**Goal:** Display recent request log with filtering and detail expansion.

**Acceptance Criteria:**
- GIVEN requests have been processed WHEN the request log tab loads THEN an infinite-scroll table shows: timestamp, model, provider, status, latency, input/output tokens, cost
- GIVEN a column header WHEN clicked THEN the table sorts by that column ascending/descending
- GIVEN filter controls WHEN used THEN the log is filtered by provider (multi-select), model (text search), status code, and date range
- GIVEN a request row WHEN clicked THEN it expands to show request/response headers, fallback chain, and body preview

**Traceability:** EPIC-4 (ACP-269) → FEAT-4.3 (ACP-289)

---

## EPIC-5: ChatGPT Plus Web Proxy Provider

**Jira:** ACP-270  
**Priority:** P1 — v1.1  
**Depends On:** EPIC-1, EPIC-2  
**Exit Criteria:** Users can configure ChatGPT Plus cookie and use ChatGPT models through the Ollama API.

### FEAT-5.1: ChatGPT Plus Provider Type

**Jira:** ACP-290

#### Stories

##### STORY-5.1.1: ChatGPT Plus Request Proxying
**Goal:** Proxy requests through ChatGPT Plus using session cookie.

**Acceptance Criteria:**
- GIVEN a valid ChatGPT Plus session cookie WHEN a chat request is sent for model "chatgpt-gpt-5" THEN OllaRouter proxies the request to ChatGPT's internal API
- GIVEN a ChatGPT Plus response WHEN received THEN it is translated to Ollama NDJSON format
- GIVEN a ChatGPT Plus streaming response WHEN received THEN SSE events are translated to NDJSON lines
- GIVEN the ChatGPT Plus session is expired WHEN a request is made THEN 401 is returned with cookie expiry error

**Edge Cases:** ChatGPT Plus rate limits (different from API), model name changes on ChatGPT side, concurrent session limit

**Traceability:** EPIC-5 (ACP-270) → FEAT-5.1 (ACP-290)

##### STORY-5.1.2: ChatGPT Model Auto-Detection
**Goal:** Auto-detect available ChatGPT Plus models.

**Acceptance Criteria:**
- GIVEN a valid ChatGPT Plus session WHEN the provider is configured THEN available models are auto-detected (30+ models)
- GIVEN ChatGPT adds new models WHEN already configured THEN new models appear in /api/tags after next poll cycle

**Traceability:** EPIC-5 (ACP-270) → FEAT-5.1 (ACP-290)

### FEAT-5.2: Cookie Management & Auto-Refresh

**Jira:** ACP-291

#### Stories

##### STORY-5.2.1: Cookie Health Monitoring
**Goal:** Monitor and report ChatGPT Plus cookie health.

**Acceptance Criteria:**
- GIVEN a valid cookie WHEN health is checked THEN the cookie status is "healthy" and provider shows Online
- GIVEN a cookie nearing expiry (<48h) WHEN health is checked THEN the provider shows Degraded with warning
- GIVEN an invalid/expired cookie WHEN a request is made THEN the provider shows Offline and requests fail with auth error

**Traceability:** EPIC-5 (ACP-270) → FEAT-5.2 (ACP-291)

##### STORY-5.2.2: Cookie Auto-Refresh and Challenge Handling
**Goal:** Auto-refresh ChatGPT Plus cookies and handle Cloudflare challenges.

**Acceptance Criteria:**
- GIVEN a 403 response from ChatGPT API WHEN received THEN OllaRouter attempts to refresh cookies and retry the request
- GIVEN a Cloudflare challenge WHEN encountered THEN OllaRouter uses the PoW solver (or existing cookies) to bypass
- GIVEN cookie refresh fails AFTER N attempts THEN the provider is marked Offline

**Traceability:** EPIC-5 (ACP-270) → FEAT-5.2 (ACP-291)

---

## EPIC-6: Prometheus Metrics, Health Endpoints & Structured Logging

**Jira:** ACP-271  
**Priority:** P1 — v1.1  
**Depends On:** EPIC-1  
**Exit Criteria:** Prometheus can scrape /metrics, health endpoints report provider status, logs are structured JSON.

### FEAT-6.1: Health Endpoints

**Jira:** ACP-292

#### Stories

##### STORY-6.1.1: /health and /health/providers Endpoints
**Goal:** Expose service and provider health endpoints.

**Acceptance Criteria:**
- GIVEN OllaRouter is running WHEN GET /health is called THEN 200 OK is returned with JSON body containing status, uptime, version
- GIVEN all providers are online WHEN GET /health/providers is called THEN each provider shows "online" status with latency
- GIVEN a provider is offline WHEN GET /health/providers is called THEN that provider shows "offline" with error info
- GIVEN Docker HEALTHCHECK uses /health WHEN OllaRouter is healthy THEN HEALTHCHECK passes; when unhealthy, container restarts

**Traceability:** EPIC-6 (ACP-271) → FEAT-6.1 (ACP-292)

### FEAT-6.2: Prometheus Metrics Endpoint

**Jira:** ACP-293

#### Stories

##### STORY-6.2.1: /metrics Endpoint in Prometheus Format
**Goal:** Expose Prometheus metrics for monitoring.

**Acceptance Criteria:**
- GIVEN Prometheus is configured to scrape /metrics WHEN it scrapes THEN valid Prometheus text format is returned
- GIVEN requests have been processed WHEN /metrics is scraped THEN metrics include: request_count (by provider, model, status), request_duration_seconds (histogram), token_count_total, providers_active, circuit_breaker_state
- GIVEN the Prometheus toggle is disabled WHEN /metrics is called THEN 404 is returned

**Traceability:** EPIC-6 (ACP-271) → FEAT-6.2 (ACP-293)

### FEAT-6.3: Structured JSON Logging

**Jira:** ACP-294

#### Stories

##### STORY-6.3.1: Structured Log Output
**Goal:** Output structured JSON logs with configurable levels.

**Acceptance Criteria:**
- GIVEN log level is "info" WHEN a request is processed THEN a JSON log line is output with: timestamp, level, message, request_id, provider, model, status, duration_ms
- GIVEN log level is "debug" WHEN enabled THEN additional debug fields are included (request body summary, response body summary)
- GIVEN an API key is present in a log context WHEN logged THEN it is masked as "sk-...xxxx"
- GIVEN log level is "error" WHEN a provider error occurs THEN full error details are logged

**Traceability:** EPIC-6 (ACP-271) → FEAT-6.3 (ACP-294)

---

## EPIC-7: Enhanced Model Discovery & Extended Ollama Endpoints

**Jira:** ACP-272  
**Priority:** P2 — v1.2  
**Depends On:** EPIC-1, EPIC-2  
**Exit Criteria:** Users can browse all available models in a web UI, see model details with pricing and fallback chains.

### FEAT-7.1: Extended Ollama Endpoints

**Jira:** ACP-295

#### Stories

##### STORY-7.1.1: GET /api/version Endpoint
**Goal:** Return OllaRouter version info.

**Acceptance Criteria:**
- GIVEN GET /api/version is called WHEN OllaRouter is running THEN response includes {"version":"x.y.z","api_version":"1.0.0"} in Ollama format

**Traceability:** EPIC-7 (ACP-272) → FEAT-7.1 (ACP-295)

##### STORY-7.1.2: POST /api/show Endpoint
**Goal:** Return model details for known models.

**Acceptance Criteria:**
- GIVEN a model name that exists WHEN POST /api/show is called THEN response includes model details (modified_at, size, details format, parameters, template if available)
- GIVEN a cloud model (e.g., gpt-4o) WHEN POST /api/show is called THEN response returns metadata from provider config with size=0
- GIVEN an unknown model WHEN POST /api/show is called THEN 404 is returned

**Traceability:** EPIC-7 (ACP-272) → FEAT-7.1 (ACP-295)

### FEAT-7.2: Model Catalog Web UI

**Jira:** ACP-296

#### Stories

##### STORY-7.2.1: Searchable Model Catalog Table
**Goal:** Display all models in searchable, filterable table.

**Acceptance Criteria:**
- GIVEN models exist WHEN the model catalog loads THEN all models are shown with name, provider badge (colored), status, cost per token
- GIVEN the search input WHEN text is typed THEN models filter in real-time by name
- GIVEN provider filter chips WHEN clicked THEN models are filtered by selected provider
- GIVEN no models exist WHEN the page loads THEN empty state shows "No models available. Add a provider to see available models."

**Traceability:** EPIC-7 (ACP-272) → FEAT-7.2 (ACP-296)

##### STORY-7.2.2: Model Detail Slide-Over
**Goal:** Show model details in a slide-over panel.

**Acceptance Criteria:**
- GIVEN a model name is clicked WHEN in the catalog THEN a slide-over panel opens from the right
- GIVEN the slide-over WHEN open THEN it shows: provider name, full model name, context window, pricing, modality, usage stats (7 days), fallback chain visualization
- GIVEN "Quick Test" is clicked WHEN in the slide-over THEN a test request is sent and the response is displayed inline
- GIVEN "Copy Model Name" is clicked WHEN in the slide-over THEN the model name is copied to clipboard

**Traceability:** EPIC-7 (ACP-272) → FEAT-7.2 (ACP-296)

---

## Open Questions

| # | Question | Suggested Resolution |
|---|----------|---------------------|
| Q1 | Should OllaRouter listen on port 11434 (replacing Ollama) or 11435 (alongside)? | Default 11435, configurable. Document both modes. |
| Q2 | Should ChatGPT Plus proxy be built-in or separate companion container? | Built-in as pluggable provider interface within the Go binary. |
| Q3 | Should cost tracking use provider-published pricing or custom pricing config? | Configurable pricing per provider with sensible defaults from published pricing. |
| Q4 | Should usage data be persisted to disk or ephemeral? | In-memory for MVP with optional volume-mounted file persistence (every 60s). |
| Q5 | Should the web UI be a separate process or embedded in Go binary? | Embedded web UI (single Go binary, serve embedded static assets). |
| Q6 | Minimum viable provider set for MVP? | OpenAI + Ollama + Anthropic for MVP; Google + ChatGPT Plus for v1.1. |
| Q7 | Support /api/embed (embeddings) in MVP? | Defer to post-MVP. |

---

## Traceability Links

| Entity | Type | Jira Key | Upstream |
|--------|------|----------|----------|
| OllaRouter Idea | idea | — | — |
| Phase 1 Market Research | research | — | derived_from idea |
| Phase 2 Competitor Analysis | research | — | derived_from idea |
| Phase 3 USP Definition | decision | — | derived_from research |
| Phase 4 UX Design | decision | — | derived_from USP |
| EPIC-1: Ollama API Core | epic | ACP-266 | derived_from USP |
| EPIC-2: Provider Config UI | epic | ACP-267 | derived_from USP |
| EPIC-3: Auth & Rate Limiting | epic | ACP-268 | derived_from USP |
| EPIC-4: Cost Tracking | epic | ACP-269 | derived_from USP |
| EPIC-5: ChatGPT Plus | epic | ACP-270 | derived_from USP |
| EPIC-6: Observability | epic | ACP-271 | derived_from USP |
| EPIC-7: Model Discovery | epic | ACP-272 | derived_from USP |

---

## Jira Issue Summary

| Key | Title | Type |
|-----|-------|------|
| ACP-266 | EPIC-1: Ollama API Core — Multi-Provider Request Routing | Epic |
| ACP-267 | EPIC-2: Provider Configuration Web UI | Epic |
| ACP-268 | EPIC-3: API Key Authentication & Rate Limiting | Epic |
| ACP-269 | EPIC-4: Usage Tracking & Stats Dashboard | Epic |
| ACP-270 | EPIC-5: ChatGPT Plus Web Proxy Provider | Epic |
| ACP-271 | EPIC-6: Prometheus Metrics, Health Endpoints & Structured Logging | Epic |
| ACP-272 | EPIC-7: Enhanced Model Discovery & Extended Ollama Endpoints | Epic |
| ACP-273 | FEAT-1.1: POST /api/chat Endpoint with NDJSON Streaming | Task |
| ACP-274 | FEAT-1.2: POST /api/generate Endpoint with NDJSON Streaming | Task |
| ACP-275 | FEAT-1.3: GET /api/tags with Aggregated Model Listing | Task |
| ACP-276 | FEAT-1.4: Provider Routing Engine | Task |
| ACP-277 | FEAT-1.5: Fallback/Failover on 429/5xx | Task |
| ACP-278 | FEAT-1.6: Local Ollama Passthrough Provider | Task |
| ACP-279 | FEAT-1.7: Error Wrapping in Ollama Format | Task |
| ACP-280 | FEAT-2.1: Provider List Page | Task |
| ACP-281 | FEAT-2.2: Add/Edit Provider Wizard | Task |
| ACP-282 | FEAT-2.3: Provider Health Status Display | Task |
| ACP-283 | FEAT-2.4: Settings Page | Task |
| ACP-284 | FEAT-3.1: API Key Authentication Middleware | Task |
| ACP-285 | FEAT-3.2: Per-Key Rate Limiting | Task |
| ACP-286 | FEAT-3.3: API Key Management UI | Task |
| ACP-287 | FEAT-4.1: Token and Cost Tracking Engine | Task |
| ACP-288 | FEAT-4.2: Stats Dashboard Web UI | Task |
| ACP-289 | FEAT-4.3: Request Log with Filtering | Task |
| ACP-290 | FEAT-5.1: ChatGPT Plus Provider Type | Task |
| ACP-291 | FEAT-5.2: Cookie Management & Auto-Refresh | Task |
| ACP-292 | FEAT-6.1: Health Endpoints | Task |
| ACP-293 | FEAT-6.2: Prometheus Metrics Endpoint | Task |
| ACP-294 | FEAT-6.3: Structured JSON Logging | Task |
| ACP-295 | FEAT-7.1: Extended Ollama Endpoints (/api/version, /api/show) | Task |
| ACP-296 | FEAT-7.2: Model Catalog Web UI | Task |

---

*End of OllaRouter Backlog — generated by story-writer agent*
