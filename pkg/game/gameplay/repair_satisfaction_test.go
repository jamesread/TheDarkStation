package gameplay

import (
	"strings"
	"testing"
	"time"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
	gameworld "darkstation/pkg/game/world"
)

func repairSatisfactionTestGame() (*state.Game, *entities.RepairObjective, *entities.RepairObjective, *world.Cell, *world.Cell) {
	g := state.NewGame()
	grid := world.NewGrid(1, 3)
	grid.MarkAsRoomWithName(0, 0, "Entry", "")
	grid.MarkAsRoomWithName(0, 1, "Pressure Annex", "")
	grid.MarkAsRoomWithName(0, 2, "Transfer Control", "")
	grid.BuildAllCellConnections()
	grid.SetStartCellAt(0, 0)
	g.Grid = grid
	g.CurrentCell = grid.GetCell(0, 0)

	valveCell := grid.GetCell(0, 1)
	couplerCell := grid.GetCell(0, 2)
	valve := entities.NewRepairObjective("valve", entities.RepairPressureValve, valveCell.Name, valveCell.Row, valveCell.Col)
	coupler := entities.NewRepairObjective("coupler", entities.RepairPowerCoupler, couplerCell.Name, couplerCell.Row, couplerCell.Col)
	coupler.PrereqIDs = []string{valve.ID}
	gameworld.GetGameData(valveCell).RepairDevice = valve
	gameworld.GetGameData(couplerCell).RepairDevice = coupler
	g.RepairObjectives = []*entities.RepairObjective{valve, coupler}
	return g, valve, coupler, valveCell, couplerCell
}

func TestRepairInvestigation_KnowledgeGatesUpstreamIdentity(t *testing.T) {
	g, valve, coupler, _, couplerCell := repairSatisfactionTestGame()

	if !CheckAdjacentRepairAtCell(g, couplerCell) {
		t.Fatal("locked coupler inspection should consume the interaction")
	}
	if !coupler.Inspected {
		t.Fatal("coupler should record local inspection")
	}
	unlocated := repairPrereqCallout(g, coupler)
	if strings.Contains(unlocated, valve.Name) || strings.Contains(unlocated, valve.RoomName) {
		t.Fatalf("unseen upstream identity leaked in %q", unlocated)
	}
	if !strings.Contains(unlocated, "PRESSURE CONTROL INPUT OUT OF TOLERANCE") {
		t.Fatalf("callout should provide causal evidence, got %q", unlocated)
	}

	valve.Inspect()
	located := repairPrereqCallout(g, coupler)
	if !strings.Contains(located, valve.Name) || !strings.Contains(located, valve.RoomName) {
		t.Fatalf("known upstream should be identified, got %q", located)
	}
}

func TestRepairCompletion_AcknowledgesNewlyActionableCoupler(t *testing.T) {
	g, valve, coupler, valveCell, _ := repairSatisfactionTestGame()
	coupler.Inspect()
	valve.Inspect()

	if callout := repairDeviceCallout(g, valve, valveCell); !strings.Contains(callout, "release the coupler interlock") {
		t.Fatalf("valve should state the predicted downstream effect, got %q", callout)
	}
	completeRepair(g, valve, valveCell)

	if !g.RepairPrereqsComplete(coupler) {
		t.Fatal("coupler should become actionable after valve completion")
	}
	found := false
	for _, msg := range g.Messages {
		if strings.Contains(msg.Text, "COUPLER INTERLOCK RELEASED") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("completion should report the known dependent's state transition; messages=%+v", g.Messages)
	}
}

func TestBeginLongUse_RecordsRepairInspection(t *testing.T) {
	g, valve, _, valveCell, _ := repairSatisfactionTestGame()
	if !beginLongUse(g, LongUseRepair, valveCell) {
		t.Fatal("expected repair long-use session to start")
	}
	if !valve.Inspected {
		t.Fatal("starting a physical repair should persist its local inspection")
	}
}

func TestTimedRepairCompletion_AcknowledgesDependent(t *testing.T) {
	g, _, dependent, pumpCell, blockerCell := repairSatisfactionTestGame()
	pump := entities.NewRepairObjective("pump", entities.RepairWastePump, pumpCell.Name, pumpCell.Row, pumpCell.Col)
	pump.BlockerName = "Toxic Slime"
	pump.BlockerCells = []entities.BlockerCell{{Row: blockerCell.Row, Col: blockerCell.Col}}
	dependent.PrereqIDs = []string{pump.ID}
	dependent.Inspect()
	gameworld.GetGameData(pumpCell).RepairDevice = pump
	gameworld.GetGameData(blockerCell).RepairBlocker = pump
	g.RepairObjectives = []*entities.RepairObjective{pump, dependent}

	now := time.Now().UnixMilli()
	pump.BeginTimedCompletion(now)
	if !g.AdvanceRepairTimers(now + entities.WastePumpDrainDurationMs + 1) {
		t.Fatal("expected waste-pump timer to advance")
	}
	OnRepairTimersAdvanced(g)
	if !pump.Acknowledged {
		t.Fatal("timed completion should receive systemic acknowledgement")
	}
	found := false
	for _, msg := range g.Messages {
		if strings.Contains(msg.Text, "DEVICE INTERLOCK RELEASED") {
			found = true
		}
	}
	if !found {
		t.Fatalf("timed completion should report the newly actionable dependent; messages=%+v", g.Messages)
	}
}
