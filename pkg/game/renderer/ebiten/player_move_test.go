package ebiten

import "testing"

func TestPlayerMoveTransition_stepsTowardTarget(t *testing.T) {
	var tAnim playerMoveTransition
	row, col := tAnim.visualPosition(1, 5, 10, 1, 0)
	if row != 5 || col != 10 {
		t.Fatalf("initial = (%v,%v), want (5,10)", row, col)
	}

	tAnim.visualPosition(1, 6, 10, 2, 0)
	row, col = tAnim.visualPosition(1, 6, 10, 2, 70)
	if row <= 5 || row >= 6 {
		t.Fatalf("mid-step row = %v, want between 5 and 6", row)
	}
	if col != 10 {
		t.Fatalf("col = %v, want 10", col)
	}

	row, col = tAnim.visualPosition(1, 6, 10, 2, playerMoveDurationMs+1)
	if row != 6 || col != 10 {
		t.Fatalf("complete = (%v,%v), want (6,10)", row, col)
	}
}

func TestPlayerMoveTransition_chainsBeforeFinish(t *testing.T) {
	var tAnim playerMoveTransition
	tAnim.visualPosition(1, 0, 0, 1, 0)
	tAnim.visualPosition(1, 1, 0, 2, 0)
	tAnim.visualPosition(1, 2, 0, 3, 0)
	row, _ := tAnim.visualPosition(1, 2, 0, 3, 60)
	if row < 1 || row > 2 {
		t.Fatalf("chained row = %v, want between 1 and 2", row)
	}
}

func TestPlayerMoveTransition_snapsOnTeleport(t *testing.T) {
	var tAnim playerMoveTransition
	tAnim.visualPosition(1, 0, 0, 1, 0)
	tAnim.visualPosition(1, 1, 0, 2, 20)
	row, col := tAnim.visualPosition(1, 20, 30, 3, 40)
	if row != 20 || col != 30 {
		t.Fatalf("teleport = (%v,%v), want (20,30)", row, col)
	}
}

func TestPlayerMoveTransition_snapsOnLevelChange(t *testing.T) {
	var tAnim playerMoveTransition
	tAnim.visualPosition(1, 0, 0, 1, 0)
	tAnim.visualPosition(1, 1, 0, 2, 20)
	row, col := tAnim.visualPosition(2, 1, 0, 3, 40)
	if row != 1 || col != 0 {
		t.Fatalf("level change = (%v,%v), want (1,0)", row, col)
	}
}
