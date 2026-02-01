// Package menu provides bindings menu implementation using the generic menu system.
package menu

import (
	"fmt"
	"strings"

	engineinput "darkstation/pkg/engine/input"
	"darkstation/pkg/game/renderer"
)

// BindingMenuItem represents a menu item for a key binding.
type BindingMenuItem struct {
	Action        engineinput.Action
	NonRebindable bool
}

// GetLabel returns the display label for this binding menu item.
func (b *BindingMenuItem) GetLabel() string {
	name := engineinput.ActionName(b.Action)
	byAction := engineinput.GetBindingsByAction()
	codes := byAction[b.Action]
	codeText := strings.Join(codes, ", ")
	if codeText == "" {
		codeText = "(unbound)"
	}

	if b.NonRebindable {
		return fmt.Sprintf("%s: %s (fixed)", renderer.StyledSubtle(name), codeText)
	}
	return fmt.Sprintf("%s: %s", name, codeText)
}

// IsSelectable returns whether this binding can be selected.
func (b *BindingMenuItem) IsSelectable() bool {
	return true
}

// GetHelpText returns help text for this binding.
func (b *BindingMenuItem) GetHelpText() string {
	if b.NonRebindable {
		return ""
	}
	return fmt.Sprintf("Editing binding for: %s", engineinput.ActionName(b.Action))
}

// BackMenuItem represents a "Back" menu item for returning to the previous menu.
type BackMenuItem struct{}

// GetLabel returns the display label for the back menu item.
func (b *BackMenuItem) GetLabel() string {
	return "Back"
}

// IsSelectable returns whether this item can be selected.
func (b *BackMenuItem) IsSelectable() bool {
	return true
}

// GetHelpText returns help text for the back menu item.
func (b *BackMenuItem) GetHelpText() string {
	return "Return to the previous menu"
}

// BindingsMenuHandler handles the bindings menu.
type BindingsMenuHandler struct {
	actions       []engineinput.Action
	nonRebindable map[engineinput.Action]bool
	fromMainMenu  bool
}

// NewBindingsMenuHandler creates a new bindings menu handler.
// If fromMainMenu is true, a "Back" option will be added to return to the main menu.
func NewBindingsMenuHandler(fromMainMenu bool) *BindingsMenuHandler {
	actions := []engineinput.Action{
		engineinput.ActionMoveNorth,
		engineinput.ActionMoveSouth,
		engineinput.ActionMoveWest,
		engineinput.ActionMoveEast,
		engineinput.ActionHint,
		engineinput.ActionInteract,
		engineinput.ActionZoomIn,
		engineinput.ActionZoomOut,
	}

	nonRebindable := make(map[engineinput.Action]bool)
	for _, act := range actions {
		if isNonRebindable(act) {
			nonRebindable[act] = true
		}
	}

	return &BindingsMenuHandler{
		actions:       actions,
		nonRebindable: nonRebindable,
		fromMainMenu:  fromMainMenu,
	}
}

// GetTitle returns the menu title.
func (h *BindingsMenuHandler) GetTitle() string {
	return "Bindings Menu"
}

// GetInstructions returns the menu instructions.
func (h *BindingsMenuHandler) GetInstructions(selected MenuItem) string {
	exitHint := "F10/Start or q to exit"
	if h.fromMainMenu {
		exitHint = "Enter to go back, F10/Start or q to exit"
	}

	if selected == nil {
		return fmt.Sprintf("Use up/down to select, %s.", exitHint)
	}

	// Check if it's the Back menu item
	if _, ok := selected.(*BackMenuItem); ok {
		return fmt.Sprintf("Press Enter to return to the main menu, %s.", exitHint)
	}

	bindingItem, ok := selected.(*BindingMenuItem)
	if !ok {
		return fmt.Sprintf("Use up/down to select, %s.", exitHint)
	}

	if !bindingItem.NonRebindable {
		return fmt.Sprintf("Use up/down to select, Enter to edit, %s.", exitHint)
	}
	return fmt.Sprintf("Use up/down to select, %s.", exitHint)
}

// OnSelect is called when an item is selected.
func (h *BindingsMenuHandler) OnSelect(item MenuItem, index int) {
	// Nothing to do on selection
}

// OnActivate is called when an item is activated.
func (h *BindingsMenuHandler) OnActivate(item MenuItem, index int) (shouldClose bool, helpText string) {
	// Check if it's the Back menu item
	if _, ok := item.(*BackMenuItem); ok {
		// Close menu to return to previous menu
		return true, ""
	}

	bindingItem, ok := item.(*BindingMenuItem)
	if !ok {
		return false, ""
	}

	action := bindingItem.Action

	// Check if action is non-rebindable - don't allow editing
	if bindingItem.NonRebindable {
		return false, ""
	}

	actionName := engineinput.ActionName(action)

	// Use renderer.GetInput() to read a raw-ish code string
	code := renderer.GetInput()
	if code != "" {
		engineinput.SetSingleBinding(action, code)
		// Show confirmation message
		helpText = fmt.Sprintf("Set binding for %s to %s", actionName, code)
	} else {
		// User cancelled or entered empty string - clear help text
		helpText = ""
	}

	return false, helpText
}

// OnExit is called when the menu is exited.
func (h *BindingsMenuHandler) OnExit() {
	// Nothing to do on exit
}

// ShouldCloseOnAnyAction returns true if the menu should close on any action.
func (h *BindingsMenuHandler) ShouldCloseOnAnyAction() bool {
	return false // Bindings menu only closes on menu/quit actions
}

// GetMenuItems returns the menu items for the bindings menu.
func (h *BindingsMenuHandler) GetMenuItems() []MenuItem {
	items := make([]MenuItem, len(h.actions))
	for i, action := range h.actions {
		items[i] = &BindingMenuItem{
			Action:        action,
			NonRebindable: h.nonRebindable[action],
		}
	}

	// Add "Back" option at the end if opened from main menu
	if h.fromMainMenu {
		items = append(items, &BackMenuItem{})
	}

	return items
}

// isNonRebindable checks if an action cannot be rebound.
func isNonRebindable(action engineinput.Action) bool {
	return action == engineinput.ActionInteract ||
		action == engineinput.ActionZoomIn ||
		action == engineinput.ActionZoomOut
}
