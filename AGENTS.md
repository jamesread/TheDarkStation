# Agent Notes

This file contains important notes and patterns for AI agents working on this codebase.

## Map coordinates

When discussing level layout, hazards, doors, and entities with the player or in debug analysis, use **`x:… y:…`** (horizontal, then vertical):

- **x** = column index (0-based, increases east/right)
- **y** = row index (0-based, increases south/down)

Example: gas at **x:37 y:30** is grid cell `(row=30, col=37)`.

The map dump (`map.txt`, F8) uses the same **`x:… y:…`** format and includes an `llm_coordinate_note` explaining the mapping to `Cell.Row` / `Cell.Col` and `Grid.GetCell(row, col)`.

The bottom-right build stamp uses **`BuildLabel`** (friendly local date/time to the minute, e.g. `28 May 2026, 14:35`). Position with the same bottom-aligned formula as below.

## Text Positioning in Ebiten Renderer

### Bottom-Aligned Text Positioning

When positioning text at the bottom of the screen using `drawColoredText()`, use the following formula:

```go
y := screenHeight - margin - int(textHeight * 2)
```

**Important Notes:**
- `drawColoredText()` uses baseline positioning and internally adds `fontSize` to the Y coordinate
- The formula `textHeight * 2` accounts for:
  1. The baseline offset added by `drawColoredText()` (approximately `fontSize`)
  2. The text height below the baseline (descenders and bounding box)
- Always measure the actual text string using `text.Measure(text, face, 0)` to get accurate `textHeight`
- This formula ensures the bottom of the text (including descenders) aligns with `screenHeight - margin`

**Example:**
```go
_, textHeight := text.Measure(versionText, face, 0)
versionY := screenHeight - margin - int(textHeight * 2)
e.drawColoredText(screen, versionText, versionX, versionY, colorSubtle)
```

**See:** `pkg/game/renderer/ebiten/menu.go` - version text positioning in main menu

## Revisit Policy (GDD, Plan Phase 5.4)

The player **cannot revisit** previous decks. The lift is **forward-only**: it has a single destination (the next deck) and no "return" option. The final deck has **no destination** (lift does not advance); reaching it triggers completion. This preserves the intended feel of "moving deeper" and sequential discovery. Per-deck state is stored so the data model could support revisit later; the UI and graph do not expose it.
