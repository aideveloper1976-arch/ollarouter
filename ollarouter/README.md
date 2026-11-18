# OllaRouter service

This folder is the Go module for OllaRouter. Project overview, screenshots and full docs are in the
[repository README](../README.md) and [`docs/`](../docs).

```bash
cp providers.yaml.example providers.yaml   # git-ignored; keep real keys out of it
go run ./cmd/ollarouter -config providers.yaml
```

| Command | Does |
|---|---|
| `make build` | Static binary in `build/ollarouter` with the version from `git describe` |
| `make test` | `go test -race ./...` |
| `make lint` | `gofmt` check and `go vet` |
| `make smoke` | End-to-end smoke test against a mock provider |
| `make webui-build` | Rebuild `../webui` and refresh the embedded copy in `internal/webui/dist` |
| `make docker-build` | Build the image (run from this folder; the build context is the repo root) |

## Keeping keys safe

- Reference keys as `${ENV_VAR}` in `providers.yaml`. They are filled in from the environment
  when the file loads (`internal/config/parser.go`), and `${VAR:-default}` is supported.
- `providers.yaml` and `.env` are git-ignored. `providers.yaml.example` is the shareable template.
- If a key was ever committed or pasted somewhere public, rotate it at the provider.

Set keys in your shell:

```bash
export DEEPSEEK_API_KEY=...        # macOS / Linux
```

```powershell
$env:DEEPSEEK_API_KEY="..."        # Windows PowerShell
```
