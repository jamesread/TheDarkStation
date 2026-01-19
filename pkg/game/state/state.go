package state

import (
	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
)

// NavStyle represents the navigation key style
type NavStyle int

// Navigation styles
const (
	NavStyleNSEW NavStyle = iota
	NavStyleVim
)

// Game represents the game state for Abandoned Station
type Game struct {
	CurrentCell *world.Cell

	Hints []string

	Grid *world.Grid

	HasMap bool

	OwnedItems world.ItemSet

	Messages []string

	NavStyle NavStyle

	Level int // Current level/floor number

	Batteries  int                   // Number of batteries in inventory
	Generators []*entities.Generator // All generators on this level
}

// NewGame creates a new game instance
func NewGame() *Game {
	return &Game{
		OwnedItems: mapset.New[*world.Item](),
		HasMap:     false,
		Messages:   make([]string, 0),
		Level:      1,
		Batteries:  0,
		Generators: make([]*entities.Generator, 0),
	}
}

// AddBatteries adds batteries to the player's inventory
func (g *Game) AddBatteries(count int) {
	g.Batteries += count
}

// UseBatteries removes batteries from inventory, returns actual amount used
func (g *Game) UseBatteries(count int) int {
	if count > g.Batteries {
		count = g.Batteries
	}
	g.Batteries -= count
	return count
}

// AddGenerator registers a generator for this level
func (g *Game) AddGenerator(gen *entities.Generator) {
	g.Generators = append(g.Generators, gen)
}

// AllGeneratorsPowered returns true if all generators are powered
func (g *Game) AllGeneratorsPowered() bool {
	if len(g.Generators) == 0 {
		return true
	}
	for _, gen := range g.Generators {
		if !gen.IsPowered() {
			return false
		}
	}
	return true
}

// UnpoweredGeneratorCount returns the number of unpowered generators
func (g *Game) UnpoweredGeneratorCount() int {
	count := 0
	for _, gen := range g.Generators {
		if !gen.IsPowered() {
			count++
		}
	}
	return count
}

// AddMessage adds a message to the game's message log
func (g *Game) AddMessage(msg string) {
	const maxMessages = 5
	g.Messages = append(g.Messages, msg)

	// Keep only the last maxMessages
	if len(g.Messages) > maxMessages {
		g.Messages = g.Messages[len(g.Messages)-maxMessages:]
	}
}

// ClearMessages clears all messages
func (g *Game) ClearMessages() {
	g.Messages = make([]string, 0)
}

// AddHint adds a hint to the game
func (g *Game) AddHint(hint string) {
	g.Hints = append(g.Hints, hint)
}

// PickUpItem adds an item to the player's inventory
func (g *Game) PickUpItem(item *world.Item) {
	g.OwnedItems.Put(item)
}

// HasItem checks if the player has a specific item
func (g *Game) HasItem(item *world.Item) bool {
	return g.OwnedItems.Has(item)
}

// AdvanceLevel increments the level counter and resets level-specific state
func (g *Game) AdvanceLevel() {
	g.Level++
	g.OwnedItems = mapset.New[*world.Item]()
	g.HasMap = false
	g.Hints = nil
	g.Batteries = 0
	g.Generators = make([]*entities.Generator, 0)
}
