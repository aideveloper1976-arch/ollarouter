# Stage 1: Build Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /webui
COPY webui/package*.json ./
RUN npm ci
COPY webui/ ./
RUN npm run build

# Stage 2: Build Backend
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /build

# Copy go mod files first for better caching
COPY ollarouter/go.mod ollarouter/go.sum ./
RUN go mod download

# Copy backend source
COPY ollarouter/ ./

# Ensure the embed directory is clean before copying fresh frontend
RUN rm -rf internal/webui/dist && mkdir -p internal/webui/dist

# Copy built frontend to the embed location
COPY --from=frontend-builder /webui/dist ./internal/webui/dist

# Build static binary
ARG OLLAROUTER_VERSION=0.22.0
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-w -s -X github.com/aideveloper1976-arch/ollarouter/internal/router.FallbackVersion=${OLLAROUTER_VERSION}" \
    -o /build/ollarouter \
    ./cmd/ollarouter

# Stage the /data dir so the runtime named volume initializes as nonroot-owned
# (the distroless image has no shell to mkdir/chown at runtime).
RUN mkdir -p /data

# Stage 3: Minimal runtime image
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /build/ollarouter /app/ollarouter
COPY --from=builder /build/providers.yaml.example /app/providers.yaml.example
COPY --from=builder --chown=nonroot:nonroot /data /data
EXPOSE 11435
USER nonroot:nonroot
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/app/ollarouter", "-healthcheck"]
ENTRYPOINT ["/app/ollarouter"]
