// Package features provides runtime feature toggles for gameplay systems.
package features

import (
	engworld "darkstation/pkg/engine/world"
	"darkstation/pkg/game/renderer"
)

// VisitedSystemEnabled reports whether the visited-cell system is active (cvar gameplay.visited).
func VisitedSystemEnabled() bool {
	return renderer.VisitedSystemEnabled()
}

// MarkVisited sets cell.Visited when the visited system is enabled.
func MarkVisited(cell *engworld.Cell) {
	if cell == nil || !VisitedSystemEnabled() {
		return
	}
	cell.Visited = true
}

// IsVisited returns whether a cell should be treated as visited for gameplay/rendering.
func IsVisited(cell *engworld.Cell) bool {
	return cell != nil && VisitedSystemEnabled() && cell.Visited
}
