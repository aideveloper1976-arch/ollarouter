# Contributing

Thanks for helping. Bug reports, provider adapters and docs fixes are all welcome.

## Set up

```bash
git clone https://github.com/aideveloper1976-arch/ollarouter.git
cd ollarouter/ollarouter
go test -race ./...
```

Go 1.23+ is needed. Node 20+ only if you change `webui/`.

## Before you open a pull request

From `ollarouter/`:

```bash
make lint          # gofmt + go vet
go test -race ./...
make smoke         # end-to-end check (needs Python 3 and curl)
```

If you changed the admin UI, rebuild it and commit the embedded output:

```bash
make webui-build   # builds webui/ and copies dist into internal/webui/dist
```

CI runs all of these, plus a Docker build, gitleaks and CodeQL.

## Adding a provider

1. Add an adapter under `internal/provider/<name>/` that implements `types.Provider`
   (`Chat`, `ChatStream`, `Generate`, `GenerateStream`, `ListModels`, `CheckHealth`).
2. Register its `type` in `internal/provider/factory.go`.
3. Add `httptest`-based tests that copy the provider's real wire format, including streaming.
4. Add an example to `ollarouter/providers.yaml.example` and a row to `docs/configuration.md`.

## Ground rules

- Never commit real keys. Use `${ENV_VAR}` placeholders in examples.
- Keep claims in docs checkable. If you add a feature, add the test that proves it.
- One focused change per pull request, with a clear description of the behaviour change.
