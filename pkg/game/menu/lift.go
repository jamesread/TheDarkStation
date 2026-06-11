package menu

import (
	"fmt"

	engineinput "darkstation/pkg/engine/input"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/state"
)

// LiftDeckItem is one row in the lift destination menu.
type LiftDeckItem struct {
	DeckID  int
	Level   int
	Blocked string
	G       *state.Game
}

func (d *LiftDeckItem) GetLabel() string {
	if d.Blocked != "" {
		return fmt.Sprintf("Deck %d\tSUBTLE{%s}", d.Level, d.Blocked)
	}
	if d.G != nil && d.G.CurrentDeckID == d.DeckID {
		return fmt.Sprintf("Deck %d\tSUBTLE{current}", d.Level)
	}
	return fmt.Sprintf("Deck %d", d.Level)
}

func (d *LiftDeckItem) IsSelectable() bool {
	return d.Blocked == "" && (d.G == nil || d.G.CurrentDeckID != d.DeckID)
}

func (d *LiftDeckItem) GetHelpText() string {
	if d.Blocked != "" {
		return d.Blocked
	}
	if d.G != nil && d.G.CurrentDeckID == d.DeckID {
		return "You are already on this deck"
	}
	return fmt.Sprintf("Travel to deck %d", d.Level)
}

// LiftMenuHandler handles deck selection at the lift shaft.
type LiftMenuHandler struct {
	g              *state.Game
	items          []MenuItem
	selectedDeckID int
	selectedLevel  int
	travel         bool
}

// NewLiftMenuHandler builds a lift menu for the current game state.
func NewLiftMenuHandler(g *state.Game) *LiftMenuHandler {
	h := &LiftMenuHandler{g: g, selectedDeckID: -1}
	for deckID := 0; deckID < deck.TotalDecks; deckID++ {
		level := deckID + 1
		blocked := ""
		if !g.IsDeckTravelUnlocked(deckID) {
			blocked = g.DeckTravelBlockReason(deckID)
		}
		h.items = append(h.items, &LiftDeckItem{
			DeckID:  deckID,
			Level:   level,
			Blocked: blocked,
			G:       g,
		})
	}
	return h
}

func (h *LiftMenuHandler) GetTitle() string {
	return "Lift Routing"
}

func (h *LiftMenuHandler) GetInstructions(selected MenuItem) string {
	return engineinput.HintMenuInstructionsGameplay()
}

func (h *LiftMenuHandler) OnSelect(item MenuItem, index int) {}

func (h *LiftMenuHandler) OnActivate(item MenuItem, index int) (shouldClose bool, helpText string) {
	deckItem, ok := item.(*LiftDeckItem)
	if !ok || !deckItem.IsSelectable() {
		return false, "Destination unavailable"
	}
	h.selectedDeckID = deckItem.DeckID
	h.selectedLevel = deckItem.Level
	h.travel = true
	return true, ""
}

func (h *LiftMenuHandler) OnExit() {}

func (h *LiftMenuHandler) ShouldCloseOnAnyAction() bool {
	return false
}

func (h *LiftMenuHandler) SelectedDeck() (deckID, level int, travel bool) {
	return h.selectedDeckID, h.selectedLevel, h.travel
}

// RunLiftMenu opens the lift deck selector. Returns target level and true when travel was chosen.
func RunLiftMenu(g *state.Game) (targetLevel int, ok bool) {
	if g == nil {
		return 0, false
	}
	handler := NewLiftMenuHandler(g)
	RunMenu(g, handler.items, handler)
	if !handler.travel {
		return 0, false
	}
	return handler.selectedLevel, true
}
