// Package renderer provides the rendering abstraction layer for the game.
// It defines the Renderer interface and provides a pluggable architecture
// for different rendering backends (currently only Ebiten is supported).
package renderer

import (
	"fmt"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/state"
)

// FormatPowerWatts returns markup for power display: POWERED{Xw} when w>0,
// UNPOWERED{Xw} or UNPOWERED_SUBTLE{Xw} when w<=0. useSubtle for dependency-blocked (e.g. room terminal off).
func FormatPowerWatts(w int, useSubtle bool) string {
	s := fmt.Sprintf("%dw", w)
	if w > 0 {
		return fmt.Sprintf("POWERED{%s}", s)
	}
	if useSubtle {
		return fmt.Sprintf("UNPOWERED_SUBTLE{%s}", s)
	}
	return fmt.Sprintf("UNPOWERED{%s}", s)
}

// ApplyMarkup formats a string with special markup using the current renderer.
// This is a convenience function for backwards compatibility.
func ApplyMarkup(msg string, a ...any) string {
	return FormatText(msg, a...)
}

// PrintString prints a formatted string (kept for backwards compatibility)
// For new code, prefer using RenderFrame or the renderer interface directly.
func PrintString(msg string, a ...any) {
	if Current != nil {
		Current.ShowMessage(Current.FormatText(msg, a...))
	}
}

// Helper functions that delegate to the current renderer for styled text.
// These provide a convenient API for game logic that needs to create styled strings.

// StyledCell returns text styled as a cell/room name
func StyledCell(text string) string {
	return StyleText(text, StyleCell)
}

// StyledItem returns text styled as an item
func StyledItem(text string) string {
	return StyleText(text, StyleItem)
}

// StyledAction returns text styled as an action
func StyledAction(text string) string {
	return StyleText(text, StyleAction)
}

// StyledDenied returns text styled as denied/error
func StyledDenied(text string) string {
	return StyleText(text, StyleDenied)
}

// StyledKeycard returns text styled as a keycard
func StyledKeycard(text string) string {
	return StyleText(text, StyleKeycard)
}

// StyledDoor returns text styled as a door
func StyledDoor(text string) string {
	return StyleText(text, StyleDoor)
}

// StyledHazard returns text styled as a hazard
func StyledHazard(text string) string {
	return StyleText(text, StyleHazard)
}

// StyledHazardCtrl returns text styled as a hazard control
func StyledHazardCtrl(text string) string {
	return StyleText(text, StyleHazardCtrl)
}

// StyledFurniture returns text styled as unchecked furniture
func StyledFurniture(text string) string {
	return StyleText(text, StyleFurniture)
}

// StyledFurnitureChecked returns text styled as checked furniture
func StyledFurnitureChecked(text string) string {
	return StyleText(text, StyleFurnitureChecked)
}

// StyledSubtle returns text styled as subtle/gray
func StyledSubtle(text string) string {
	return StyleText(text, StyleSubtle)
}

// CanEnterCell checks if the player can enter a cell (without logging)
// This is game logic that doesn't belong in a specific renderer
func CanEnterCell(g *state.Game, r *world.Cell) (bool, *world.ItemSet) {
	missingItems := world.NewItemSet()

	if r == nil || !r.Room {
		return false, missingItems
	}

	r.RequiredItems.Each(func(reqItem *world.Item) {
		if !g.OwnedItems.Has(reqItem) {
			missingItems.Put(reqItem)
		}
	})

	return missingItems.Size() == 0, missingItems
}

// SetJoin joins item names from a set with commas
func SetJoin(set *world.ItemSet) string {
	ret := ""

	set.Each(func(i *world.Item) {
		ret += i.Name + ","
	})

	// Remove trailing comma
	if len(ret) > 0 {
		ret = ret[:len(ret)-1]
	}

	return ret
}
