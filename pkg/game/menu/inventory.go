package menu

import (
	"fmt"
	"slices"
	"strings"

	engineinput "darkstation/pkg/engine/input"
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
)

type InventoryItem struct {
	Label string
}

func (i *InventoryItem) GetLabel() string   { return i.Label }
func (i *InventoryItem) IsSelectable() bool { return false }
func (i *InventoryItem) GetHelpText() string {
	return "Run-wide inventory persists across decks"
}

// InventoryMenuHandler shows run-wide and deck-local carried items.
type InventoryMenuHandler struct {
	items []MenuItem
}

func inventoryRowLabel(name string) string {
	dep := renderer.InventoryDepictionForName(name)
	textLabel := fmt.Sprintf("ITEM{%s}", name)
	if state.IsRunWideKeycardName(name) {
		textLabel = fmt.Sprintf("KEYCARD{%s}", name)
	}
	return renderer.FormatInventoryRowLine(dep, textLabel)
}

func inventoryTypeOrder(name string) int {
	if state.IsRunWideKeycardName(name) {
		return 0
	}
	switch renderer.InventoryDepictionForName(name).Key {
	case renderer.InventoryDepictionKeycard:
		return 0
	case renderer.InventoryDepictionKeyMap:
		return 1
	case renderer.InventoryDepictionKeyBattery:
		return 2
	default:
		return 3
	}
}

func sortInventoryNames(names []string) {
	slices.SortFunc(names, func(a, b string) int {
		if oa, ob := inventoryTypeOrder(a), inventoryTypeOrder(b); oa != ob {
			return oa - ob
		}
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})
}

type inventorySectionRow struct {
	typeOrder int
	title     string
	label     string
}

func buildInventorySectionRows(names []string, includeMap bool, batteries int) []inventorySectionRow {
	sortInventoryNames(names)
	var rows []inventorySectionRow
	for _, name := range names {
		rows = append(rows, inventorySectionRow{
			typeOrder: inventoryTypeOrder(name),
			title:     strings.ToLower(name),
			label:     inventoryRowLabel(name),
		})
	}
	if includeMap {
		rows = append(rows, inventorySectionRow{
			typeOrder: inventoryTypeOrder("Map"),
			title:     "map",
			label:     renderer.FormatInventoryRowLine(renderer.InventoryDepictionForMap(), "ITEM{Map}"),
		})
	}
	if batteries > 0 {
		label := fmt.Sprintf("ITEM{Batteries x%d}", batteries)
		rows = append(rows, inventorySectionRow{
			typeOrder: inventoryTypeOrder("Battery"),
			title:     "batteries",
			label:     renderer.FormatInventoryRowLine(renderer.InventoryDepictionForBatteries(), label),
		})
	}
	slices.SortFunc(rows, func(a, b inventorySectionRow) int {
		if a.typeOrder != b.typeOrder {
			return a.typeOrder - b.typeOrder
		}
		return strings.Compare(a.title, b.title)
	})
	return rows
}

func appendInventorySection(h *InventoryMenuHandler, header string, names []string, batteries int, includeMap bool) {
	rows := buildInventorySectionRows(names, includeMap, batteries)
	if len(rows) == 0 {
		return
	}
	h.items = append(h.items, &BindingHeaderItem{Label: fmt.Sprintf("TITLE{%s}", header)})
	for _, row := range rows {
		h.items = append(h.items, &InventoryItem{Label: row.label})
	}
}

// NewInventoryMenuHandler lists keycards and other carried items.
func NewInventoryMenuHandler(g *state.Game) *InventoryMenuHandler {
	h := &InventoryMenuHandler{}
	if g == nil {
		h.items = []MenuItem{&InventoryItem{Label: "SUBTLE{Empty}"}}
		return h
	}

	g.PromoteOwnedRunKeycards()

	var runNames, deckNames []string
	g.RunInventory.Each(func(item *world.Item) {
		if item != nil && item.Name != "" {
			runNames = append(runNames, item.Name)
		}
	})
	g.OwnedItems.Each(func(item *world.Item) {
		if item == nil || item.Name == "" {
			return
		}
		if state.IsRunWideKeycardName(item.Name) {
			return
		}
		if item.Name == "Map" {
			return
		}
		deckNames = append(deckNames, item.Name)
	})

	hasRun := len(runNames) > 0 || g.HasMap
	hasDeck := len(deckNames) > 0 || g.Batteries > 0
	if !hasRun && !hasDeck {
		h.items = []MenuItem{&InventoryItem{Label: "SUBTLE{Nothing carried}"}}
		return h
	}

	appendInventorySection(h, "Run Inventory", runNames, 0, g.HasMap)
	appendInventorySection(h, "Deck Inventory", deckNames, g.Batteries, false)
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
func (h *InventoryMenuHandler) OnExit()                      {}
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
	if g.HasMap {
		parts = append(parts, "Map")
	}
	if len(parts) == 0 {
		return "SUBTLE{No keycards}"
	}
	return strings.Join(parts, ", ")
}
