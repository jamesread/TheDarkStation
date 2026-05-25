package gameplay

import (
	"fmt"

	"darkstation/pkg/game/devtools"
	gamemenu "darkstation/pkg/game/menu"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
)

// DevMenuAction identifies developer menu selections.
type DevMenuAction int

const (
	DevMenuActionDumpMap DevMenuAction = iota
	DevMenuActionDevTestMap
	DevMenuActionToggleMapAreaBorder
	DevMenuActionToggleFOVRays
	DevMenuActionToggleFPSDisplay
	DevMenuActionTogglePlayerPosition
)

// DevMenuItem is a selectable row in the developer menu.
type DevMenuItem struct {
	Label  string
	Action DevMenuAction
	G      *state.Game
}

func (d *DevMenuItem) GetLabel() string { return d.Label }

func (d *DevMenuItem) IsSelectable() bool { return true }

func (d *DevMenuItem) GetHelpText() string {
	switch d.Action {
	case DevMenuActionDumpMap:
		return "Write map.txt (same as F8)"
	case DevMenuActionDevTestMap:
		return "Load the 50×50 developer testing map"
	case DevMenuActionToggleMapAreaBorder:
		return "Toggle red border around the map viewport"
	case DevMenuActionToggleFOVRays:
		return "Toggle FOV ray-cast debug lines from the player"
	case DevMenuActionToggleFPSDisplay:
		return "Toggle draw.fps cvar (FPS counter in top-right corner)"
	case DevMenuActionTogglePlayerPosition:
		return "Toggle draw.player_pos cvar (player X/Y below FPS counter)"
	default:
		return ""
	}
}

// DevMenuHandler handles the in-game developer menu (F9).
type DevMenuHandler struct {
	g *state.Game
}

// NewDevMenuHandler creates a developer menu handler for the current game.
func NewDevMenuHandler(g *state.Game) *DevMenuHandler {
	return &DevMenuHandler{g: g}
}

func (h *DevMenuHandler) GetTitle() string {
	return "Developer Menu"
}

func (h *DevMenuHandler) GetInstructions(selected gamemenu.MenuItem) string {
	return "Up/Down: select | Enter: activate | q or Esc: close"
}

func (h *DevMenuHandler) OnSelect(item gamemenu.MenuItem, index int) {}

func (h *DevMenuHandler) OnActivate(item gamemenu.MenuItem, index int) (bool, string) {
	if _, isClose := item.(*gamemenu.CloseMenuItem); isClose {
		return true, ""
	}
	devItem, ok := item.(*DevMenuItem)
	if !ok || h.g == nil {
		return false, ""
	}
	switch devItem.Action {
	case DevMenuActionDumpMap:
		path, err := devtools.DumpRevealedMapToFile(h.g)
		if err != nil {
			return false, "Map dump failed: " + err.Error()
		}
		msg := renderer.FormatText("Map dumped to ITEM{%s}", path)
		renderer.ShowDeveloperMessage(msg)
		return false, msg
	case DevMenuActionDevTestMap:
		devtools.SwitchToDevMap(h.g)
		return true, "Switched to developer testing map"
	case DevMenuActionToggleMapAreaBorder:
		on := renderer.ToggleDrawMapAreaBorder()
		if on {
			return false, "Map area border: ON"
		}
		return false, "Map area border: OFF"
	case DevMenuActionToggleFOVRays:
		on := renderer.ToggleDrawFOVRays()
		if on {
			return false, "FOV ray lines: ON"
		}
		return false, "FOV ray lines: OFF"
	case DevMenuActionToggleFPSDisplay:
		on := renderer.ToggleShowFPSCounter()
		if on {
			return false, "FPS display: ON"
		}
		return false, "FPS display: OFF"
	case DevMenuActionTogglePlayerPosition:
		on := renderer.ToggleShowPlayerPosition()
		if on {
			return false, "Player position: ON"
		}
		return false, "Player position: OFF"
	default:
		return false, ""
	}
}

func (h *DevMenuHandler) OnExit() {}

func (h *DevMenuHandler) ShouldCloseOnAnyAction() bool {
	return false
}

func mapAreaBorderMenuLabel() string {
	if renderer.DrawMapAreaBorderEnabled() {
		return "Map area border: ON"
	}
	return "Map area border: OFF"
}

func fovRaysMenuLabel() string {
	if renderer.DrawFOVRaysEnabled() {
		return "FOV ray lines: ON"
	}
	return "FOV ray lines: OFF"
}

func fpsDisplayMenuLabel() string {
	if renderer.ShowFPSCounterEnabled() {
		return "FPS display: ON"
	}
	return "FPS display: OFF"
}

func playerPositionMenuLabel() string {
	if renderer.ShowPlayerPositionEnabled() {
		return "Player position: ON"
	}
	return "Player position: OFF"
}

func zoomMenuLabel() string {
	tileSize := renderer.GetTileSize()
	rows, cols := renderer.GetViewportSize()
	return fmt.Sprintf("Zoom: %dpx (%d×%d tiles)", tileSize, cols, rows)
}

func (h *DevMenuHandler) GetMenuItems() []gamemenu.MenuItem {
	return []gamemenu.MenuItem{
		&gamemenu.InfoMenuItem{Label: zoomMenuLabel()},
		&gamemenu.InfoMenuItem{Label: ""},
		&DevMenuItem{Label: "Dump map", Action: DevMenuActionDumpMap, G: h.g},
		&DevMenuItem{Label: "Developer test map", Action: DevMenuActionDevTestMap, G: h.g},
		&DevMenuItem{Label: mapAreaBorderMenuLabel(), Action: DevMenuActionToggleMapAreaBorder, G: h.g},
		&DevMenuItem{Label: fovRaysMenuLabel(), Action: DevMenuActionToggleFOVRays, G: h.g},
		&DevMenuItem{Label: fpsDisplayMenuLabel(), Action: DevMenuActionToggleFPSDisplay, G: h.g},
		&DevMenuItem{Label: playerPositionMenuLabel(), Action: DevMenuActionTogglePlayerPosition, G: h.g},
		&gamemenu.CloseMenuItem{Label: "Close"},
	}
}

// RunDeveloperMenu opens the developer menu until the player closes it.
func RunDeveloperMenu(g *state.Game) {
	if g == nil {
		return
	}
	handler := NewDevMenuHandler(g)
	gamemenu.RunMenuDynamic(g, handler)
}
