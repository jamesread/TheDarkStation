package menu

import (
	"fmt"
	"strings"

	engineinput "darkstation/pkg/engine/input"
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
)

type InventoryItem struct {
	Label string
}

func (i *InventoryItem) GetLabel() string  { return i.Label }
func (i *InventoryItem) IsSelectable() bool { return false }
func (i *InventoryItem) GetHelpText() string {
	return "Run-wide inventory persists across decks"
}

// InventoryMenuHandler shows run-wide and deck-local carried items.
type InventoryMenuHandler struct {
	items []MenuItem
}

// NewInventoryMenuHandler lists keycards and other carried items.
func NewInventoryMenuHandler(g *state.Game) *InventoryMenuHandler {
	h := &InventoryMenuHandler{}
	if g == nil {
		h.items = []MenuItem{&InventoryItem{Label: "SUBTLE{Empty}"}}
		return h
	}

	var names []string
	g.RunInventory.Each(func(item *world.Item) {
		if item != nil && item.Name != "" {
			names = append(names, item.Name)
		}
	})
	g.OwnedItems.Each(func(item *world.Item) {
		if item == nil || item.Name == "" {
			return
		}
		if strings.Contains(strings.ToLower(item.Name), "keycard") {
			return // run inventory lists keycards once
		}
		names = append(names, item.Name)
	})
	if g.Batteries > 0 {
		names = append(names, fmt.Sprintf("Batteries x%d", g.Batteries))
	}
	if g.HasMap {
		names = append(names, "Map")
	}

	if len(names) == 0 {
		h.items = []MenuItem{&InventoryItem{Label: "SUBTLE{Nothing carried}"}}
		return h
	}
	for _, name := range names {
		label := name
		if strings.Contains(strings.ToLower(name), "keycard") {
			label = fmt.Sprintf("KEYCARD{%s}", name)
		} else {
			label = fmt.Sprintf("ITEM{%s}", name)
		}
		h.items = append(h.items, &InventoryItem{Label: label})
	}
	return h
}

func (h *InventoryMenuHandler) GetTitle() string {
	return "Inventory"
}

func (h *InventoryMenuHandler) GetInstructions(selected MenuItem) string {
	return engineinput.HintMenuInstructionsGameplay()
}

func (h *InventoryMenuHandler) OnSelect(item MenuItem, index int) {}
func (h *InventoryMenuHandler) OnActivate(item MenuItem, index int) (bool, string) {
	return false, ""
}
func (h *InventoryMenuHandler) OnExit()                     {}
func (h *InventoryMenuHandler) ShouldCloseOnAnyAction() bool { return false }

// RunInventoryMenu opens the inventory viewer overlay.
func RunInventoryMenu(g *state.Game) {
	if g == nil {
		return
	}
	handler := NewInventoryMenuHandler(g)
	RunMenu(g, handler.items, handler)
}

// FormatRunInventoryLine returns a compact inventory summary for the status bar.
func FormatRunInventoryLine(g *state.Game) string {
	if g == nil {
		return ""
	}
	var parts []string
	g.RunInventory.Each(func(item *world.Item) {
		if item != nil && item.Name != "" {
			parts = append(parts, renderer.StyledKeycard(item.Name))
		}
	})
	if len(parts) == 0 {
		return "SUBTLE{No keycards}"
	}
	return strings.Join(parts, ", ")
}
