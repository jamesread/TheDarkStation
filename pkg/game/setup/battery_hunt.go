package setup

import (
	"fmt"

	"github.com/zyedidia/generic/mapset"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/generator"
	"darkstation/pkg/game/levelrand"
	"darkstation/pkg/game/renderer"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

// SetupBatteryHuntLevel configures a stripped deck with one generator and scattered batteries.
func SetupBatteryHuntLevel(g *state.Game) *SetupConfig {
	avoid := mapset.New[*world.Cell]()
	if g.Grid != nil {
		g.Grid.ForEachCell(func(row, col int, cell *world.Cell) {
			if cell != nil && generator.IsEmptyOverlayRoom(cell.Name) {
				avoid.Put(cell)
			}
		})
	}
	if entry := PlayerEntryCell(g); entry != nil {
		avoid.Put(entry)
	}

	lockedDoorCells := mapset.New[*world.Cell]()
	InitRoomPower(g)

	required := g.Mode().LevelGen.BatteryHuntRequiredRoll(levelrand.Intn)
	placeBatteryHuntGenerator(g, &avoid, required)
	scatterBatteryHuntLoot(g, &avoid, required)

	if exit := g.Grid.ExitCell(); exit != nil {
		exit.Locked = false
	}

	g.CurrentCell = g.Grid.GetCenterCell()

	EnsureInitProgressReachability(g)
	EnsureFloorLootReachability(g)
	EnsureLiftShaftEntryClearance(g)

	return &SetupConfig{
		Avoid:           avoid,
		LockedDoorCells: lockedDoorCells,
	}
}

func placeBatteryHuntGenerator(g *state.Game, avoid *mapset.Set[*world.Cell], batteriesRequired int) {
	if g == nil || g.Grid == nil || batteriesRequired < 1 {
		return
	}
	cell := liftShaftGeneratorCell(g, avoid)
	if cell == nil {
		cell = liftShaftBootstrapCell(g, avoid, nil)
	}
	if cell == nil {
		cell = findGeneratorCellAnywhere(g, avoid, true)
	}
	if cell == nil {
		return
	}

	gen := entities.NewGenerator("Generator #1", batteriesRequired)
	gameworld.GetGameData(cell).Generator = gen
	g.AddGenerator(gen)
	avoid.Put(cell)

	g.AddHint(fmt.Sprintf("Power the %s in %s — find %d batteries on this deck",
		renderer.StyledItem("Generator"), renderer.StyledCell(cell.Name), batteriesRequired))
}

func scatterBatteryHuntLoot(g *state.Game, avoid *mapset.Set[*world.Cell], count int) {
	if g == nil || g.Grid == nil || count < 1 {
		return
	}
	entry := PlayerEntryCell(g)
	for i := 0; i < count; i++ {
		battery := world.NewItem("Battery")
		if placeItem(g, entry, battery, avoid) == nil {
			return
		}
	}
}
