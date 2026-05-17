// Instrument strata regression tests (Story 5.4).
// Strata render only while the maintenance menu is open; unpowered terminals never open that menu —
// see gameplay.TestCheckAdjacentMaintenanceTerminalAtCell_UnpoweredBlocksMenu.
package menu

import (
	"strings"
	"testing"
)

func TestMergeCorrelatesLimited_prioritizesXCOREAndJNCTBeforeENVREF(t *testing.T) {
	xcore := []string{"XCORE-\tb\tsecond", "XCORE-\ta\tfirst"}
	jnct := []string{"JNCT-\tz\tz"}
	env := []string{
		"ENVREF-\tENV_PLAQUE_A\ta",
		"ENVREF-\tENV_PLAQUE_B\tb",
		"ENVREF-\tENV_PLAQUE_C\tc",
		"ENVREF-\tENV_PLAQUE_D\td",
		"ENVREF-\tENV_PLAQUE_E\te",
	}
	got := mergeCorrelatesLimited(xcore, jnct, env, 6)
	if len(got) != 6 {
		t.Fatalf("len=%d want 6", len(got))
	}
	if got[0] != "XCORE-\ta\tfirst" || got[1] != "XCORE-\tb\tsecond" {
		t.Fatalf("XCORE tier wrong order/presence: %#v", got)
	}
	if got[2] != "JNCT-\tz\tz" {
		t.Fatalf("JNCT missing or misplaced: %#v", got)
	}
	if !strings.HasPrefix(got[3], "ENVREF-\t") {
		t.Fatalf("expected ENVREF after tiers: %#v", got)
	}
	for _, line := range got {
		if strings.Contains(line, "ENV_PLAQUE_E") {
			t.Fatalf("sixth slot should not squeeze out puzzle tier; got overflow env line: %#v", got)
		}
	}
}

func TestMaintenanceInstrument_traceRows_haveTwoTabsForMaintenanceColumnLayout(t *testing.T) {
	g, _ := makeMenuTestGame(t)
	g.Level = 3
	g.CurrentDeckID = 2
	g.LevelSeed = 42

	for _, line := range maintenanceInstrumentMenuLines(g, "RoomA") {
		if line == "" || strings.HasPrefix(line, "SUBTLE{") {
			continue
		}
		if strings.Count(line, "\t") < 2 {
			t.Fatalf("maintenance renderer expects ≥2 tabs per instrument row for column layout; got %q", line)
		}
	}
}
