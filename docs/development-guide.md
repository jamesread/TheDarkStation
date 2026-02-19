# Development Guide

**Generated:** 2026-02-01  
**Part:** main

## Prerequisites

- **Go:** 1.24.x (see `go.mod`; toolchain 1.24.9)
- **Optional:** `msgfmt` for i18n (compile `.mo` from `po/`); golangci-lint for codestyle

## Environment

- No `.env` or runtime config files required for local run.
- **Dev testing:** `LEVEL` (env) or `-level N` (flag) to start at deck N (e.g. `LEVEL=2` or `./darkstation -level 5`).

## Installation (from source)

```bash
git clone https://github.com/jamesread/TheDarkStation.git
cd TheDarkStation
go build -o darkstation main.go
./darkstation
```

Or use Make: `make build` then `./darkstation`.

## Local Development Commands

| Command | Purpose |
|--------|---------|
| `make` or `go run .` | Run the game (default target) |
| `make build` | Build binary to `darkstation` |
| `make test` | Run tests: `go test ./...` |
| `make codestyle` | Format and lint: `go fmt ./...`, `golangci-lint run` |
| `make mo` | Compile translations: `msgfmt -c -v po/default.pot -o mo/en_GB.utf8/LC_MESSAGES/default.mo` |
| `make clean` | Remove `darkstation` and `dist/` |

## Build Process

- Single binary: `go build -o darkstation main.go`.
- Release builds use goreleaser (see `.goreleaser.yml`): ldflags set `main.version`, `main.commit`, `main.date`; targets Linux/Windows (amd64, arm64; arm6/7 on Linux).

## Testing

- **Command:** `go test ./...` or `make test`.
- **Pattern:** `*_test.go` alongside source (e.g. `pkg/game/setup/helpers_test.go`).
- CI runs tests on push (`.github/workflows/build.yml`).

## Common Development Tasks

- **Run at a specific deck:** `./darkstation -level 5` or `LEVEL=5 go run .`
- **Update translations:** Edit `po/default.pot`, run `make mo`
- **Lint before PR:** `make codestyle`
- **Install golangci-lint:** `make go-tools` (installs to `$(go env GOPATH)/bin`)
