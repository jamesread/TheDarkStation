package menu

import (
	"strings"
	"testing"

	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
)

func TestRoutingCouplerLockColored_tracksProgress(t *testing.T) {
	if got := routingCouplerLockColored(0, 0.85, "0%"); got != "UNPOWERED{0%}" {
		t.Fatalf("zero lock = %q, want UNPOWERED{0%%}", got)
	}
	if got := routingCouplerLockColored(0.2, 0.85, "weak"); got != "UNPOWERED{weak}" {
		t.Fatalf("low lock = %q, want UNPOWERED{weak}", got)
	}
	if got := routingCouplerLockColored(0.5, 0.85, "mid"); got != "DOOR{mid}" {
		t.Fatalf("mid lock = %q, want DOOR{mid}", got)
	}
	if got := routingCouplerLockColored(0.9, 0.85, "ready"); got != "POWERED{ready}" {
		t.Fatalf("ready lock = %q, want POWERED{ready}", got)
	}
}

func TestRoutingCouplerLockLabel_updatesWithAlignment(t *testing.T) {
	handler := &RoutingCouplerMenuHandler{
		targets: []int{50},
		values:  []int{10},
		params:  entities.RoutingCouplerDifficulty(5),
	}
	lockItem := &RoutingCouplerLockItem{Handler: handler}
	before := lockItem.GetLabel()
	handler.values[0] = handler.targets[0]
	after := lockItem.GetLabel()
	if before == after {
		t.Fatal("signal lock label should change when alignment improves")
	}
	if !strings.Contains(before, "\t") || !strings.Contains(after, "\t") {
		t.Fatalf("lock labels should use tab columns, before=%q after=%q", before, after)
	}
	if !strings.Contains(before, "UNPOWERED{") && !strings.Contains(before, "DOOR{") {
		t.Fatalf("misaligned label should be red or warming, got %q", before)
	}
	if !strings.Contains(after, "POWERED{") {
		t.Fatalf("aligned label should be green, got %q", after)
	}
}

func TestRoutingCouplerTuningMeter_centersWhenAligned(t *testing.T) {
	got := routingCouplerTuningMeter(50, 50, 30, 11)
	if !strings.Contains(got, "●") {
		t.Fatalf("meter missing marker: %q", got)
	}
	// Aligned: marker replaces the center null.
	mid := len([]rune(got)) / 2
	runes := []rune(got)
	if runes[mid] != '●' {
		t.Fatalf("aligned meter center = %q, want ● in %q", string(runes[mid]), got)
	}
}

func TestRoutingCouplerTuningMeter_offsetsWhenDetuned(t *testing.T) {
	left := routingCouplerTuningMeter(50, 20, 30, 11)
	right := routingCouplerTuningMeter(50, 80, 30, 11)
	if left == right {
		t.Fatalf("left/right detune should differ: %q vs %q", left, right)
	}
	leftPos := strings.IndexRune(left, '●')
	rightPos := strings.IndexRune(right, '●')
	center := strings.IndexRune(routingCouplerTuningMeter(50, 50, 30, 11), '●')
	if leftPos >= center || rightPos <= center {
		t.Fatalf("marker sides wrong: left=%d center=%d right=%d", leftPos, center, rightPos)
	}
}

func TestRoutingCouplerAxisItem_usesTabColumns(t *testing.T) {
	g := &state.Game{LevelSeed: 11}
	repair := entities.NewRepairObjective("routing-repair-deck3-0", entities.RepairSignalCalibrator, "Lab", 0, 0)
	repair.SkipExitGate = true
	repair.TargetDeckID = 2
	handler := NewRoutingCouplerMenuHandler(g, nil, repair, nil)
	axis := &RoutingCouplerAxisItem{Handler: handler, Index: 0, Name: "Phase"}
	label := axis.GetLabel()
	if strings.Count(label, "\t") < 2 {
		t.Fatalf("axis label should have tab columns, got %q", label)
	}
	if !strings.Contains(label, "●") {
		t.Fatalf("axis label should include tuning marker, got %q", label)
	}
	if !axis.HandlesOwnCycleSound() {
		t.Fatal("axis should handle its own cycle sound")
	}
}

func TestRoutingCouplerMenuHandler_commitGatedBySignalLock(t *testing.T) {
	g := &state.Game{LevelSeed: 7}
	repair := entities.NewRepairObjective("routing-repair-deck5-0", entities.RepairSignalCalibrator, "Lab", 1, 2)
	repair.SkipExitGate = true
	repair.TargetDeckID = 4
	repair.Name = "Lift Routing Coupler (Deck 5)"

	completed := false
	handler := NewRoutingCouplerMenuHandler(g, &world.Cell{}, repair, func() {
		completed = true
	})

	items := handler.GetMenuItems()
	var commit *RoutingCouplerCommitItem
	for _, item := range items {
		if c, ok := item.(*RoutingCouplerCommitItem); ok {
			commit = c
			break
		}
	}
	if commit == nil {
		t.Fatal("missing commit item")
	}

	if commit.IsSelectable() {
		t.Fatal("commit should start disabled when misaligned")
	}
	for i := range handler.values {
		handler.values[i] = handler.targets[i]
	}
	if !commit.IsSelectable() {
		t.Fatal("commit should be selectable when aligned")
	}
	closeMenu, _ := handler.OnActivate(commit, 0)
	if !closeMenu || !completed {
		t.Fatalf("activate closeMenu=%v completed=%v", closeMenu, completed)
	}
}

func TestRoutingCouplerAxisItem_HandleCycle_updatesLock(t *testing.T) {
	g := &state.Game{LevelSeed: 11}
	repair := entities.NewRepairObjective("routing-repair-deck3-0", entities.RepairSignalCalibrator, "Lab", 0, 0)
	repair.SkipExitGate = true
	repair.TargetDeckID = 2
	handler := NewRoutingCouplerMenuHandler(g, nil, repair, nil)

	before := handler.signalLock()
	axis := &RoutingCouplerAxisItem{Handler: handler, Index: 0, Name: "Phase"}
	consumed, _ := axis.HandleCycle(1)
	if !consumed {
		t.Fatal("cycle should be consumed")
	}
	after := handler.signalLock()
	if before == after && handler.values[0] == handler.targets[0] {
		t.Fatal("expected value or lock to change after cycle")
	}
}

func TestRoutingCouplerDifficulty_menuAxesMatchDeck(t *testing.T) {
	g := &state.Game{LevelSeed: 3}
	repair := entities.NewRepairObjective("routing-repair-deck10-fallback", entities.RepairSignalCalibrator, "Hub", 0, 0)
	repair.SkipExitGate = true
	repair.TargetDeckID = 9
	handler := NewRoutingCouplerMenuHandler(g, nil, repair, nil)

	axisCount := 0
	for _, item := range handler.GetMenuItems() {
		if _, ok := item.(*RoutingCouplerAxisItem); ok {
			axisCount++
		}
	}
	want := entities.RoutingCouplerDifficulty(10).Axes
	if axisCount != want {
		t.Fatalf("menu axis rows = %d, want %d for deck 10", axisCount, want)
	}
}
