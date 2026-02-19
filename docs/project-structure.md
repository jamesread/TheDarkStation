# Project Structure

**Generated:** 2026-02-01
**Scan level:** Deep
**Repository type:** Monolith

## Classification

| Part ID | Root path | Project type | Display name |
|---------|-----------|--------------|--------------|
| main | / (project root) | backend | Go (Ebiten game application) |

## Directory overview

- **Root:** `main.go`, `go.mod`, `Makefile`, `darkcastle.tsx`, config/tooling files
- **pkg/engine:** Generic engine primitives (world, cell, grid, terminal, input)
- **pkg/game:** Game-specific code (state, renderer/ebiten, menu, gameplay, generator, setup, entities)
- **pkg/resources:** Fonts and resources
- **specs/:** GDD, power-system, level-layout, deck plan
- **_bmad/:** BMad/GDS workflows and config (not application code)
- **.github/:** CI workflows

## Key files

- `main.go` — Application entry point
- `go.mod` — Go module (darkstation)
- `pkg/game/state/state.go` — Game state
- `pkg/game/renderer/ebiten/` — Ebiten renderer
- `pkg/engine/world/` — Grid and cell types
