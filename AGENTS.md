# Repository Guidelines

## Project Structure & Module Organization
- `cmd/`: Service entrypoint(s). `go run ./cmd` starts the server.
- `internal/`: Core business logic and server implementation (non-public packages).
- `pkg/`: Reusable, public packages used by the server.
- `config/`: Configuration defaults and example config (`config.yaml`).
- `web/` and `webassets/`: Web client and static assets for the VTuber UI.
- `live2d-models/` and `model_dict.json`: Model assets and mapping data.

## Build, Test, and Development Commands
- `go run ./cmd` — Run the server directly from source.
- `./start.sh` — Start the server with port checks; respects `MIO_ROOT_DIR`, `CONFIG_PATH`, `MIO_SERVER_PORT`, and `DEFAULT_PORT`.
- `go test ./...` — Run all Go tests (currently none in repo).

## Coding Style & Naming Conventions
- Use idiomatic Go and keep code readable and small in scope.
- Run `gofmt` (e.g., `gofmt -w .`) before committing.
- Add or update **English** doc comments for exported symbols (per `CONTRIBUTING.md`).
- Prefer clear package names and avoid abbreviations unless standard in Go.

## Testing Guidelines
- Tests should use Go’s standard `testing` package and live alongside code as `*_test.go`.
- Name tests with `TestXxx` and table tests with subtests (`t.Run`).
- Add tests when changing logic in `internal/` or `pkg/`.

## Commit & Pull Request Guidelines
- Recent commits follow a conventional style like `chore: ...`; use this pattern when possible.
- Keep PRs focused and include a short description of the change and its impact.
- Link issues when relevant and note any config changes or new assets.

## Security & Configuration Tips
- Follow `SECURITY.md` for private vulnerability reporting.
- Place runtime config in `config/` and avoid committing secrets. Use env vars for local overrides.
