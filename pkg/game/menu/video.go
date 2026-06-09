package menu

import (
	"darkstation/pkg/engine/input"
	"darkstation/pkg/game/renderer"
)

// VideoMenuHandler handles display settings.
type VideoMenuHandler struct{}

// NewVideoMenuHandler creates a new video settings menu handler.
func NewVideoMenuHandler() *VideoMenuHandler {
	return &VideoMenuHandler{}
}

func (h *VideoMenuHandler) GetTitle() string {
	return "Video"
}

func (h *VideoMenuHandler) GetInstructions(selected MenuItem) string {
	return input.HintMenuSelect() + ", " + input.HintMenuActivate() + ", D-pad left/right or A/D to cycle, " + input.HintMenuClose() + "."
}

func (h *VideoMenuHandler) OnSelect(item MenuItem, index int) {}

func (h *VideoMenuHandler) OnActivate(item MenuItem, index int) (shouldClose bool, helpText string) {
	if _, isClose := item.(*CloseMenuItem); isClose {
		return true, ""
	}
	if cycler, ok := item.(CycleMenuItem); ok && cycler.CanCycle() {
		_, helpText := cycler.HandleCycle(1)
		return false, helpText
	}
	return false, ""
}

func (h *VideoMenuHandler) OnExit() {}

func (h *VideoMenuHandler) ShouldCloseOnAnyAction() bool {
	return false
}

func (h *VideoMenuHandler) GetMenuItems() []MenuItem {
	return []MenuItem{
		&WindowModeMenuItem{},
		&CloseMenuItem{Label: "Back"},
	}
}

// WindowModeMenuItem cycles between windowed and borderless fullscreen.
type WindowModeMenuItem struct{}

func (w *WindowModeMenuItem) GetLabel() string {
	mode := "windowed"
	if renderer.IsFullscreen() {
		mode = "fullscreen"
	}
	return "Window Mode\tACTION{" + mode + "}\tSUBTLE{< left/right >}"
}

func (w *WindowModeMenuItem) IsSelectable() bool {
	return true
}

func (w *WindowModeMenuItem) GetHelpText() string {
	return "Toggle borderless fullscreen without changing screen resolution"
}

func (w *WindowModeMenuItem) CanCycle() bool {
	return true
}

func (w *WindowModeMenuItem) HandleCycle(delta int) (bool, string) {
	on := !renderer.IsFullscreen()
	renderer.SetFullscreen(on)
	if on {
		return true, "Window mode: fullscreen"
	}
	return true, "Window mode: windowed"
}
