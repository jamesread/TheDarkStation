package menu

import (
	"strings"
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func TestRepairInterlockMenuLines_RespectKnowledgeBoundary(t *testing.T) {
	g := state.NewGame()
	grid := world.NewGrid(1, 2)
	grid.MarkAsRoomWithName(0, 0, "Pressure Annex", "")
	grid.MarkAsRoomWithName(0, 1, "Transfer Control", "")
	grid.BuildAllCellConnections()
	g.Grid = grid

	valve := entities.NewRepairObjective("valve", entities.RepairPressureValve, "Pressure Annex", 0, 0)
	coupler := entities.NewRepairObjective("coupler", entities.RepairPowerCoupler, "Transfer Control", 0, 1)
	coupler.PrereqIDs = []string{valve.ID}
	gameworld.GetGameData(grid.GetCell(0, 0)).RepairDevice = valve
	gameworld.GetGameData(grid.GetCell(0, 1)).RepairDevice = coupler
	g.RepairObjectives = []*entities.RepairObjective{valve, coupler}

	if lines := repairInterlockMenuLines(g, coupler.RoomName); len(lines) != 0 {
		t.Fatalf("unknown coupler should not appear in diagnostics: %v", lines)
	}

	coupler.Inspect()
	unlocated := strings.Join(repairInterlockMenuLines(g, coupler.RoomName), "\n")
	if !strings.Contains(unlocated, "PRESSURE CONTROL INPUT OUT OF TOLERANCE") {
		t.Fatalf("known coupler should expose its diagnosis, got %q", unlocated)
	}
	if strings.Contains(unlocated, valve.Name) || strings.Contains(unlocated, valve.RoomName) {
		t.Fatalf("unseen upstream identity leaked in diagnostics: %q", unlocated)
	}

	valve.Inspect()
	located := strings.Join(repairInterlockMenuLines(g, coupler.RoomName), "\n")
	if !strings.Contains(located, valve.Name) || !strings.Contains(located, valve.RoomName) {
		t.Fatalf("known upstream should be identified, got %q", located)
	}
}
