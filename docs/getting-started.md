# Getting started

## Option 1: run from source

Requirements: Go 1.23 or newer. Node 20+ is only needed to change the admin UI. The built UI is
already committed and embedded in the binary.

```bash
git clone https://github.com/aideveloper1976-arch/ollarouter.git
cd ollarouter/ollarouter
cp providers.yaml.example providers.yaml
```

Edit `providers.yaml` and delete any providers you don't use. Keys stay out of the file and are
read from the environment through `${VAR}` placeholders:

```bash
export OPENAI_API_KEY=...
export ANTHROPIC_API_KEY=...
export OLLAROUTER_API_KEY=$(openssl rand -hex 32)   # what your clients will send
```

Run it:

```bash
go run ./cmd/ollarouter -config providers.yaml
# or: make build && ./build/ollarouter -config providers.yaml
```

OllaRouter listens on port `11435` by default, so it can sit next to a real Ollama on `11434`.

## Option 2: Docker Compose (with Open WebUI)

From the repository root:

```bash
cp .env.example .env                                   # fill in the keys you use
cp ollarouter/providers.yaml.example ollarouter/providers.yaml
docker compose up -d --build
```

| Service | URL |
|---|---|
| OllaRouter API and admin console | http://localhost:11434 and http://localhost:11434/admin/ |
| Open WebUI | http://localhost:3000 |

In Compose, OllaRouter listens on `11434`, the port Ollama clients expect. `OLLAROUTER_PORT`
overrides `settings.port`. The image is built in three stages (Node for the UI, Go for the binary,
then distroless `static-debian12:nonroot`), runs as a non-root user and has a built-in
`-healthcheck`.

> **Open WebUI and API keys.** The bundled Open WebUI runs with `WEBUI_AUTH=False` and talks to
> OllaRouter's Ollama API. If you configure `api_keys`, also give Open WebUI that key in its
> Ollama connection settings. Keep this setup on a trusted machine or network.

## Connect your tools

| Client | Setting |
|---|---|
| Open WebUI, Page Assist, Continue (Ollama provider) | Ollama URL `http://localhost:11435` |
| LangChain / LlamaIndex `Ollama` classes | `base_url="http://localhost:11435"` |
| OpenAI SDKs, VS Code OpenAI-compatible providers | base URL `http://localhost:11435/v1`, API key = an OllaRouter key |
| curl | `-H "Authorization: Bearer $OLLAROUTER_API_KEY"` |

Model names are `provider/model` (for example `openai-main/gpt-4o`), a model's short name if only
one provider offers it, or a virtual model name such as `best-quality`.

## Check it works

```bash
curl -s localhost:11435/api/version
curl -s localhost:11435/api/tags | head -c 300
curl -s localhost:11435/health
```

`/health` returns `503` with `"status":"degraded"` until at least one provider is online.

## Run the tests

```bash
cd ollarouter
go test -race ./...           # unit and integration tests
make lint                     # gofmt + go vet
make smoke                    # end-to-end smoke test against a mock provider
```
