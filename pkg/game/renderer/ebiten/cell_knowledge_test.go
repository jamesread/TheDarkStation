package ebiten

import (
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// knowledgeFixture builds a 1x2 grid with a generator cell at (0,1).
func knowledgeFixture(t *testing.T) (*EbitenRenderer, *state.Game, *world.Cell, *renderSnapshot) {
	t.Helper()
	e := &EbitenRenderer{}
	g := state.NewGame()
	grid := world.NewGrid(1, 2)
	grid.MarkAsRoomWithName(0, 0, "Start", "")
	grid.MarkAsRoomWithName(0, 1, "Engineering", "")
	grid.BuildAllCellConnections()
	g.Grid = grid
	g.CurrentCell = grid.GetCell(0, 0)
	gameworld.InitGameData(g.CurrentCell)
	genCell := grid.GetCell(0, 1)
	gameworld.InitGameData(genCell).Generator = entities.NewGenerator("G", 1)
	snap := &renderSnapshot{playerRow: -1, playerCol: -1}
	return e, g, genCell, snap
}

func TestCellKnowledge_unknownCellIsNotDrawn(t *testing.T) {
	e, g, genCell, snap := knowledgeFixture(t)

	opts := e.getCellRenderOptions(g, genCell, snap, false)
	if opts.Icon != IconVoid {
		t.Fatalf("undiscovered cell without map: icon=%q, want void", opts.Icon)
	}
}

func TestCellKnowledge_mapItemRevealsLayoutOnly(t *testing.T) {
	e, g, genCell, snap := knowledgeFixture(t)
	g.HasMap = true

	opts := e.getCellRenderOptions(g, genCell, snap, false)
	if opts.Icon == IconGeneratorUnpowered || opts.Icon == IconGeneratorPowered {
		t.Fatal("Map item must not reveal equipment, only the floor plan")
	}
	if opts.Color != colorLayout {
		t.Fatalf("map-revealed cell color = %v, want layout %v", opts.Color, colorLayout)
	}
}

func TestCellKnowledge_discoveredDarkCellShowsLayoutOnly(t *testing.T) {
	e, g, genCell, snap := knowledgeFixture(t)
	genCell.Discovered = true

	opts := e.getCellRenderOptions(g, genCell, snap, false)
	if opts.Icon == IconGeneratorUnpowered || opts.Icon == IconGeneratorPowered {
		t.Fatal("dark never-lit cell must not reveal equipment")
	}
	if opts.Color != colorLayout {
		t.Fatalf("dark cell color = %v, want layout %v", opts.Color, colorLayout)
	}
}

func TestCellKnowledge_rememberedCellKeepsIdentityNotState(t *testing.T) {
	e, g, genCell, snap := knowledgeFixture(t)
	genCell.Discovered = true
	gameworld.GetGameData(genCell).Lighted = true // seen lit before, dark now

	opts := e.getCellRenderOptions(g, genCell, snap, false)
	if opts.Icon != IconGeneratorUnpowered {
		t.Fatalf("remembered generator icon = %q, want %q", opts.Icon, IconGeneratorUnpowered)
	}
	if opts.Color != colorRemembered {
		t.Fatalf("remembered cell color = %v, want %v (no live state colors)", opts.Color, colorRemembered)
	}
	if opts.BackgroundColor != colorRememberedBg {
		t.Fatalf("remembered cell bg = %v, want %v", opts.BackgroundColor, colorRememberedBg)
	}
}

func TestCellKnowledge_litCellShowsFullState(t *testing.T) {
	e, g, genCell, snap := knowledgeFixture(t)
	genCell.Discovered = true
	data := gameworld.GetGameData(genCell)
	data.Lighted = true
	data.LightsOn = true

	opts := e.getCellRenderOptions(g, genCell, snap, false)
	if opts.Icon != IconGeneratorUnpowered || opts.BackgroundColor != colorGeneratorFocusBg {
		t.Fatalf("lit generator: icon=%q bg=%v, want full state rendering", opts.Icon, opts.BackgroundColor)
	}
}

func TestCellKnowledge_maintenanceFocusRevealsUndiscoveredRoomWalls(t *testing.T) {
	e := &EbitenRenderer{}
	g := state.NewGame()
	grid := world.NewGrid(3, 3)
	grid.MarkAsRoomWithName(1, 1, "Engineering", "")
	grid.BuildAllCellConnections()
	g.Grid = grid

	wallNorth := grid.GetCell(0, 1)
	snap := &renderSnapshot{
		mapPower: mapPowerSnapshot{maintenanceMenuRoom: "Engineering"},
	}

	opts := e.getCellRenderOptions(g, wallNorth, snap, false)
	if opts.Icon != IconWall {
		t.Fatalf("focused undiscovered room wall: icon=%q, want %q", opts.Icon, IconWall)
	}

	optsNoFocus := e.getCellRenderOptions(g, wallNorth, &renderSnapshot{}, false)
	if optsNoFocus.Icon != IconVoid {
		t.Fatalf("unfocused undiscovered wall: icon=%q, want %q", optsNoFocus.Icon, IconVoid)
	}
}

func TestCellKnowledge_itemsHiddenAtLayoutTier(t *testing.T) {
	e, g, _, snap := knowledgeFixture(t)
	itemCell := g.Grid.GetCell(0, 0)
	itemCell.ItemsOnFloor.Put(world.NewItem("Blue Keycard"))
	itemCell.Discovered = true

	opts := e.getCellRenderOptions(g, itemCell, snap, false)
	if opts.Icon == IconKey {
		t.Fatal("items must not be visible at layout (dark) tier")
	}

	gameworld.GetGameData(itemCell).Lighted = true
	gameworld.GetGameData(itemCell).LightsOn = true
	opts = e.getCellRenderOptions(g, itemCell, snap, false)
	if opts.Icon != IconKey {
		t.Fatalf("lit keycard cell icon = %q, want %q", opts.Icon, IconKey)
	}
}
