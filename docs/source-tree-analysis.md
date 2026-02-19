# The Dark Castle — Source Tree Analysis

**Date:** 2026-02-01

## Overview

Single Go module (`darkstation`) with engine primitives under `pkg/engine/` and game logic under `pkg/game/`. Entry point is `main.go`; Ebiten runs the game loop (menu → game → quit). No multi-part structure.

## Complete Directory Structure

```
project-root/
├── main.go                    # Application entry; init gettext, Ebiten, game loop
├── go.mod
├── Makefile
├── darkcastle.tsx             # (Tiled/TSX reference; not runtime)
├── map.txt
├── translations.go            # Embedded .mo for i18n
├── po/                        # gettext source
│   └── default.pot
├── mo/                        # Compiled .mo (en_GB)
│   └── en_GB.utf8/LC_MESSAGES/default.mo
├── res/fonts/                 # Font assets
├── pkg/
│   ├── engine/                # Generic engine primitives (world, input, terminal)
│   │   ├── input/             # Tiered input handling
│   │   ├── terminal/          # Terminal abstraction
│   │   └── world/             # Grid, Cell, FOV, Direction, Item
│   ├── game/                  # Game-specific code
│   │   ├── config/            # Game config
│   │   ├── deck/              # Deck graph and navigation (forward-only)
│   │   ├── devtools/          # Dev map, mapdump, screenshot
│   │   ├── entities/          # Doors, generators, terminals, furniture, hazards
│   │   ├── gameplay/          # Input, movement, interactions, lifecycle, lighting, hints
│   │   ├── generator/         # BSP level generator, line walker
│   │   ├── levelgen/          # Furniture, hazards, maintenance, puzzles placement
│   │   ├── menu/              # Main menu, bindings, gameplay/maintenance menus
│   │   ├── renderer/          # Renderer interface + Ebiten implementation
│   │   │   └── ebiten/        # Ebiten-specific rendering, input, menu, text
│   │   ├── setup/             # Level setup: batteries, doors, generators, room power, solvability, terminals
│   │   ├── state/             # Game state, deck state, messages
│   │   └── world/             # GameCellData, cell helpers (extends engine/world)
│   └── resources/             # Font loading (fonts.go, fonts/)
├── specs/                     # GDD, power-system, level-layout, deck plan
├── .github/workflows/         # Build, codestyle
├── _bmad/                     # BMad/GDS workflows (not application code)
└── docs/                      # Generated documentation
```

## Critical Directories

### `main.go`
Application entry. Inits gettext, Ebiten renderer, then runs loop: main menu → build game (generate or load) → run until quit. `LEVEL` / `-level` for dev start deck.

### `pkg/engine/`
Engine primitives shared across game logic: world (grid, cell, FOV, direction, item), input (tiered), terminal. No game-specific types.

### `pkg/engine/world/`
Grid, Cell, FOV, Direction, Item. Used by `pkg/game/state` and `pkg/game/world`.

### `pkg/game/state/`
Core game state: `Game`, `DeckState`, messages, power, deck index, completion. Entry point for all state reads/writes.

### `pkg/game/renderer/` and `pkg/game/renderer/ebiten/`
Renderer interface and Ebiten implementation. Drawing, input, menu UI, text, callouts.

### `pkg/game/gameplay/`
Input handling, movement, interactions, lifecycle, lighting, hints. Drives in-game behavior.

### `pkg/game/generator/` and `pkg/game/levelgen/`
BSP level generation; placement of furniture, hazards, maintenance, puzzles.

### `pkg/game/setup/`
Level setup: batteries, doors, generators, room power, solvability, terminals. Runs after level generation.

### `pkg/game/entities/`
Entity types: Door, Generator, CCTVTerminal, PuzzleTerminal, Furniture, Hazard, HazardControl, MaintenanceTerminal.

### `pkg/game/deck/`
Deck graph and navigation (forward-only lift; no revisit).

### `pkg/game/menu/`
Main menu, bindings, gameplay and maintenance menus.

### `pkg/game/world/`
GameCellData and cell helpers; extends engine/world with game entities per cell.

### `specs/`
Design docs: GDD, power-system, level-layout, deck plan.

### `.github/workflows/`
CI: build (go build, test, goreleaser/semantic-release), codestyle (vet, golangci-lint).

## Entry Points

- **Main entry:** `main.go` — inits gettext, renderer, then Ebiten game loop (menu → game → quit).
- **No other runnable entry points.**

## File Organization Patterns

- **Go packages:** One directory per package; `pkg/engine` vs `pkg/game` separates engine from game.
- **Tests:** `*_test.go` alongside source (e.g. `pkg/game/setup/helpers_test.go`).
- **Config/tooling:** Root: `go.mod`, `Makefile`, `.goreleaser.yml`, `.github/`, `.golangci.yml`, `.releaserc.yaml`.
- **i18n:** `po/`, `mo/`, `translations.go` for embedded .mo.

## Asset Locations

- **Fonts:** `res/fonts/`, `pkg/resources/fonts/` (CascadiaCodeNF-Regular.otf).
- **Tileset:** `var/tileset.png` (and .xcf); map data in root (e.g. `1.1.spawn.*`, `map.txt`).

## Configuration Files

- **Module:** `go.mod` — Go 1.24, module darkstation, dependencies.
- **Build/lint:** `Makefile` (default, mo, build, codestyle, test), `.golangci.yml`, `.goreleaser.yml`, `.releaserc.yaml`.
- **Game config:** `pkg/game/config/config.go`.

## Notes for Development

- Run: `make` or `go run .`; dev start deck: `LEVEL=2` or `-level=2`.
- Build/test: `make build`, `make test`; codestyle: `make codestyle`.
- Translations: `make mo` after editing `po/default.pot`.
- Docs: Generated under `docs/`; master index `docs/index.md` (created in later step).
