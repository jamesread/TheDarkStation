package ebiten

import (
	"math"
	"testing"
)

func TestTitleScreenMenuTransition(t *testing.T) {
	if !titleScreenMenuTransition("The Dark Station", "Settings") {
		t.Error("main -> settings should transition")
	}
	if !titleScreenMenuTransition("Settings", "The Dark Station") {
		t.Error("settings -> main should transition")
	}
	if titleScreenMenuTransition("The Dark Station", "The Dark Station") {
		t.Error("same title should not transition")
	}
	if titleScreenMenuTransition("Maintenance Terminal", "Select room") {
		t.Error("in-game menus should not use title-screen transition")
	}
}

func TestMenuTransitionPhases_contentWaitsForPanel(t *testing.T) {
	start := int64(1000)
	midPanel := start + titleScreenMenuPanelResizeMs/2
	endPanel := start + titleScreenMenuPanelResizeMs
	midFade := endPanel + titleScreenMenuContentFadeMs/2

	if menuTransitionContentProgress(midPanel, start) != 0 {
		t.Error("incoming text should stay hidden while panel resizes")
	}
	if p := menuTransitionPanelProgress(endPanel, start); p != 1 {
		t.Errorf("panel progress at end of resize = %v, want 1", p)
	}
	if menuTransitionContentProgress(endPanel, start) != 0 {
		t.Error("incoming text should not start until panel resize completes")
	}
	if menuTransitionContentProgress(midFade, start) <= 0 {
		t.Error("incoming text should be fading in after panel resize")
	}
}

func TestMenuTransitionSlides_staggered(t *testing.T) {
	if got := menuTransitionOutgoingSlide(true, 1); got != -float64(titleScreenMenuSlidePx) {
		t.Fatalf("outgoing end = %v", got)
	}
	if got := menuTransitionIncomingSlide(true, 0); got != float64(titleScreenMenuSlidePx) {
		t.Fatalf("incoming start = %v", got)
	}
	if got := menuTransitionIncomingSlide(true, 1); got != 0 {
		t.Fatalf("incoming end = %v", got)
	}
}

func TestMenuPanelWidthForTitle_mainMenuNarrower(t *testing.T) {
	mainW := menuPanelWidthForTitle("The Dark Station", 1000, false, false)
	settingsW := menuPanelWidthForTitle("Settings", 1000, false, false)
	if mainW >= settingsW {
		t.Errorf("main menu width %d should be less than settings %d", mainW, settingsW)
	}
}

func TestMenuPanelWidthForTitle_mainMenuMinWidth(t *testing.T) {
	if got := menuPanelWidthForTitle("The Dark Station", 400, false, false); got != mainMenuPanelMinWidth {
		t.Fatalf("main menu width on narrow screen = %d, want %d", got, mainMenuPanelMinWidth)
	}
}

func TestSettingsMenuUsesFixedTop(t *testing.T) {
	if !settingsMenuUsesFixedTop("Settings") {
		t.Error("Settings title should use fixed top")
	}
	if settingsMenuUsesFixedTop("The Dark Station") {
		t.Error("main menu should stay vertically centered")
	}
}

func TestSettingsMenuHeightAnimProgress_waitsForDuration(t *testing.T) {
	start := int64(1000)
	mid := start + settingsMenuHeightAnimDurationMs()/2
	end := start + settingsMenuHeightAnimDurationMs()
	if settingsMenuHeightAnimProgress(mid, start) <= 0 {
		t.Error("mid animation should be in progress")
	}
	if settingsMenuHeightAnimProgress(end, start) != 1 {
		t.Error("animation should complete at duration")
	}
}

func TestBeginSettingsMenuHeightAnim_shrinkAfterExpand(t *testing.T) {
	e := &EbitenRenderer{}
	e.genericMenuTitle = "Settings"
	e.settingsMenuHeightBaseline = 400
	e.menuHeightAnimating = true
	e.menuHeightAnimStartHeight = 300
	e.menuHeightAnimTargetHeight = 400
	now := int64(2000)
	e.menuHeightAnimStartTime = now - settingsMenuHeightAnimDurationMs()

	e.finishMenuHeightAnimationIfDone(now)
	if e.settingsMenuHeightBaseline != 400 {
		t.Fatalf("baseline = %v, want 400", e.settingsMenuHeightBaseline)
	}

	e.beginSettingsMenuHeightAnim(now, 200)
	if !e.menuHeightAnimating {
		t.Fatal("expected shrink animation to start")
	}
	if e.menuHeightAnimStartHeight != 400 {
		t.Fatalf("shrink start = %v, want 400", e.menuHeightAnimStartHeight)
	}
	if e.menuHeightAnimTargetHeight != 200 {
		t.Fatalf("shrink target = %v, want 200", e.menuHeightAnimTargetHeight)
	}
}

func TestBeginSettingsMenuHeightAnim_retargetMidAnimation(t *testing.T) {
	e := &EbitenRenderer{}
	e.genericMenuTitle = "Settings"
	e.settingsMenuHeightBaseline = 300
	e.menuHeightAnimating = true
	e.menuHeightAnimStartHeight = 300
	e.menuHeightAnimTargetHeight = 400
	start := int64(1000)
	e.menuHeightAnimStartTime = start - settingsMenuHeightAnimDurationMs()/2
	now := start

	e.beginSettingsMenuHeightAnim(now, 200)
	if !e.menuHeightAnimating {
		t.Fatal("expected retargeted animation")
	}
	wantStart := lerpFloat(300, 400, settingsMenuHeightAnimProgress(now, start-settingsMenuHeightAnimDurationMs()/2))
	if math.Abs(e.menuHeightAnimStartHeight-wantStart) > 1 {
		t.Fatalf("retarget start = %v, want ~%v", e.menuHeightAnimStartHeight, wantStart)
	}
	if e.menuHeightAnimTargetHeight != 200 {
		t.Fatalf("retarget target = %v, want 200", e.menuHeightAnimTargetHeight)
	}
}

func TestLerpInt(t *testing.T) {
	if got := lerpInt(100, 200, 0.5); got != 150 {
		t.Errorf("lerpInt = %d, want 150", got)
	}
}
