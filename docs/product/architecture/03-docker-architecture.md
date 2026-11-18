# OllaRouter — Docker Architecture

**Version:** 0.1.0  
**Date:** 2026-05-20  
**Status:** Approved

---

## 1. Multi-Stage Build Strategy

### 1.1 Dockerfile

```dockerfile
# ============================================================
# Stage 1: Build Go binary
# ============================================================
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build

# Dependency caching layer
COPY go.mod go.sum ./
RUN go mod download

# Copy source (admin UI static assets are pre-built and committed)
COPY . .

# Build static binary with debug symbols stripped
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
    -o /build/ollarouter \
    ./cmd/ollarouter

# ============================================================
# Stage 2: Minimal runtime image
# ============================================================
FROM gcr.io/distroless/static-debian12:nonroot

# Copy only the binary (static assets are compiled in via go:embed)
COPY --from=builder /build/ollarouter /app/ollarouter

# Expose OllaRouter API port
EXPOSE 11435

# Run as non-root user
USER nonroot:nonroot

# Health check (built-in: the binary runs a healthcheck subcommand)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/app/ollarouter", "healthcheck"]

ENTRYPOINT ["/app/ollarouter"]
```

### 1.2 Build Size Budget

| Component | Size |
|-----------|------|
| Go binary (stripped, static) | ~15-25 MB |
| Distroless base image | ~2 MB |
| **Total image size** | **~20-30 MB** |
| Runtime RSS (idle) | ~10-20 MB |
| Runtime RSS (under load) | ~50-200 MB |

### 1.3 Makefile

```makefile
.PHONY: build docker-build docker-run clean

BINARY_NAME=ollarouter
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR=build

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build \
		-ldflags="-w -s -X main.version=$(VERSION)" \
		-o $(BUILD_DIR)/$(BINARY_NAME) \
		./cmd/$(BINARY_NAME)

docker-build:
	docker build -t $(BINARY_NAME):$(VERSION) .
	docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest

docker-run:
	docker compose up -d

clean:
	rm -rf $(BUILD_DIR)
	docker rmi $(BINARY_NAME):latest 2>/dev/null || true

test:
	go test ./... -v -race -count=1

lint:
	golangci-lint run ./...

.PHONY: webui-build
webui-build:
	cd webui && npm ci && npm run build
```

---

## 2. Docker Compose Integration

### 2.1 Complete ollama-stack with OllaRouter

```yaml
version: '3.8'

services:
  # ============================================================
  # Ollama — Local LLM Inference Engine
  # ============================================================
  ollama:
    image: ollama/ollama:latest
    container_name: ollama
    volumes:
      - ollama_data:/root/.ollama
      - /mnt/c/Temp/models:/mnt/windows_models:ro   # Optional: pre-downloaded models
    environment:
      - OLLAMA_HOST=0.0.0.0
      - OLLAMA_FLASH_ATTENTION=1
      - OLLAMA_KV_CACHE_TYPE=q8_0
      - OLLAMA_NUM_PARALLEL=1
      - OLLAMA_MAX_LOADED_MODELS=1
      - OLLAMA_KEEP_ALIVE=30m
      - OLLAMA_CONTEXT_LENGTH=8192
    ports:
      - "11434:11434"
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: all
              capabilities: [gpu]
    restart: unless-stopped

  # ============================================================
  # OllaRouter — Multi-Provider LLM Gateway
  # ============================================================
  ollarouter:
    build:
      context: ./ollarouter
      dockerfile: Dockerfile
    image: ollarouter:latest
    container_name: ollarouter
    ports:
      - "11435:11435"               # Ollama-compatible API
    environment:
      # OllaRouter settings
      - OLLAROUTER_PORT=11435
      - OLLAROUTER_CONFIG_PATH=/app/config/providers.yaml
      - OLLAROUTER_LOG_LEVEL=info
      - OLLAROUTER_PROMETHEUS_ENABLED=true
      - OLLAROUTER_USAGE_PERSISTENCE=memory

      # Provider API keys (from .env file or host environment)
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - GROQ_API_KEY=${GROQ_API_KEY:-}
      - CHATGPT_SESSION_COOKIE=${CHATGPT_SESSION_COOKIE:-}

      # Client API keys (for authentication)
      - OLLAROUTER_API_KEY=${OLLAROUTER_API_KEY:-}
    volumes:
      # Config file (read-only for security)
      - ./ollarouter/providers.yaml:/app/config/providers.yaml:ro
      # Optional: usage data persistence
      - ollarouter_data:/data
    depends_on:
      - ollama
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "/app/ollarouter", "healthcheck"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 5s
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 256M

  # ============================================================
  # Open WebUI — Ollama-Compatible Web Chat Interface
  # ============================================================
  open-webui:
    image: ghcr.io/open-webui/open-webui:main
    container_name: open-webui
    restart: unless-stopped
    ports:
      - "3000:8080"
    volumes:
      - open_webui_data:/app/backend/data
    environment:
      # Point to OllaRouter instead of direct Ollama
      - OLLAMA_BASE_URL=http://ollarouter:11435
      - WEBUI_AUTH=False
      - WEBUI_SECRET_KEY=${WEBUI_SECRET_KEY:-change_me_please}
    depends_on:
      - ollarouter
    restart: unless-stopped

volumes:
  ollama_data:
    external: false
  ollarouter_data:
    external: false
  open_webui_data:
    external: false
```

### 2.2 Example .env File

```bash
# OllaRouter Provider API Keys
OPENAI_API_KEY=sk-proj-xxxxxxxxxxxxxxxx
ANTHROPIC_API_KEY=sk-ant-xxxxxxxxxxxxxxxx
GROQ_API_KEY=gsk_xxxxxxxxxxxxxxxx
CHATGPT_SESSION_COOKIE=__Secure-next-auth.session-token=xxxxx...

# OllaRouter Client Authentication (optional)
OLLAROUTER_API_KEY=or_auth_key_here

# Open WebUI
WEBUI_SECRET_KEY=a_secure_random_secret_key_2026
```

### 2.3 Provider Config File (providers.yaml)

```yaml
# See docs/product/architecture/01-system-architecture.md for full schema.
settings:
  port: 11435
  log_level: info
  prometheus_enabled: true

providers:
  - name: openai-main
    type: openai
    api_key: "${OPENAI_API_KEY}"
    models:
      - name: gpt-4o
      - name: gpt-4o-mini
    pricing:
      input_per_1k: 0.0025
      output_per_1k: 0.01

  - name: local-ollama
    type: ollama
    base_url: http://ollama:11434
    auto_discover: true

virtual_models:
  - name: fast-chat
    fallback_chain:
      - openai-main/gpt-4o-mini
      - local-ollama/*
```

---

## 3. Networking

### 3.1 Default Network Topology

```
Host Network
    │
    ├── :11434 ──── ollama (internal + exposed)
    │
    ├── :11435 ──── ollarouter (internal + exposed)
    │                   │
    │                   ├── http://ollama:11434 (internal bridge)
    │                   ├── https://api.openai.com (external)
    │                   ├── https://api.anthropic.com (external)
    │                   └── https://chatgpt.com (external)
    │
    └── :3000  ──── open-webui (internal + exposed)
                        │
                        └── http://ollarouter:11435 (internal bridge)
```

### 3.2 Port Mapping

| Service | Container Port | Host Port | Protocol | Purpose |
|---------|---------------|-----------|----------|---------|
| ollama | 11434 | 11434 | TCP | Ollama API (direct) |
| ollarouter | 11435 | 11435 | TCP | Ollama-compatible API (via gateway) |
| open-webui | 8080 | 3000 | TCP | Web chat interface |

### 3.3 Docker Network Configuration

Services communicate over the default bridge network created by Docker Compose. No custom network is required.

For production:
```yaml
networks:
  internal:
    driver: bridge
    internal: true  # No external access
  external:
    driver: bridge

services:
  ollama:
    networks:
      - internal    # Only accessible within the stack

  ollarouter:
    networks:
      - internal    # Can reach ollama
      - external    # Exposes :11435 to host

  open-webui:
    networks:
      - internal    # Can reach ollarouter
      - external    # Exposes :3000 to host
```

---

## 4. Health Checks

### 4.1 Docker HEALTHCHECK

The `healthcheck` subcommand is a built-in feature of the OllaRouter binary:

```go
// healthcheckCmd runs the health check subcommand.
// Exit code 0 = healthy, 1 = unhealthy.
func healthcheckCmd() int {
    resp, err := http.Get("http://localhost:11435/health")
    if err != nil {
        return 1
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return 1
    }
    return 0
}
```

### 4.2 Health Endpoint

```json
GET /health
{
  "status": "ok",
  "version": "0.1.0",
  "uptime_seconds": 86400,
  "providers_total": 5,
  "providers_online": 4,
  "providers_degraded": 1,
  "providers_offline": 0
}
```

Unhealthy conditions:
- No providers online (all offline/degraded) → return 503
- Config file failed to load → return 503

---

## 5. Resource Profiles

| Scenario | CPU | Memory | Network |
|----------|-----|--------|---------|
| Idle (no requests) | <0.05 core | 10-20 MB RSS | Minimal (health polling) |
| Low load (1 req/s) | 0.1-0.2 core | 30-50 MB RSS | ~1 Mbps per stream |
| Medium load (10 req/s) | 0.3-0.5 core | 50-100 MB RSS | ~10 Mbps |
| High load (50 req/s) | 1-2 cores | 100-200 MB RSS | ~50 Mbps |

**Docker resource limits:**
```yaml
deploy:
  resources:
    limits:
      cpus: '1.0'
      memory: 256M
    reservations:
      cpus: '0.1'
      memory: 64M
```

---

## 6. Security Considerations

| Concern | Mitigation |
|---------|------------|
| Config contains API keys | All keys via `${ENV_VAR}` from `.env` file; YAML stores only variable references |
| Container compromise | Distroless base (no shell, no package manager); non-root user |
| Config tampering | Read-only bind mount (`:ro`) for providers.yaml |
| Secret leakage in logs | API key masking (`sk-...xxxx`) in all log output |
| Network sniffing | Use Docker internal network for Ollama communication |
| Docker layer secrets | `.dockerignore` excludes `.env` and `providers.yaml` from build context |

### .dockerignore

```
.env
providers.yaml
.git/
.gitignore
*.md
webui/node_modules/
webui/src/
build/
```

---

## 7. Volumes

| Volume | Mount Point | Purpose | Size | Persistence |
|--------|-------------|---------|------|-------------|
| `ollama_data` | `/root/.ollama` | Ollama model storage | 10-100 GB | Required |
| `ollarouter_data` | `/data` | Usage tracking persistence (optional) | <100 MB | Optional |
| `open_webui_data` | `/app/backend/data` | Open WebUI user data | <1 GB | Required |

---

*End of Docker Architecture Document*
