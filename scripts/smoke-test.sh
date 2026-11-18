#!/usr/bin/env bash
# End-to-end smoke test: builds OllaRouter, starts it against a mock
# OpenAI-compatible provider and checks auth, both API styles, streaming,
# virtual-model fallback, cost tracking, health, metrics and the admin UI.
# Needs Go, Python 3 and curl. Run from anywhere: scripts/smoke-test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="$(mktemp -d)"
BASE="http://127.0.0.1:18435"
KEY="Authorization: Bearer smoke-test-key"
PASS=0
FAIL=0

cleanup() {
  for pid in "${SRV_PID:-}" "${MOCK_PID:-}"; do
    [[ -n "$pid" ]] || continue
    kill "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
  done
  rm -rf "$WORK"
}
trap cleanup EXIT

check() { # name, expected substring, actual
  if [[ "$3" == *"$2"* ]]; then
    PASS=$((PASS + 1)); printf '  ok    %s\n' "$1"
  else
    FAIL=$((FAIL + 1)); printf '  FAIL  %s\n        expected to contain: %s\n        got: %s\n' "$1" "$2" "$3"
  fi
}

echo "Building..."
(cd "$ROOT/ollarouter" && CGO_ENABLED=0 go build \
  -ldflags "-X github.com/aideveloper1976-arch/ollarouter/internal/router.FallbackVersion=smoke" \
  -o "$WORK/ollarouter" ./cmd/ollarouter)

python3 "$ROOT/scripts/mock_provider.py" 18998 & MOCK_PID=$!
"$WORK/ollarouter" -config "$ROOT/scripts/smoke-providers.yaml" >"$WORK/server.log" 2>&1 & SRV_PID=$!

for _ in $(seq 1 50); do
  curl -fs -o /dev/null "$BASE/api/version" && break
  sleep 0.2
done

echo "Running checks..."
check "version endpoint" '"version":"smoke"' "$(curl -s "$BASE/api/version")"
check "model list includes provider/model names" '"name":"mock/mock-small"' "$(curl -s "$BASE/api/tags")"

check "/api/chat rejects missing key" 'authentication required' \
  "$(curl -s "$BASE/api/chat" -d '{"model":"mock/mock-small","messages":[{"role":"user","content":"hi"}],"stream":false}')"
check "/v1/chat/completions rejects missing key" 'authentication required' \
  "$(curl -s "$BASE/v1/chat/completions" -d '{"model":"mock/mock-small","messages":[{"role":"user","content":"hi"}]}')"

check "Ollama chat, direct provider/model" '"content":"echo: hi"' \
  "$(curl -s -H "$KEY" "$BASE/api/chat" -d '{"model":"mock/mock-small","messages":[{"role":"user","content":"hi"}],"stream":false}')"
check "Ollama generate" '"response":"echo: gen"' \
  "$(curl -s -H "$KEY" "$BASE/api/generate" -d '{"model":"mock/mock-small","prompt":"gen","stream":false}')"
check "virtual model falls back past a dead provider" '"content":"echo: fallback"' \
  "$(curl -s -H "$KEY" "$BASE/api/chat" -d '{"model":"resilient","messages":[{"role":"user","content":"fallback"}],"stream":false}')"
check "Ollama NDJSON stream through fallback" '"content":"stream"' \
  "$(curl -s -H "$KEY" "$BASE/api/chat" -d '{"model":"resilient","messages":[{"role":"user","content":"stream"}],"stream":true}')"
check "OpenAI chat completion" '"content":"echo: openai"' \
  "$(curl -s -H "$KEY" "$BASE/v1/chat/completions" -d '{"model":"mock/mock-small","messages":[{"role":"user","content":"openai"}]}')"
check "OpenAI SSE stream ends with [DONE]" 'data: [DONE]' \
  "$(curl -s -H "$KEY" "$BASE/v1/chat/completions" -d '{"model":"mock/mock-small","stream":true,"messages":[{"role":"user","content":"sse"}]}')"

check "cost tracked from provider pricing" '"mock":' \
  "$(curl -s -H "$KEY" "$BASE/admin/api/stats/cost-by-provider")"
check "admin API rejects missing key" 'authentication required' \
  "$(curl -s "$BASE/admin/api/stats/summary")"
check "health reports one provider online" '"providers_online":1' "$(curl -s "$BASE/health")"
check "Prometheus metrics" 'ollarouter_providers_total 2' "$(curl -s "$BASE/metrics")"
check "admin UI served" 'text/html' "$(curl -s -o /dev/null -w '%{content_type}' "$BASE/admin/")"

echo
echo "Passed: $PASS  Failed: $FAIL"
if (( FAIL > 0 )); then
  echo "--- server log"; cat "$WORK/server.log"
  exit 1
fi
