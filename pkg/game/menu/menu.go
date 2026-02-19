// Package menu provides a generic menu system for the game.
package menu

import (
	"fmt"

	engineinput "darkstation/pkg/engine/input"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
)

// MenuItem represents a single item in a menu.
type MenuItem interface {
	// GetLabel returns the display label for this menu item.
	GetLabel() string
	// IsSelectable returns whether this item can be selected.
	IsSelectable() bool
	// GetHelpText returns optional help text for this item.
	GetHelpText() string
}

// MenuHandler handles menu item selection and activation.
type MenuHandler interface {
	// OnSelect is called when an item is selected (navigated to).
	OnSelect(item MenuItem, index int)

	// OnActivate is called when an item is activated (e.g., Enter pressed).
	// Returns true if the menu should close, and any help text to display.
	OnActivate(item MenuItem, index int) (shouldClose bool, helpText string)
	// OnExit is called when the menu is exited.
	OnExit()
	// GetTitle returns the menu title.
	GetTitle() string
	// GetInstructions returns the menu instructions.
	GetInstructions(selected MenuItem) string
	// ShouldCloseOnAnyAction returns true if the menu should close on any action (not just menu/quit).
	ShouldCloseOnAnyAction() bool
}

// DynamicMenuHandler extends MenuHandler with dynamic menu items.
// RunMenuDynamic calls GetMenuItems each loop iteration so the menu can refresh.
type DynamicMenuHandler interface {
	MenuHandler
	GetMenuItems() []MenuItem
}

// MaintenanceRoomProvider is an optional interface for handlers that display a maintenance view
// for a specific room. When implemented, the room name is set on game state so the renderer can
// highlight that room's wall cells on the map. selectedIndex and items are the current menu
// selection (for sub-menus like room selector that highlight the focused room).
type MaintenanceRoomProvider interface {
	GetMaintenanceRoom(selectedIndex int, items []MenuItem) string
}

// MenuRenderer is an optional interface for renderers that can draw
// a full-screen menu overlay on top of the map.
type MenuRenderer interface {
	// RenderMenu draws the menu overlay with the given items, selected index, help text, and title.
	RenderMenu(g *state.Game, items []MenuItem, selected int, helpText string, title string)
	// ClearMenu hides any active menu overlay.
	ClearMenu()
}

// RunMenu runs a generic menu with the given items and handler.
func RunMenu(g *state.Game, items []MenuItem, handler MenuHandler) {
	selected := 0
	helpText := ""

	// Find first selectable item
	for i, item := range items {
		if item.IsSelectable() {
			selected = i
			break
		}
	}

	for {
		// Set maintenance room for renderer to highlight walls (when handler provides it)
		if provider, ok := handler.(MaintenanceRoomProvider); ok {
			g.MaintenanceMenuRoom = provider.GetMaintenanceRoom(selected, items)
		}

		// Use renderer-native, full-screen overlay (Ebiten).
		if mr, ok := renderer.Current.(MenuRenderer); ok {
			mr.RenderMenu(g, items, selected, helpText, handler.GetTitle())
		} else {
			// Fallback: render menu into the message log (should not happen with Ebiten).
			renderMenuFallback(g, items, selected, helpText, handler)
		}

		// Get next intent
		intent := renderer.Current.GetInput()

		// Check if handler wants to close on any action (except navigation)
		if handler.ShouldCloseOnAnyAction() && intent.Action != engineinput.ActionNone &&
			intent.Action != engineinput.ActionMoveNorth && intent.Action != engineinput.ActionMoveSouth {
			g.ClearMessages()
			if mr, ok := renderer.Current.(MenuRenderer); ok {
				mr.ClearMenu()
			}
			handler.OnExit()
			return
		}

		switch intent.Action {
		case engineinput.ActionMoveNorth:
			// Move selection up to previous selectable item (with wrap-around)
			found := false
			for i := selected - 1; i >= 0; i-- {
				if items[i].IsSelectable() {
					selected = i
					helpText = "" // Clear help text when navigating
					handler.OnSelect(items[selected], selected)
					found = true
					break
				}
			}
			// If no item found above, wrap to the last selectable item
			if !found {
				for i := len(items) - 1; i > selected; i-- {
					if items[i].IsSelectable() {
						selected = i
						helpText = "" // Clear help text when navigating
						handler.OnSelect(items[selected], selected)
						break
					}
				}
			}
		case engineinput.ActionMoveSouth:
			// Move selection down to next selectable item (with wrap-around)
			found := false
			for i := selected + 1; i < len(items); i++ {
				if items[i].IsSelectable() {
					selected = i
					helpText = "" // Clear help text when navigating
					handler.OnSelect(items[selected], selected)
					found = true
					break
				}
			}
			// If no item found below, wrap to the first selectable item
			if !found {
				for i := 0; i < selected; i++ {
					if items[i].IsSelectable() {
						selected = i
						helpText = "" // Clear help text when navigating
						handler.OnSelect(items[selected], selected)
						break
					}
				}
			}
		case engineinput.ActionAction, engineinput.ActionInteract:
			// Activate selected item
			if selected >= 0 && selected < len(items) && items[selected].IsSelectable() {
				shouldClose, newHelpText := handler.OnActivate(items[selected], selected)
				helpText = newHelpText
				if shouldClose {
					g.ClearMessages()
					if mr, ok := renderer.Current.(MenuRenderer); ok {
						mr.ClearMenu()
					}
					handler.OnExit()
					return
				}
			}
		case engineinput.ActionOpenMenu, engineinput.ActionQuit:
			// Exit menu
			g.ClearMessages()
			if mr, ok := renderer.Current.(MenuRenderer); ok {
				mr.ClearMenu()
			}
			handler.OnExit()
			return
		case engineinput.ActionNone:
			// Ignore
		default:
			// Ignore other actions while in menu
		}
	}
}

// RunMenuDynamic runs a menu whose items can change. The handler's GetMenuItems
// is called each loop iteration so the menu content can refresh (e.g. after room selection).
func RunMenuDynamic(g *state.Game, handler DynamicMenuHandler) {
	selected := 0
	helpText := ""

	// Clear maintenance room highlight when menu exits (all return paths)
	defer func() { g.MaintenanceMenuRoom = "" }()

	for {
		items := handler.GetMenuItems()

		// Find first selectable item, or keep current if still valid
		if selected >= len(items) || !items[selected].IsSelectable() {
			selected = 0
			for i, item := range items {
				if item.IsSelectable() {
					selected = i
					break
				}
			}
		}

		// Set maintenance room for renderer to highlight walls (when handler provides it)
		if provider, ok := handler.(MaintenanceRoomProvider); ok {
			g.MaintenanceMenuRoom = provider.GetMaintenanceRoom(selected, items)
		} else {
			g.MaintenanceMenuRoom = ""
		}

		// Use renderer-native, full-screen overlay (Ebiten).
		if mr, ok := renderer.Current.(MenuRenderer); ok {
			mr.RenderMenu(g, items, selected, helpText, handler.GetTitle())
		} else {
			renderMenuFallback(g, items, selected, helpText, handler)
		}

		// Get next intent
		intent := renderer.Current.GetInput()

		// Check if handler wants to close on any action (except navigation)
		if handler.ShouldCloseOnAnyAction() && intent.Action != engineinput.ActionNone &&
			intent.Action != engineinput.ActionMoveNorth && intent.Action != engineinput.ActionMoveSouth {
			g.ClearMessages()
			if mr, ok := renderer.Current.(MenuRenderer); ok {
				mr.ClearMenu()
			}
			handler.OnExit()
			return
		}

		switch intent.Action {
		case engineinput.ActionMoveNorth:
			found := false
			for i := selected - 1; i >= 0; i-- {
				if items[i].IsSelectable() {
					selected = i
					helpText = ""
					handler.OnSelect(items[selected], selected)
					found = true
					break
				}
			}
			if !found {
				for i := len(items) - 1; i > selected; i-- {
					if items[i].IsSelectable() {
						selected = i
						helpText = ""
						handler.OnSelect(items[selected], selected)
						break
					}
				}
			}
		case engineinput.ActionMoveSouth:
			found := false
			for i := selected + 1; i < len(items); i++ {
				if items[i].IsSelectable() {
					selected = i
					helpText = ""
					handler.OnSelect(items[selected], selected)
					found = true
					break
				}
			}
			if !found {
				for i := 0; i < selected; i++ {
					if items[i].IsSelectable() {
						selected = i
						helpText = ""
						handler.OnSelect(items[selected], selected)
						break
					}
				}
			}
		case engineinput.ActionAction, engineinput.ActionInteract:
			if selected >= 0 && selected < len(items) && items[selected].IsSelectable() {
				shouldClose, newHelpText := handler.OnActivate(items[selected], selected)
				helpText = newHelpText
				if shouldClose {
					g.ClearMessages()
					if mr, ok := renderer.Current.(MenuRenderer); ok {
						mr.ClearMenu()
					}
					handler.OnExit()
					return
				}
			}
		case engineinput.ActionOpenMenu, engineinput.ActionQuit:
			g.ClearMessages()
			if mr, ok := renderer.Current.(MenuRenderer); ok {
				mr.ClearMenu()
			}
			handler.OnExit()
			return
		case engineinput.ActionNone:
			// Ignore
		default:
			// Ignore other actions while in menu
		}
	}
}

// renderMenuFallback renders the menu in the message log as a fallback.
// This should not be used with Ebiten renderer, which supports MenuRenderer.
func renderMenuFallback(g *state.Game, items []MenuItem, selected int, helpText string, handler MenuHandler) {
	g.ClearMessages()
	logMessage(g, "=== %s ===", handler.GetTitle())

	// Show version information
	versionText := fmt.Sprintf("Version: %s", renderer.Version)
	if renderer.Commit != "unknown" && len(renderer.Commit) > 0 {
		versionText += fmt.Sprintf(" (%s)", renderer.Commit[:7])
	}
	logMessage(g, versionText)

	// Show instructions
	var selectedItem MenuItem
	if selected >= 0 && selected < len(items) {
		selectedItem = items[selected]
	}
	instructions := handler.GetInstructions(selectedItem)
	if instructions != "" {
		logMessage(g, instructions)
	}

	// Show help text if provided
	if helpText != "" {
		logMessage(g, helpText)
	}

	// Show menu items
	for i, item := range items {
		prefix := "  "
		if i == selected {
			prefix = "> "
		}
		label := item.GetLabel()
		if !item.IsSelectable() {
			// Style non-selectable items differently
			label = renderer.StyledSubtle(label)
		}
		logMessage(g, "%s%s", prefix, label)
	}

	// Re-render frame with updated messages
	renderer.RenderFrame(g)
}

// Helper function to match logMessage signature
func logMessage(g *state.Game, msg string, a ...any) {
	formatted := renderer.ApplyMarkup(msg, a...)
	g.AddMessage(formatted)
}
