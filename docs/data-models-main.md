# Data Models — Main

**Generated:** 2026-02-01  
**Part:** main

## Overview

No database or migrations. All state is in-memory: core game state in `pkg/game/state/`, world/cell data in `pkg/game/world/`, and entities in `pkg/game/entities/`. Engine primitives (grid, cell, FOV) live in `pkg/engine/world/`.

## Core State

| Model | Location | Purpose |
|-------|----------|---------|
| `Game` | pkg/game/state/state.go | Root game state: current cell, grid, deck, batteries, messages, power, room power, completion flags |
| `DeckState` | pkg/game/state/state.go | Per-deck state: grid, seed, room doors/CCTV powered, generators |
| `MessageEntry` | pkg/game/state/state.go | Message text + timestamp |

## World / Cell

| Model | Location | Purpose |
|-------|----------|---------|
| `GameCellData` | pkg/game/world/cell.go | Per-cell game data: generator, door, terminals, furniture, hazard, hazard control, lights |

Engine types: `world.Cell`, `world.Grid`, `world.Item`, `world.ItemSet` (pkg/engine/world).

## Entities

| Model | Location | Purpose |
|-------|----------|---------|
| `Generator` | pkg/game/entities/generator.go | Power generator |
| `Door` | pkg/game/entities/door.go | Keycard door |
| `CCTVTerminal` | pkg/game/entities/terminal.go | CCTV terminal |
| `PuzzleTerminal` | pkg/game/entities/puzzle.go | Puzzle terminal |
| `Furniture` / `FurnitureTemplate` | pkg/game/entities/furniture.go | Furniture and templates |
| `Hazard` / `HazardControl` / `HazardInfo` | pkg/game/entities/hazard.go | Hazard and control panel |
| `MaintenanceTerminal` / `DeviceInfo` | pkg/game/entities/maintenance.go | Maintenance terminal and device info |

Relationships are by reference (pointers) from `Game`, `DeckState`, and `GameCellData`; no ORM or schema.
