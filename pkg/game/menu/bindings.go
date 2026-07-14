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
	keyboard := bindingLabelsForAction(b.Action, false)
	gamepad := bindingLabelsForAction(b.Action, true)
	if b.NonRebindable {
		name += " (fixed)"
	}
	return fmt.Sprintf("%s\t%s\t%s", name, keyboard, gamepad)
}

// IsSelectable returns whether this binding can be selected.
func (b *BindingMenuItem) IsSelectable() bool {
	return true
}

// GetHelpText returns help text for this binding.
func (b *BindingMenuItem) GetHelpText() string {
	if b.NonRebindable {
		return "This binding is fixed"
	}
	return fmt.Sprintf("Editing binding for: %s", engineinput.ActionName(b.Action))
}

// BindingHeaderItem represents a non-selectable section or column header.
type BindingHeaderItem struct {
	Label string
}

// GetLabel returns the display label for this header.
func (b *BindingHeaderItem) GetLabel() string {
	return b.Label
}

// IsSelectable returns whether this item can be selected.
func (b *BindingHeaderItem) IsSelectable() bool {
	return false
}

// GetHelpText returns help text for this header.
func (b *BindingHeaderItem) GetHelpText() string {
	return ""
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
	groups        []bindingGroup
	nonRebindable map[engineinput.Action]bool
	fromMainMenu  bool
}

type bindingGroup struct {
	Name    string
	Actions []engineinput.Action
}

// NewBindingsMenuHandler creates a new bindings menu handler.
// If fromMainMenu is true, a "Back" option will be added to return to the main menu.
func NewBindingsMenuHandler(fromMainMenu bool) *BindingsMenuHandler {
	groups := []bindingGroup{
		{
			Name: "Navigation",
			Actions: []engineinput.Action{
				engineinput.ActionMoveNorth,
				engineinput.ActionMoveSouth,
				engineinput.ActionMoveWest,
				engineinput.ActionMoveEast,
			},
		},
		{
			Name: "Interaction",
			Actions: []engineinput.Action{
				engineinput.ActionInteract,
				engineinput.ActionHint,
			},
		},
		{
			Name: "View",
			Actions: []engineinput.Action{
				engineinput.ActionZoomIn,
				engineinput.ActionZoomOut,
			},
		},
		{
			Name: "System",
			Actions: []engineinput.Action{
				engineinput.ActionOpenMenu,
				engineinput.ActionOpenInventory,
				engineinput.ActionCancel,
				engineinput.ActionQuit,
			},
		},
	}

	nonRebindable := make(map[engineinput.Action]bool)
	for _, group := range groups {
		for _, act := range group.Actions {
			if isNonRebindable(act) {
				nonRebindable[act] = true
			}
		}
	}

	return &BindingsMenuHandler{
		groups:        groups,
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
	exitHint := engineinput.HintBindingsExit(h.fromMainMenu)

	if selected == nil {
		return fmt.Sprintf("%s, %s.", engineinput.HintMenuSelect(), exitHint)
	}

	if _, ok := selected.(*BackMenuItem); ok {
		return fmt.Sprintf("%s, %s.", engineinput.HintMenuBackToMain(), exitHint)
	}

	bindingItem, ok := selected.(*BindingMenuItem)
	if !ok {
		return fmt.Sprintf("%s, %s.", engineinput.HintMenuSelect(), exitHint)
	}
	if !bindingItem.NonRebindable {
		return fmt.Sprintf("%s, %s, %s.", engineinput.HintMenuSelect(), engineinput.HintMenuEditBinding(), exitHint)
	}
	return fmt.Sprintf("%s, %s.", engineinput.HintMenuSelect(), exitHint)
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

	code := renderer.CaptureBindingCode()
	if code != "" {
		engineinput.SetSingleBinding(action, code)
		helpText = fmt.Sprintf("Set binding for %s to %s", actionName, engineinput.FormatBindingCode(code))
	} else {
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

// CoreMenuItems returns binding rows without navigation chrome (used by SettingsMenuHandler).
func (h *BindingsMenuHandler) CoreMenuItems() []MenuItem {
	items := []MenuItem{
		&BindingHeaderItem{Label: "Action\tKeyboard\tController"},
	}
	for _, group := range h.groups {
		items = append(items, &BindingHeaderItem{Label: fmt.Sprintf("TITLE{%s}", group.Name)})
		for _, action := range group.Actions {
			items = append(items, &BindingMenuItem{
				Action:        action,
				NonRebindable: h.nonRebindable[action],
			})
		}
	}
	return items
}

// GetMenuItems returns the menu items for the bindings menu.
func (h *BindingsMenuHandler) GetMenuItems() []MenuItem {
	items := h.CoreMenuItems()
	if h.fromMainMenu {
		items = append(items, &BackMenuItem{})
	}
	return items
}

// isNonRebindable checks if an action cannot be rebound.
func isNonRebindable(action engineinput.Action) bool {
	return action == engineinput.ActionInteract ||
		action == engineinput.ActionZoomIn ||
		action == engineinput.ActionZoomOut ||
		action == engineinput.ActionOpenMenu ||
		action == engineinput.ActionCancel ||
		action == engineinput.ActionQuit
}

func bindingLabelsForAction(action engineinput.Action, gamepad bool) string {
	byAction := engineinput.GetBindingsByAction()
	codes := byAction[action]
	labels := make([]string, 0, len(codes))
	for _, code := range codes {
		if engineinput.IsGamepadCode(code) != gamepad {
			continue
		}
		labels = append(labels, engineinput.FormatBindingCode(code))
	}
	if len(labels) == 0 {
		return "SUBTLE{unbound}"
	}
	return strings.Join(labels, ", ")
}
