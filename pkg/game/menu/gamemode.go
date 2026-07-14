package menu

import (
	"fmt"

	engineinput "darkstation/pkg/engine/input"
	"darkstation/pkg/game/gamemode"
	"darkstation/pkg/game/state"
)

// GameModeMenuItem is one selectable game mode on the new-game screen.
type GameModeMenuItem struct {
	Mode gamemode.Mode
}

func (m *GameModeMenuItem) GetLabel() string {
	deckLabel := "deck"
	if m.Mode.TotalDecks != 1 {
		deckLabel = "decks"
	}
	return fmt.Sprintf("%s\tSUBTLE{%d %s}", m.Mode.DisplayName, m.Mode.TotalDecks, deckLabel)
}

func (m *GameModeMenuItem) IsSelectable() bool {
	return true
}

func (m *GameModeMenuItem) GetHelpText() string {
	switch m.Mode.ID {
	case gamemode.SinglePlayerPuzzle:
		return "Full station run: restore power, clear hazards, and travel between all decks"
	case gamemode.SingleDeckSandbox:
		return "Single-deck layout for quick sessions and experiments"
	case gamemode.FindTheBatteries:
		return "Explore one large deck, collect every battery, and power the generator"
	default:
		return fmt.Sprintf("Start a new game in %s mode", m.Mode.DisplayName)
	}
}

// GameModeMenuHandler handles game mode selection before a new run.
type GameModeMenuHandler struct {
	items        []MenuItem
	defaultMode  gamemode.ID
	selectedMode gamemode.ID
	confirmed    bool
}

// NewGameModeMenuHandler builds the game mode picker.
func NewGameModeMenuHandler(defaultMode gamemode.ID) *GameModeMenuHandler {
	if defaultMode == "" {
		defaultMode = gamemode.SinglePlayerPuzzle
	}
	h := &GameModeMenuHandler{defaultMode: defaultMode}
	for _, mode := range gamemode.All() {
		h.items = append(h.items, &GameModeMenuItem{Mode: mode})
	}
	return h
}

func (h *GameModeMenuHandler) GetTitle() string {
	return "Select Game Mode"
}

func (h *GameModeMenuHandler) GetInstructions(selected MenuItem) string {
	return engineinput.HintMenuInstructionsMain()
}

func (h *GameModeMenuHandler) OnSelect(item MenuItem, index int) {}

func (h *GameModeMenuHandler) OnActivate(item MenuItem, index int) (shouldClose bool, helpText string) {
	modeItem, ok := item.(*GameModeMenuItem)
	if !ok {
		return false, ""
	}
	h.selectedMode = modeItem.Mode.ID
	h.confirmed = true
	return true, ""
}

func (h *GameModeMenuHandler) InitialMenuSelection(items []MenuItem) int {
	for i, item := range items {
		modeItem, ok := item.(*GameModeMenuItem)
		if ok && modeItem.Mode.ID == h.defaultMode {
			return i
		}
	}
	return 0
}

func (h *GameModeMenuHandler) OnExit() {}

func (h *GameModeMenuHandler) ShouldCloseOnAnyAction() bool {
	return false
}

// RunGameModeMenu opens the game mode picker. Returns the chosen mode and true when confirmed.
func RunGameModeMenu(g *state.Game, defaultMode gamemode.ID) (gamemode.ID, bool) {
	handler := NewGameModeMenuHandler(defaultMode)
	RunMenu(g, handler.items, handler)
	if !handler.confirmed {
		return "", false
	}
	return handler.selectedMode, true
}
