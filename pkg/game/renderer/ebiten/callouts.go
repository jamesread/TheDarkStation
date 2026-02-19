// Package ebiten provides an Ebiten-based 2D graphical renderer for The Dark Station.
package ebiten

import (
	"image/color"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// AddCallout adds a floating message callout near a specific cell
func (e *EbitenRenderer) AddCallout(row, col int, message string, col_color color.Color, durationMs int) {
	e.calloutsMutex.Lock()
	defer e.calloutsMutex.Unlock()

	var expiresAt int64
	if durationMs > 0 {
		expiresAt = time.Now().UnixMilli() + int64(durationMs)
	}

	// Remove any existing callout at the same position
	filtered := make([]Callout, 0)
	for _, c := range e.callouts {
		if c.Row != row || c.Col != col {
			filtered = append(filtered, c)
		}
	}

	now := time.Now().UnixMilli()
	filtered = append(filtered, Callout{
		Row:       row,
		Col:       col,
		Message:   message,
		Color:     col_color,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	})
	e.callouts = filtered
}

// AddCalloutAtPlayer adds a callout at the player's current position
func (e *EbitenRenderer) AddCalloutAtPlayer(message string, col color.Color, durationMs int) {
	e.snapshotMutex.RLock()
	row, col_pos := e.snapshot.playerRow, e.snapshot.playerCol
	e.snapshotMutex.RUnlock()
	e.AddCallout(row, col_pos, message, col, durationMs)
}

// AddCalloutNearPlayer adds a callout at an adjacent cell (for interactions)
func (e *EbitenRenderer) AddCalloutNearPlayer(row, col int, message string, col_color color.Color, durationMs int) {
	e.AddCallout(row, col, message, col_color, durationMs)
}

// ClearCallouts removes all active callouts
func (e *EbitenRenderer) ClearCallouts() {
	e.calloutsMutex.Lock()
	defer e.calloutsMutex.Unlock()
	e.callouts = nil
}

// ClearCalloutsIfMoved clears callouts if player has moved from tracked position
// Returns true if callouts were cleared
func (e *EbitenRenderer) ClearCalloutsIfMoved(row, col int) bool {
	if !e.lastPosInitialized {
		return false
	}
	if e.lastPlayerRow != row || e.lastPlayerCol != col {
		e.calloutsMutex.Lock()
		e.callouts = nil
		e.calloutsMutex.Unlock()
		return true
	}
	return false
}

// ShowRoomEntryIfNew shows a room entry callout if the player entered a new room
// Skips corridors and returns true if a callout was shown
func (e *EbitenRenderer) ShowRoomEntryIfNew(row, col int, roomName string) bool {
	// Skip if room name hasn't changed
	if e.lastRoomName == roomName {
		return false
	}

	// Update tracked room name
	oldRoom := e.lastRoomName
	e.lastRoomName = roomName

	// Skip corridors
	lowerName := strings.ToLower(roomName)
	if strings.Contains(lowerName, "corridor") || strings.Contains(lowerName, "hallway") {
		return false
	}

	// Skip if this is the first room (game just started)
	if oldRoom == "" {
		return false
	}

	// Room entry callout removed - no longer showing room titles on entry
	return true
}

// SetDebounceAnimation triggers a debounce animation in the given direction
func (e *EbitenRenderer) SetDebounceAnimation(direction string) {
	e.debounceMutex.Lock()
	defer e.debounceMutex.Unlock()
	e.debounceDirection = direction
	e.debounceStartTime = time.Now().UnixMilli()
}

// drawCallouts renders floating message callouts near cells
func (e *EbitenRenderer) drawCallouts(screen *ebiten.Image, snap *renderSnapshot, mapX, mapY float64, startRow, startCol int) {
	if len(snap.callouts) == 0 {
		return
	}

	fontSize := e.getUIFontSize()
	titleFace := e.getSansBoldTitleFontFace()
	titleFontSize := fontSize + 2
	padding := 6
	now := time.Now().UnixMilli()

	// Animation timing constants
	const (
		entranceDuration = 200 // milliseconds for entrance animation
		exitDuration     = 200 // milliseconds for exit animation
	)

	for _, callout := range snap.callouts {
		// Calculate screen position from cell position
		vRow := callout.Row - startRow
		vCol := callout.Col - startCol

		// Skip if outside viewport
		if vRow < 0 || vRow >= e.viewportRows || vCol < 0 || vCol >= e.viewportCols {
			continue
		}

		// Calculate animation progress
		age := now - callout.CreatedAt
		var alpha float64 = 1.0
		var slideOffsetY float32 = 0.0

		// Entrance animation (fade in from black + slide in from top)
		if age < entranceDuration {
			progress := float64(age) / entranceDuration
			alpha = progress                               // Fade in from 0 to 1
			slideOffsetY = float32(-20 * (1.0 - progress)) // Slide in from 20px above
		}

		// Exit animation (fade out to black + slide out to bottom)
		if callout.ExpiresAt > 0 {
			timeUntilExpiry := callout.ExpiresAt - now
			if timeUntilExpiry < exitDuration && timeUntilExpiry > 0 {
				progress := float64(timeUntilExpiry) / exitDuration
				alpha = progress                              // Fade out from 1 to 0
				slideOffsetY = float32(20 * (1.0 - progress)) // Slide out to 20px below
			} else if timeUntilExpiry <= 0 {
				continue // Skip expired callouts
			}
		}

		// Calculate pixel position (center of the cell)
		cellX := mapX + float64(vCol*e.tileSize)
		cellY := mapY + float64(vRow*e.tileSize)

		// Split message by newlines to handle multi-line callouts
		lines := strings.Split(callout.Message, "\n")
		// Optional title: first line uses title styling (bold, larger, colorAction) only when TITLE{} markup is present
		hasTitle := len(lines) > 0 && hasTitleMarkup(lines[0])
		maxTextWidth := 0.0
		for lineIdx, line := range lines {
			lineSegments := e.parseMarkup(line)
			lineWidth := 0.0
			face := e.getSansFontFace()
			if lineIdx == 0 && hasTitle {
				face = titleFace
			}
			for _, seg := range lineSegments {
				lineWidth += e.getTextWidthWithFace(seg.text, face)
			}
			if lineWidth > maxTextWidth {
				maxTextWidth = lineWidth
			}
		}
		textWidth := maxTextWidth
		firstLineHeight := int(titleFontSize) + 4
		otherLineHeight := int(fontSize) + 4
		var boxHeight int
		if hasTitle {
			boxHeight = firstLineHeight
			if len(lines) > 1 {
				boxHeight += (len(lines) - 1) * otherLineHeight
			}
		} else {
			boxHeight = len(lines) * otherLineHeight
		}
		boxHeight += padding * 2

		// Determine base position (to the right or left of cell)
		boxWidth := int(textWidth) + padding*2
		baseCalloutX := cellX + float64(e.tileSize) + 8

		// If callout would go off right edge, position to the left instead
		if baseCalloutX+float64(boxWidth) > mapX+float64(e.viewportCols*e.tileSize) {
			baseCalloutX = cellX - float64(boxWidth) - 8
		}

		// Check if callout would overlap with player icon
		// Player position in viewport coordinates
		playerVRow := snap.playerRow - startRow
		playerVCol := snap.playerCol - startCol
		playerX := mapX + float64(playerVCol*e.tileSize)
		playerY := mapY + float64(playerVRow*e.tileSize)

		// Calculate callout box bounds
		baseCalloutY := cellY + float64((e.tileSize-boxHeight)/2)
		calloutBoxLeft := float32(baseCalloutX)
		calloutBoxRight := float32(baseCalloutX + float64(boxWidth))
		calloutBoxTop := float32(baseCalloutY)
		calloutBoxBottom := float32(baseCalloutY + float64(boxHeight))

		// Check if callout overlaps with player icon (player icon is roughly centered in its tile)
		playerIconLeft := float32(playerX + float64(e.tileSize/4))
		playerIconRight := float32(playerX + float64(e.tileSize*3/4))
		playerIconTop := float32(playerY + float64(e.tileSize/4))
		playerIconBottom := float32(playerY + float64(e.tileSize*3/4))

		overlapsPlayer := calloutBoxLeft < playerIconRight && calloutBoxRight > playerIconLeft &&
			calloutBoxTop < playerIconBottom && calloutBoxBottom > playerIconTop

		// If overlapping, move callout back a column and down a row
		if overlapsPlayer {
			// Move back a column (left if on right side, right if on left side)
			if baseCalloutX > cellX+float64(e.tileSize) {
				// Callout is on the right, move it further right (back a column)
				baseCalloutX = cellX
			} else {
				// Callout is on the left, move it further left (back a column)
			}
			// Move down a row
			baseCalloutY = cellY + float64(e.tileSize) + float64((e.tileSize-boxHeight)/2)
		}

		// Apply slide animation offset (vertical only)
		calloutX := float32(baseCalloutX)
		calloutY := float32(baseCalloutY) + slideOffsetY

		// Keep callout within vertical bounds (check after applying slide offset)
		if calloutY < float32(mapY) {
			calloutY = float32(mapY)
		}
		if calloutY+float32(boxHeight) > float32(mapY+float64(e.viewportRows*e.tileSize)) {
			calloutY = float32(mapY + float64(e.viewportRows*e.tileSize) - float64(boxHeight))
		}

		// Skip drawing if alpha is too low (avoid rendering artifacts)
		if alpha < 0.01 {
			continue
		}

		// Apply alpha to colors (fade from black/transparent, not white)
		// The applyAlpha function multiplies the alpha channel, so colors fade to transparent black
		bgColor := e.applyAlpha(color.RGBA{15, 15, 25, 240}, alpha)
		borderColor := e.applyAlpha(color.RGBA{80, 80, 100, 255}, alpha)
		if hasTitle {
			titleColor := e.getTitleColorFromLine(lines[0])
			borderColor = e.applyAlpha(titleColor, alpha)
		}

		boxW := float32(boxWidth)
		boxH := float32(boxHeight)
		const tooltipCornerRadius = 6
		const tooltipBorderWidth = 1
		drawRoundedRectWithShadow(screen, calloutX, calloutY, boxW, boxH, tooltipCornerRadius, tooltipBorderWidth, bgColor, borderColor, float32(alpha))

		// Draw pointer/arrow toward the cell
		arrowSize := float32(6)
		arrowY := calloutY + float32(boxHeight/2)
		if calloutX > float32(cellX+float64(e.tileSize)) {
			// Arrow pointing left
			arrowX := calloutX - 1
			vector.DrawFilledRect(screen, arrowX-arrowSize, arrowY-2, arrowSize, 4, borderColor, false)
		} else {
			// Arrow pointing right
			arrowX := calloutX + float32(boxWidth) + 1
			vector.DrawFilledRect(screen, arrowX, arrowY-2, arrowSize, 4, borderColor, false)
		}

		// Draw text - first line uses title font when TITLE{} present, else body font
		lineFontSize := fontSize
		if hasTitle {
			lineFontSize = titleFontSize
		}
		startY := int(calloutY) + padding - int(lineFontSize)

		for i, line := range lines {
			lineSegments := e.parseMarkup(line)
			fadedSegments := make([]textSegment, len(lineSegments))
			for j, seg := range lineSegments {
				fadedSegments[j] = textSegment{
					text:  seg.text,
					color: e.applyAlpha(seg.color, alpha),
				}
			}
			face := e.getSansFontFace()
			if i == 0 && hasTitle {
				face = titleFace
			}
			var textY int
			if i == 0 {
				textY = startY
			} else if hasTitle {
				textY = startY + firstLineHeight + (i-1)*otherLineHeight
			} else {
				textY = startY + i*otherLineHeight
			}
			e.drawColoredTextSegmentsWithFace(screen, fadedSegments, int(calloutX)+padding, textY, face)
		}
	}
}
