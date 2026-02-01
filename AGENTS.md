# Agent Notes

This file contains important notes and patterns for AI agents working on this codebase.

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
