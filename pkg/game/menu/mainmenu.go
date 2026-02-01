// Package menu provides main menu implementation using the generic menu system.
package menu

import (
	"os"

	"darkstation/pkg/game/state"
)

// MainMenuAction represents the action type for main menu items.
type MainMenuAction int

const (
	MainMenuActionGenerate MainMenuAction = iota
	MainMenuActionDebug
	MainMenuActionBindings
	MainMenuActionQuit
)

// MainMenuItem represents a menu item in the main menu.
type MainMenuItem struct {
	Label  string
	Action MainMenuAction
}

// GetLabel returns the display label for this menu item.
func (m *MainMenuItem) GetLabel() string {
	return m.Label
}

// IsSelectable returns whether this item can be selected.
func (m *MainMenuItem) IsSelectable() bool {
	return true
}

// GetHelpText returns help text for this menu item.
func (m *MainMenuItem) GetHelpText() string {
	switch m.Action {
	case MainMenuActionGenerate:
		return "Start a new game on Deck 1"
	case MainMenuActionDebug:
		return "Open the developer testing map"
	case MainMenuActionBindings:
		return "Configure keyboard and gamepad bindings"
	case MainMenuActionQuit:
		return "Exit the game"
	default:
		return ""
	}
}

// MainMenuHandler handles the main menu.
type MainMenuHandler struct {
	selectedAction MainMenuAction
	shouldQuit     bool
}

// NewMainMenuHandler creates a new main menu handler.
func NewMainMenuHandler() *MainMenuHandler {
	return &MainMenuHandler{
		shouldQuit: false,
	}
}

// GetTitle returns the menu title.
func (h *MainMenuHandler) GetTitle() string {
	return "The Dark Station"
}

// GetInstructions returns the menu instructions.
func (h *MainMenuHandler) GetInstructions(selected MenuItem) string {
	return "Use up/down to select, Enter to activate, q to quit"
}

// OnSelect is called when an item is selected.
func (h *MainMenuHandler) OnSelect(item MenuItem, index int) {
	if mainItem, ok := item.(*MainMenuItem); ok {
		h.selectedAction = mainItem.Action
	}
}

// OnActivate is called when an item is activated.
func (h *MainMenuHandler) OnActivate(item MenuItem, index int) (shouldClose bool, helpText string) {
	if mainItem, ok := item.(*MainMenuItem); ok {
		h.selectedAction = mainItem.Action
		if mainItem.Action == MainMenuActionQuit {
			h.shouldQuit = true
			return true, ""
		}
		// For Bindings, close menu so bindings menu can run, then caller will loop back
		// For Generate and Debug, close menu and let caller handle the action
		return true, ""
	}
	return false, ""
}

// OnExit is called when the menu is exited.
func (h *MainMenuHandler) OnExit() {
	// Nothing to do on exit
}

// ShouldCloseOnAnyAction returns true if the menu should close on any action.
func (h *MainMenuHandler) ShouldCloseOnAnyAction() bool {
	return false // Main menu only closes on activation or quit
}

// GetSelectedAction returns the selected action (if any).
func (h *MainMenuHandler) GetSelectedAction() MainMenuAction {
	return h.selectedAction
}

// ShouldQuit returns true if the user selected Quit.
func (h *MainMenuHandler) ShouldQuit() bool {
	return h.shouldQuit
}

// GetMenuItems returns the menu items for the main menu.
func (h *MainMenuHandler) GetMenuItems() []MenuItem {
	return []MenuItem{
		&MainMenuItem{Label: "Generate", Action: MainMenuActionGenerate},
		&MainMenuItem{Label: "Debug", Action: MainMenuActionDebug},
		&MainMenuItem{Label: "Bindings", Action: MainMenuActionBindings},
		&MainMenuItem{Label: "Quit", Action: MainMenuActionQuit},
	}
}

// RunMainMenu runs the main menu and returns the selected action or quits.
// Returns the selected action, or MainMenuActionQuit if user quit.
func RunMainMenu() MainMenuAction {
	// Create a minimal game state for the menu (needed for rendering)
	g := state.NewGame()

	handler := NewMainMenuHandler()
	items := handler.GetMenuItems()
	RunMenu(g, items, handler)

	if handler.ShouldQuit() {
		os.Exit(0)
	}

	return handler.GetSelectedAction()
}
