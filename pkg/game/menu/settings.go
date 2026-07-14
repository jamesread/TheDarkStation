package menu

import (
	"fmt"

	engineinput "darkstation/pkg/engine/input"
)

// SettingsTab identifies a tab in the unified settings menu.
type SettingsTab int

const (
	SettingsTabBindings SettingsTab = iota
	SettingsTabVideo
)

// SettingsMenuHandler is the title-screen and in-game settings menu (bindings + video tabs).
type SettingsMenuHandler struct {
	tab          SettingsTab
	fromMainMenu bool
	bindings     *BindingsMenuHandler
}

// NewSettingsMenuHandler creates a settings menu. When fromMainMenu is true, Back returns to the title screen.
func NewSettingsMenuHandler(fromMainMenu bool) *SettingsMenuHandler {
	return &SettingsMenuHandler{
		tab:          SettingsTabBindings,
		fromMainMenu: fromMainMenu,
		bindings:     NewBindingsMenuHandler(false),
	}
}

// NewSettingsMenuHandlerWithTab opens settings on a specific tab.
func NewSettingsMenuHandlerWithTab(fromMainMenu bool, tab SettingsTab) *SettingsMenuHandler {
	h := NewSettingsMenuHandler(fromMainMenu)
	h.tab = tab
	return h
}

func (h *SettingsMenuHandler) GetTitle() string {
	return "Settings"
}

func (h *SettingsMenuHandler) GetInstructions(selected MenuItem) string {
	exitHint := engineinput.HintBindingsExit(h.fromMainMenu)
	if _, ok := selected.(*SettingsTabItem); ok {
		return fmt.Sprintf("Down to enter menu, left/right to switch tab. %s", exitHint)
	}
	if _, ok := selected.(*BindingMenuItem); ok {
		return fmt.Sprintf("%s, %s, %s.", engineinput.HintMenuSelect(), engineinput.HintMenuEditBinding(), exitHint)
	}
	if _, ok := selected.(*WindowModeMenuItem); ok {
		return engineinput.HintMenuSelect() + ", " + engineinput.HintMenuActivate() + ", left/right to cycle, " + exitHint + "."
	}
	if _, ok := selected.(*BackMenuItem); ok {
		return fmt.Sprintf("%s, %s.", engineinput.HintMenuBackToMain(), exitHint)
	}
	if _, ok := selected.(*CloseMenuItem); ok {
		return fmt.Sprintf("%s, %s.", engineinput.HintMenuSelect(), exitHint)
	}
	return fmt.Sprintf("%s, %s.", engineinput.HintMenuSelect(), exitHint)
}

func (h *SettingsMenuHandler) OnSelect(item MenuItem, index int) {
	if tab, ok := item.(*SettingsTabItem); ok {
		h.setTab(tab.tab)
	}
}

func (h *SettingsMenuHandler) OnActivate(item MenuItem, index int) (shouldClose bool, helpText string) {
	if _, ok := item.(*BackMenuItem); ok {
		return true, ""
	}
	if _, ok := item.(*CloseMenuItem); ok {
		return true, ""
	}
	if tab, ok := item.(*SettingsTabItem); ok {
		h.setTab(tab.tab)
		return false, ""
	}
	if cycler, ok := item.(CycleMenuItem); ok && cycler.CanCycle() {
		_, helpText := cycler.HandleCycle(1)
		return false, helpText
	}
	return h.bindings.OnActivate(item, index)
}

func (h *SettingsMenuHandler) OnExit() {}

func (h *SettingsMenuHandler) ShouldCloseOnAnyAction() bool {
	return false
}

func (h *SettingsMenuHandler) setTab(tab SettingsTab) {
	if h.tab == tab {
		return
	}
	h.tab = tab
}

// InitialMenuSelection implements InitialSelectionProvider.
func (h *SettingsMenuHandler) InitialMenuSelection(items []MenuItem) int {
	for i, item := range items {
		if tab, ok := item.(*SettingsTabItem); ok && tab.tab == h.tab {
			return i
		}
	}
	return 0
}

// TryHorizontalTabNav implements HorizontalTabNavigator for the settings tab strip.
func (h *SettingsMenuHandler) TryHorizontalTabNav(items []MenuItem, selected int, intent engineinput.Intent) (newSelected int, consumed bool, helpText string) {
	var delta int
	switch intent.Action {
	case engineinput.ActionMoveWest:
		delta = -1
	case engineinput.ActionMoveEast:
		delta = 1
	default:
		return selected, false, ""
	}
	stripLen := SettingsTabStripLength(items)
	if stripLen < 2 || selected < 0 || selected >= stripLen {
		return selected, false, ""
	}
	next := selected + delta
	if next < 0 || next >= stripLen {
		return selected, false, ""
	}
	tabItem, ok := items[next].(*SettingsTabItem)
	if !ok {
		return selected, false, ""
	}
	h.setTab(tabItem.tab)
	return next, true, ""
}

func (h *SettingsMenuHandler) activeTabIndex(items []MenuItem) int {
	stripLen := SettingsTabStripLength(items)
	for i := 0; i < stripLen; i++ {
		if tab, ok := items[i].(*SettingsTabItem); ok && tab.tab == h.tab {
			return i
		}
	}
	return 0
}

// TryVerticalTabNav implements TabStripNavigator.
func (h *SettingsMenuHandler) TryVerticalTabNav(items []MenuItem, selected int, intent engineinput.Intent) (newSelected int, consumed bool) {
	stripLen := SettingsTabStripLength(items)
	if stripLen < 2 {
		return selected, false
	}
	switch intent.Action {
	case engineinput.ActionMoveSouth:
		if selected < 0 || selected >= stripLen {
			return selected, false
		}
		for i := stripLen; i < len(items); i++ {
			if items[i].IsSelectable() {
				return i, true
			}
		}
		return selected, true
	case engineinput.ActionMoveNorth:
		if selected >= stripLen {
			return h.activeTabIndex(items), true
		}
		if selected >= 0 && selected < stripLen {
			for i := len(items) - 1; i >= 0; i-- {
				if items[i].IsSelectable() {
					return i, true
				}
			}
			return selected, true
		}
	}
	return selected, false
}

func (h *SettingsMenuHandler) GetMenuItems() []MenuItem {
	items := []MenuItem{
		&SettingsTabItem{owner: h, tab: SettingsTabBindings},
		&SettingsTabItem{owner: h, tab: SettingsTabVideo},
	}
	switch h.tab {
	case SettingsTabBindings:
		items = append(items, h.bindings.CoreMenuItems()...)
	case SettingsTabVideo:
		items = append(items, &WindowModeMenuItem{})
	}
	if h.fromMainMenu {
		items = append(items, &BackMenuItem{})
	} else {
		items = append(items, &CloseMenuItem{Label: "Back"})
	}
	return items
}

// SettingsTabStripLength returns how many leading items are settings tabs (0 or 2).
func SettingsTabStripLength(items []MenuItem) int {
	n := 0
	for _, item := range items {
		if _, ok := item.(*SettingsTabItem); ok {
			n++
			continue
		}
		break
	}
	if n < 2 {
		return 0
	}
	return n
}

// SettingsTabItem is one focusable tab in the settings menu tab strip.
type SettingsTabItem struct {
	owner *SettingsMenuHandler
	tab   SettingsTab
}

func (s *SettingsTabItem) GetLabel() string {
	switch s.tab {
	case SettingsTabBindings:
		return "Bindings"
	default:
		return "Video"
	}
}

func (s *SettingsTabItem) IsSelectable() bool { return true }

func (s *SettingsTabItem) GetHelpText() string {
	switch s.tab {
	case SettingsTabBindings:
		return "Keyboard and controller bindings"
	default:
		return "Display and window options"
	}
}

// IsActiveContentTab reports whether this tab's panel is currently shown.
func (s *SettingsTabItem) IsActiveContentTab() bool {
	return s.owner != nil && s.owner.tab == s.tab
}
