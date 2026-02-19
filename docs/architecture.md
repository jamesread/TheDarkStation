# Architecture — Main

**Generated:** 2026-02-01
**Part:** main

## Executive Summary

The Dark Castle (The Dark Station) is a single-player 2D exploration puzzle game. It is a single Go executable using the Ebiten game engine. Engine primitives (grid, cell, FOV, input) live under `pkg/engine/`; all game logic (state, rendering, menu, gameplay, level generation, setup, entities) lives under `pkg/game/`. There is no HTTP API, no database, and no multi-part deployment; state is in-memory and progression is deck-based with a forward-only lift.

## Technology Stack

| Category | Technology | Version |
|----------|------------|---------|
| Language | Go | 1.24.0 (toolchain 1.24.9) |
| Module | darkstation | — |
| Graphics / Game | Ebiten | v2.9.7 |
| Localization | gotext (leonelquinteros) | v1.7.2 |
| Terminal / CLI | golang.org/x/term | v0.39.0 |
| Build / Lint | Make, golangci-lint | — |
| CI | GitHub Actions | — |

See [technology-stack.md](./technology-stack.md) for full table and justification.

## Architecture Pattern

- **Style:** Game application with a single Ebiten-based renderer; clear separation between engine primitives (`pkg/engine/`) and game logic (`pkg/game/`).
- **Entry:** `main.go` → init gettext, Ebiten renderer, then game loop: main menu → build game (generate or load) → run until exit.
- **State:** Central game state in `pkg/game/state/`; deck-based progression and level setup in `pkg/game/setup/` and `pkg/game/deck/`. Per-deck state is stored; UI and graph are forward-only (no revisit).
- **Rendering:** Abstract renderer in `pkg/game/renderer/` with Ebiten implementation in `renderer/ebiten/`.

See [architecture-patterns.md](./architecture-patterns.md) for more detail.

## Data Architecture

No database or migrations. All state is in-memory:

- **Core state:** `Game`, `DeckState`, `MessageEntry` in `pkg/game/state/state.go`.
- **World/cell:** `GameCellData` in `pkg/game/world/cell.go`; engine types in `pkg/engine/world/` (Grid, Cell, Item, FOV).
- **Entities:** Generator, Door, CCTVTerminal, PuzzleTerminal, Furniture, Hazard, HazardControl, MaintenanceTerminal in `pkg/game/entities/`.

Relationships are by reference (pointers). See [data-models-main.md](./data-models-main.md).

## API Design

No HTTP/REST or RPC API. Interaction is via keyboard/mouse input and internal state. See [api-contracts-main.md](./api-contracts-main.md).

## Source Tree

- **Entry:** `main.go`.
- **Engine:** `pkg/engine/` — world (grid, cell, FOV, direction, item), input (tiered), terminal.
- **Game:** `pkg/game/` — state, deck, renderer/ebiten, menu, gameplay, generator, levelgen, setup, entities, world, config, devtools.
- **Specs:** `specs/` — GDD, power-system, level-layout, deck plan.

See [source-tree-analysis.md](./source-tree-analysis.md) for the full annotated tree.

## Development Workflow

- **Run:** `make` or `go run .`; dev start deck: `LEVEL=N` or `-level N`.
- **Build/test:** `make build`, `make test`.
- **Lint:** `make codestyle`.
- **i18n:** `make mo` after editing `po/default.pot`.

See [development-guide.md](./development-guide.md).

## Deployment Architecture

- **CI:** GitHub Actions — Build (go build, test, goreleaser/semantic-release on main/tags), Codestyle (vet, golangci-lint on Go file changes).
- **Release:** Goreleaser — Linux/Windows (amd64, arm64; Linux arm6/7), ldflags for version/commit/date; no Docker/K8s in repo.

See [deployment-configuration.md](./deployment-configuration.md) and [contribution-guide.md](./contribution-guide.md).

## Testing Strategy

- **Command:** `go test ./...` or `make test`.
- **Location:** `*_test.go` alongside source.
- **CI:** Tests run on push (build workflow).
