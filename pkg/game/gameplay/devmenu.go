package gameplay

import (
	"fmt"
	"strconv"
	"strings"

	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/devtools"
	"darkstation/pkg/game/levelseed"
	gamemenu "darkstation/pkg/game/menu"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/setup"
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
	DevMenuActionTriggerOverload
	DevMenuActionLoadSeed
	DevMenuActionJumpToDeck
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
	case DevMenuActionTriggerOverload:
		return "Force power overload in the current room (trips generators, shorts other loads)"
	case DevMenuActionLoadSeed:
		return "Regenerate the current deck from a hexadecimal seed (for map reproduction)"
	case DevMenuActionJumpToDeck:
		return fmt.Sprintf("Jump to any deck (1–%d); loads saved state or generates if not yet visited", deck.TotalDecks)
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
	case DevMenuActionTriggerOverload:
		if h.g.CurrentCell == nil || h.g.CurrentCell.Name == "" || h.g.CurrentCell.Name == "Corridor" {
			return false, "Stand in a named room to trigger overload"
		}
		room := h.g.CurrentCell.Name
		if setup.TriggerPowerOverloadForDev(h.g, room) {
			UpdateLightingExploration(h.g)
			msg := fmt.Sprintf("Power overload triggered (protected %s)", room)
			renderer.ShowDeveloperMessage(msg)
			return true, msg
		}
		UpdateLightingExploration(h.g)
		return false, "No overload applied (consumption already within supply)"
	case DevMenuActionLoadSeed:
		initial := ""
		if h.g.LevelSeed != 0 {
			initial = levelseed.Format(h.g.LevelSeed)
		}
		seedText, ok := gamemenu.RunTextInputDialog(h.g, gamemenu.TextInputOptions{
			Title:   "Load level seed",
			Prompt:  fmt.Sprintf("Enter hex seed for deck %d", h.g.Level),
			Initial: initial,
			Hex:     true,
		})
		if !ok {
			return false, "Seed entry cancelled"
		}
		seed, err := levelseed.Parse(seedText)
		if err != nil {
			return false, err.Error()
		}
		LoadLevelFromSeed(h.g, seed)
		msg := fmt.Sprintf("Loaded seed %s on deck %d", levelseed.Format(seed), h.g.Level)
		renderer.ShowDeveloperMessage(msg)
		return true, msg
	case DevMenuActionJumpToDeck:
		deckText, ok := gamemenu.RunTextInputDialog(h.g, gamemenu.TextInputOptions{
			Title:   "Jump to deck",
			Prompt:  fmt.Sprintf("Enter deck number (1–%d)", deck.TotalDecks),
			Initial: fmt.Sprintf("%d", h.g.Level),
		})
		if !ok {
			return false, "Deck entry cancelled"
		}
		target, err := strconv.Atoi(strings.TrimSpace(deckText))
		if err != nil {
			return false, "Invalid deck number"
		}
		if target == h.g.Level {
			return false, fmt.Sprintf("Already on deck %d", target)
		}
		if err := JumpToDeck(h.g, target); err != nil {
			return false, err.Error()
		}
		msg := fmt.Sprintf("Jumped to deck %d", h.g.Level)
		renderer.ShowDeveloperMessage(msg)
		return true, msg
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

func levelSeedMenuLabel(g *state.Game) string {
	if g != nil && g.LevelSeed != 0 {
		return fmt.Sprintf("Load level seed (%s)", levelseed.Format(g.LevelSeed))
	}
	return "Load level seed"
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
		&DevMenuItem{Label: levelSeedMenuLabel(h.g), Action: DevMenuActionLoadSeed, G: h.g},
		&DevMenuItem{Label: "Jump to deck", Action: DevMenuActionJumpToDeck, G: h.g},
		&DevMenuItem{Label: "Trigger overload", Action: DevMenuActionTriggerOverload, G: h.g},
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
