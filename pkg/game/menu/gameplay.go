// Package menu provides gameplay menu implementation using the generic menu system.
package menu

// GameplayMenuAction represents the action type for gameplay menu items.
type GameplayMenuAction int

const (
	GameplayMenuActionBindings GameplayMenuAction = iota
	GameplayMenuActionQuitToTitle
)

// GameplayMenuItem represents a menu item in the gameplay menu.
type GameplayMenuItem struct {
	Label  string
	Action GameplayMenuAction
}

// GetLabel returns the display label for this menu item.
func (m *GameplayMenuItem) GetLabel() string {
	return m.Label
}

// IsSelectable returns whether this item can be selected.
func (m *GameplayMenuItem) IsSelectable() bool {
	return true
}

// GetHelpText returns help text for this menu item.
func (m *GameplayMenuItem) GetHelpText() string {
	switch m.Action {
	case GameplayMenuActionBindings:
		return "Configure keyboard and gamepad bindings"
	case GameplayMenuActionQuitToTitle:
		return "Return to the main menu"
	default:
		return ""
	}
}

// GameplayMenuHandler handles the gameplay menu.
type GameplayMenuHandler struct {
	selectedAction GameplayMenuAction
	shouldQuit     bool
}

// NewGameplayMenuHandler creates a new gameplay menu handler.
func NewGameplayMenuHandler() *GameplayMenuHandler {
	return &GameplayMenuHandler{
		shouldQuit: false,
	}
}

// GetTitle returns the menu title.
func (h *GameplayMenuHandler) GetTitle() string {
	return "Gameplay Menu"
}

// GetInstructions returns the menu instructions.
func (h *GameplayMenuHandler) GetInstructions(selected MenuItem) string {
	return "Use up/down to select, Enter to activate, F10/Start or q to close"
}

// OnSelect is called when an item is selected.
func (h *GameplayMenuHandler) OnSelect(item MenuItem, index int) {
	if gameplayItem, ok := item.(*GameplayMenuItem); ok {
		h.selectedAction = gameplayItem.Action
	}
}

// OnActivate is called when an item is activated.
func (h *GameplayMenuHandler) OnActivate(item MenuItem, index int) (shouldClose bool, helpText string) {
	if gameplayItem, ok := item.(*GameplayMenuItem); ok {
		h.selectedAction = gameplayItem.Action
		if gameplayItem.Action == GameplayMenuActionQuitToTitle {
			h.shouldQuit = true
			return true, ""
		}
		// For Bindings, close menu so bindings menu can run
		return true, ""
	}
	return false, ""
}

// OnExit is called when the menu is exited.
func (h *GameplayMenuHandler) OnExit() {
	// Nothing to do on exit
}

// ShouldCloseOnAnyAction returns true if the menu should close on any action.
func (h *GameplayMenuHandler) ShouldCloseOnAnyAction() bool {
	return false // Gameplay menu only closes on activation or quit
}

// GetSelectedAction returns the selected action (if any).
func (h *GameplayMenuHandler) GetSelectedAction() GameplayMenuAction {
	return h.selectedAction
}

// ShouldQuitToTitle returns true if the user selected Quit to Title.
func (h *GameplayMenuHandler) ShouldQuitToTitle() bool {
	return h.shouldQuit
}

// GetMenuItems returns the menu items for the gameplay menu.
func (h *GameplayMenuHandler) GetMenuItems() []MenuItem {
	return []MenuItem{
		&GameplayMenuItem{Label: "Bindings", Action: GameplayMenuActionBindings},
		&GameplayMenuItem{Label: "Quit to Title", Action: GameplayMenuActionQuitToTitle},
	}
}
