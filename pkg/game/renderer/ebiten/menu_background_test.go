package ebiten

import (
	"math/rand"
	"testing"

	"darkstation/pkg/game/state"
)

func TestTitleScreenFloatingTilesMenu(t *testing.T) {
	for _, title := range []string{"The Dark Station", "Settings"} {
		if !titleScreenFloatingTilesMenu(title) {
			t.Errorf("title %q should keep floating tiles", title)
		}
	}
	if titleScreenFloatingTilesMenu("Maintenance Terminal") {
		t.Error("in-game menus should not use title-screen floating tiles")
	}
}

func TestTitleScreenFloatingTilesActive_requiresTitleScreenContext(t *testing.T) {
	e := &EbitenRenderer{}
	e.genericMenuActive = true
	e.genericMenuTitle = "Settings"
	e.snapshot.valid = true
	e.game = nil
	if !e.titleScreenFloatingTilesActive() {
		t.Error("bindings on title screen should animate floating tiles")
	}
	e.game = &state.Game{} //nolint:exhaustruct // minimal stub
	e.snapshot.valid = true
	if e.titleScreenFloatingTilesActive() {
		t.Error("bindings during gameplay should not animate title-screen tiles")
	}
}

func TestFloatingTileTargetCount_scalesWithArea(t *testing.T) {
	small := floatingTileTargetCount(800, 600)
	large := floatingTileTargetCount(1920, 1080)
	if large <= small {
		t.Errorf("larger screen should need more tiles: small=%d large=%d", small, large)
	}
	if got := floatingTileTargetCount(1024, 768); got < 35 || got > 45 {
		t.Errorf("1024×768 target = %d, want ~40", got)
	}
}

func TestSyncFloatingTilesToScreen_growsOnResize(t *testing.T) {
	e := &EbitenRenderer{}
	rand.Seed(1)

	e.floatingTilesMutex.Lock()
	e.syncFloatingTilesToScreenUnlocked(800, 600)
	initial := len(e.floatingTiles)
	e.floatingTilesMutex.Unlock()
	if initial < floatingTileMinCount {
		t.Fatalf("initial count = %d, want at least %d", initial, floatingTileMinCount)
	}

	e.floatingTilesMutex.Lock()
	e.syncFloatingTilesToScreenUnlocked(1600, 1200)
	afterGrow := len(e.floatingTiles)
	w, h := e.floatingTilesScreenW, e.floatingTilesScreenH
	e.floatingTilesMutex.Unlock()

	if afterGrow <= initial {
		t.Errorf("resize grow: count %d -> %d, want increase", initial, afterGrow)
	}
	if w != 1600 || h != 1200 {
		t.Errorf("tracked screen = %dx%d, want 1600x1200", w, h)
	}
}

func TestRandomFloatingTilePos_biasesToExpandedStrip(t *testing.T) {
	rand.Seed(42)
	const oldW, oldH = 800, 600
	const newW, newH = 1200, 600 // widened only
	inStrip := 0
	const samples = 500
	for i := 0; i < samples; i++ {
		x, _ := randomFloatingTilePos(oldW, oldH, newW, newH)
		if x >= float64(oldW) {
			inStrip++
		}
	}
	// Expect strong bias toward the new right strip (not uniform over full width).
	if inStrip < samples*3/4 {
		t.Errorf("only %d/%d spawns in expanded strip, want strong bias", inStrip, samples)
	}
}

func TestSyncFloatingTilesToScreen_trimsOnShrink(t *testing.T) {
	e := &EbitenRenderer{}
	rand.Seed(2)

	e.floatingTilesMutex.Lock()
	e.syncFloatingTilesToScreenUnlocked(1920, 1080)
	largeCount := len(e.floatingTiles)
	e.syncFloatingTilesToScreenUnlocked(800, 600)
	smallCount := len(e.floatingTiles)
	e.floatingTilesMutex.Unlock()

	if smallCount >= largeCount {
		t.Errorf("shrink: count %d -> %d, want decrease", largeCount, smallCount)
	}
}

func TestSyncFloatingTilesToScreen_idempotentAtSameSize(t *testing.T) {
	e := &EbitenRenderer{}
	rand.Seed(3)

	e.floatingTilesMutex.Lock()
	e.syncFloatingTilesToScreenUnlocked(1024, 768)
	first := append([]floatingTile(nil), e.floatingTiles...)
	e.syncFloatingTilesToScreenUnlocked(1024, 768)
	second := e.floatingTiles
	e.floatingTilesMutex.Unlock()

	if len(first) != len(second) {
		t.Fatalf("count changed on repeat sync: %d -> %d", len(first), len(second))
	}
	for i := range first {
		if first[i].x != second[i].x || first[i].y != second[i].y ||
			first[i].vx != second[i].vx || first[i].vy != second[i].vy {
			t.Fatalf("tile %d changed on idempotent sync", i)
		}
	}
}

func TestClampFloatingTileSpeed_limitsVelocity(t *testing.T) {
	tile := floatingTile{vx: 3.5, vy: -2.2}
	clampFloatingTileSpeed(&tile)
	if tile.vx != floatingTileMaxSpeed || tile.vy != -floatingTileMaxSpeed {
		t.Errorf("clamp = (%v,%v), want (±%v)", tile.vx, tile.vy, floatingTileMaxSpeed)
	}
}
