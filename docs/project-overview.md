# The Dark Castle — Project Overview

**Date:** 2026-02-01
**Type:** Go (Ebiten game application)
**Architecture:** Monolith, single executable

## Executive Summary

The Dark Castle (The Dark Station) is a relaxing, single-player 2D exploration puzzle game. The player navigates an abandoned space station, restores power to generators, unlocks sealed doors, and finds the lift to the next deck. The codebase is a single Go module using the Ebiten game engine, with engine primitives under `pkg/engine/` and game logic under `pkg/game/`. There is no HTTP API or database; state is in-memory and progression is deck-based (forward-only).

## Project Classification

- **Repository Type:** Monolith
- **Project Type:** backend (Go Ebiten game application)
- **Primary Language:** Go 1.24
- **Architecture Pattern:** Game loop with renderer abstraction; state-centric; deck-based progression

## Technology Stack Summary

| Category | Technology | Version |
|----------|------------|---------|
| Language | Go | 1.24.0 |
| Game engine | Ebiten | v2.9.7 |
| Localization | gotext | v1.7.2 |
| Build / CI | Make, GitHub Actions, goreleaser | — |

See [technology-stack.md](./technology-stack.md) for the full table.

## Documentation Map

- **[index.md](./index.md)** — Master documentation index (primary entry for AI retrieval)
- **[architecture.md](./architecture.md)** — Architecture for main part
- **[source-tree-analysis.md](./source-tree-analysis.md)** — Annotated directory tree
- **[development-guide.md](./development-guide.md)** — Prerequisites, build, test, common tasks
- **[deployment-configuration.md](./deployment-configuration.md)** — CI/CD and release
- **[contribution-guide.md](./contribution-guide.md)** — CoC, PR guidelines, communication
- **[data-models-main.md](./data-models-main.md)** — In-memory state and entities
- **[api-contracts-main.md](./api-contracts-main.md)** — No HTTP API; input/internal flow
- **[existing-documentation-inventory.md](./existing-documentation-inventory.md)** — Inventory of pre-existing docs

## Getting Started

1. **Run:** `git clone ... && cd TheDarkStation && go build -o darkstation main.go && ./darkstation`
2. **Dev start deck:** `./darkstation -level 5` or `LEVEL=5 go run .`
3. **Tests:** `make test`; **lint:** `make codestyle`

See [development-guide.md](./development-guide.md) and the project [README](../README.md).
