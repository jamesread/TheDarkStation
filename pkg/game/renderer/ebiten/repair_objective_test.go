package ebiten

import (
	"strings"
	"testing"

	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
)

func TestRepairObjectiveLine_FollowsLearnedCausalState(t *testing.T) {
	g := state.NewGame()
	valve := entities.NewRepairObjective("valve", entities.RepairPressureValve, "Pressure", 0, 0)
	coupler := entities.NewRepairObjective("coupler", entities.RepairPowerCoupler, "Transfer", 0, 1)
	coupler.PrereqIDs = []string{valve.ID}
	coupler.Inspect()
	g.RepairObjectives = []*entities.RepairObjective{valve, coupler}

	if got := repairObjectiveLine(g, 2); !strings.Contains(got, "PRESSURE CONTROL INPUT OUT OF TOLERANCE") {
		t.Fatalf("blocked objective = %q", got)
	}

	valve.Inspect()
	if got := repairObjectiveLine(g, 2); !strings.Contains(got, "release the coupler interlock") {
		t.Fatalf("known upstream objective should state predicted effect, got %q", got)
	}

	valve.Complete()
	if got := repairObjectiveLine(g, 1); !strings.Contains(got, "COUPLER INTERLOCK RELEASED") {
		t.Fatalf("ready dependent objective = %q", got)
	}
}
