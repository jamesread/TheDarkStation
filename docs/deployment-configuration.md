# Deployment Configuration

**Generated:** 2026-02-01  
**Part:** main

## CI/CD

| Pipeline | Trigger | Actions |
|----------|---------|---------|
| **Build** (.github/workflows/build.yml) | Push to main, push tags | Checkout, setup Go from go.mod, `go build`, `go test`, goreleaser (install); on non-tag branches runs semantic-release |
| **Codestyle** (.github/workflows/codestyle.yml) | Push/PR when `**.go`, go.mod, go.sum change | Checkout, setup Go, `go vet ./...`, golangci-lint (latest) |

## Release (Goreleaser)

- **Config:** .goreleaser.yml  
- **Binary:** darkstation (main.go)  
- **Platforms:** Linux (amd64, arm64, arm6/7), Windows (amd64). Darwin and Windows arm/arm64 ignored.  
- **Ldflags:** `-s -w -X main.version={{.Version}} -X main.commit={{.ShortCommit}} -X main.date={{.CommitDate}}`  
- **Artifacts:** Checksums; snapshot template `{{.Branch}}-{{.ShortCommit}}`  
- **Changelog:** From git (see .goreleaser.yml)

## Local Run

- **Default:** `make` or `go run .`  
- **Build:** `make build` → `go build -o darkstation main.go`  
- **Test:** `make test` → `go test ./...`  
- **Dev:** `LEVEL=2` or `-level=2` for starting deck (dev testing)

No Docker, Kubernetes, or deployment manifests in repo; distribution is via Go build and goreleaser artifacts.
