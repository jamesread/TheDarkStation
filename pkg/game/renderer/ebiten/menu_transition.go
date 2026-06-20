package ebiten

import (
	"math"
	"time"

	gamemenu "darkstation/pkg/game/menu"
)

const (
	titleScreenMenuPanelResizeMs = 150
	titleScreenMenuContentFadeMs = 110
	titleScreenMenuTransitionMs  = titleScreenMenuPanelResizeMs + titleScreenMenuContentFadeMs
	titleScreenMenuSlidePx       = 32
	// settingsMenuPanelTopY is the fixed distance from the screen top for the Settings menu panel.
	settingsMenuPanelTopY = 220
	// mainMenuPanelMinWidth is the minimum panel width for the title-screen main menu.
	mainMenuPanelMinWidth = 300
)

// menuPanelWidthFrac returns the panel width as a fraction of screen width for a menu title.
func menuPanelWidthFrac(title string, maintOverlay, devMenuRight bool) float32 {
	const defaultPanelWidthFrac = 0.7
	const mainMenuPanelWidthFrac = defaultPanelWidthFrac * 0.25
	const maintPanelWidthFrac = defaultPanelWidthFrac / 2
	switch {
	case title == "The Dark Station":
		return mainMenuPanelWidthFrac
	case maintOverlay:
		return maintPanelWidthFrac
	default:
		return defaultPanelWidthFrac
	}
}

func menuPanelWidthForTitle(title string, screenWidth int, maintOverlay, devMenuRight bool) int {
	w := int(float32(screenWidth) * menuPanelWidthFrac(title, maintOverlay, devMenuRight))
	if title == "The Dark Station" && w < mainMenuPanelMinWidth {
		w = mainMenuPanelMinWidth
	}
	return w
}

func titleScreenMenuTransition(from, to string) bool {
	return titleScreenFloatingTilesMenu(from) && titleScreenFloatingTilesMenu(to) && from != to
}

func titleScreenTransitionForward(from string) bool {
	return from == "The Dark Station"
}

func (e *EbitenRenderer) beginTitleScreenMenuTransition(fromTitle, toTitle string, fromHelp, toHelp string, fromLabels, toLabels []string, toItems []gamemenu.MenuItem, screenWidth, screenHeight int) {
	e.menuPanelTransitionAnimating = true
	e.menuPanelTransitionForward = titleScreenTransitionForward(fromTitle)
	e.menuPanelTransitionStartMs = e.menuAnimClockMilli
	if e.menuPanelTransitionStartMs == 0 {
		e.menuPanelTransitionStartMs = time.Now().UnixMilli()
	}
	e.menuPanelTransitionFromW = menuPanelWidthForTitle(fromTitle, screenWidth, false, false)
	e.menuPanelTransitionToW = menuPanelWidthForTitle(toTitle, screenWidth, false, false)
	e.menuPanelTransitionFromH = e.calculateMenuHeight(fromLabels, fromTitle, fromHelp, e.prevMenuItems)
	e.menuPanelTransitionToH = e.calculateMenuHeight(toLabels, toTitle, toHelp, toItems)
	e.menuHeightAnimating = false
}

func menuTransitionProgress(nowMs, startMs int64) (eased float64, done bool) {
	if startMs == 0 {
		return 1, true
	}
	elapsed := nowMs - startMs
	if elapsed >= titleScreenMenuTransitionMs {
		return 1, true
	}
	t := float64(elapsed) / float64(titleScreenMenuTransitionMs)
	return easeInOut(t), false
}

// menuTransitionPanelProgress eases the panel resize during the first transition phase.
func menuTransitionPanelProgress(nowMs, startMs int64) float64 {
	if startMs == 0 {
		return 1
	}
	elapsed := nowMs - startMs
	if elapsed >= titleScreenMenuPanelResizeMs {
		return 1
	}
	return easeInOut(float64(elapsed) / float64(titleScreenMenuPanelResizeMs))
}

// menuTransitionContentProgress eases incoming text only after the panel has finished resizing.
func menuTransitionContentProgress(nowMs, startMs int64) float64 {
	if startMs == 0 {
		return 1
	}
	elapsed := nowMs - startMs
	if elapsed <= titleScreenMenuPanelResizeMs {
		return 0
	}
	fadeElapsed := elapsed - titleScreenMenuPanelResizeMs
	if fadeElapsed >= titleScreenMenuContentFadeMs {
		return 1
	}
	return easeInOut(float64(fadeElapsed) / float64(titleScreenMenuContentFadeMs))
}

func menuTransitionOutgoingSlide(forward bool, panelEased float64) float64 {
	slide := float64(titleScreenMenuSlidePx)
	if forward {
		return -slide * panelEased
	}
	return slide * panelEased
}

func menuTransitionIncomingSlide(forward bool, contentEased float64) float64 {
	slide := float64(titleScreenMenuSlidePx)
	if forward {
		return slide * (1 - contentEased)
	}
	return -slide * (1 - contentEased)
}

func lerpFloat(a, b, t float64) float64 {
	return a + (b-a)*t
}

func lerpInt(a, b int, t float64) int {
	return int(math.Round(lerpFloat(float64(a), float64(b), t)))
}

func settingsMenuHeightAnimDurationMs() int64 {
	return titleScreenMenuPanelResizeMs
}

func settingsMenuHeightAnimProgress(nowMs, startMs int64) float64 {
	if startMs == 0 {
		return 1
	}
	elapsed := nowMs - startMs
	dur := settingsMenuHeightAnimDurationMs()
	if elapsed >= dur {
		return 1
	}
	return easeInOut(float64(elapsed) / float64(dur))
}

func menuHeightAnimDurationMs(title string) int64 {
	if title == "Settings" {
		return settingsMenuHeightAnimDurationMs()
	}
	return 200
}

// settingsMenuCurrentHeight returns the settings panel height at nowMs, including any in-flight resize.
func (e *EbitenRenderer) settingsMenuCurrentHeight(nowMs int64) float64 {
	if e.settingsMenuHeightBaseline <= 0 && !e.menuHeightAnimating {
		return 0
	}
	if !e.menuHeightAnimating {
		return e.settingsMenuHeightBaseline
	}
	progress := settingsMenuHeightAnimProgress(nowMs, e.menuHeightAnimStartTime)
	return lerpFloat(e.menuHeightAnimStartHeight, e.menuHeightAnimTargetHeight, progress)
}

// beginSettingsMenuHeightAnim starts or retargets a settings tab height tween from the current visual height.
func (e *EbitenRenderer) beginSettingsMenuHeightAnim(nowMs int64, targetHeight float64) {
	if e.genericMenuTitle != "Settings" || e.menuPanelTransitionAnimating {
		return
	}
	startH := e.settingsMenuCurrentHeight(nowMs)
	if startH <= 0 {
		startH = e.settingsMenuHeightBaseline
	}
	if startH <= 0 || math.Abs(targetHeight-startH) <= 0.5 {
		return
	}
	e.menuHeightAnimStartHeight = startH
	e.menuHeightAnimTargetHeight = targetHeight
	e.menuHeightAnimStartTime = nowMs
	e.menuHeightAnimating = true
}

// finishMenuHeightAnimationIfDone clears height animation state once the tween completes.
// Draw runs every frame; RenderMenu only runs on input, so completion must be detected here too.
func (e *EbitenRenderer) finishMenuHeightAnimationIfDone(nowMs int64) {
	if !e.menuHeightAnimating || e.menuPanelTransitionAnimating {
		return
	}
	if nowMs-e.menuHeightAnimStartTime < menuHeightAnimDurationMs(e.genericMenuTitle) {
		return
	}
	e.menuHeightAnimating = false
	if e.settingsMenuHeightBaseline > 0 {
		e.settingsMenuHeightBaseline = e.menuHeightAnimTargetHeight
	}
	e.prevMenuItems = nil
	e.prevMenuTitle = ""
	e.prevMenuHelpText = ""
	e.prevMenuSelected = 0
}

// settingsMenuUsesFixedTop reports whether the panel should use settingsMenuPanelTopY.
func settingsMenuUsesFixedTop(title string) bool {
	return title == "Settings"
}

// finishPanelTransitionIfDone clears title-screen transition state once the animation
// completes. Draw runs every frame; RenderMenu only runs on input, so completion must
// be detected here to avoid stale transition flags affecting layout.
func (e *EbitenRenderer) finishPanelTransitionIfDone(nowMs int64) {
	e.genericMenuMutex.Lock()
	defer e.genericMenuMutex.Unlock()
	if !e.menuPanelTransitionAnimating {
		return
	}
	if _, done := menuTransitionProgress(nowMs, e.menuPanelTransitionStartMs); !done {
		return
	}
	e.menuPanelTransitionAnimating = false
	e.prevMenuItems = nil
	e.prevMenuTitle = ""
	e.prevMenuHelpText = ""
	e.prevMenuSelected = 0
}

// menuPanelContentParams holds everything needed to draw the inner menu list once.
type menuPanelContentParams struct {
	items    []gamemenu.MenuItem
	labels   []string
	title    string
	helpText string
	selected int

	panelX, panelY, panelW int
	panelH                 float64
	contentSlideX          float64
	alpha                  float64

	highlightAnimating                       bool
	highlightAnimStartIndex, highlightTarget int
	highlightAnimStartWidth, highlightTargetWidth float64
	highlightAnimStartTime                   int64

	screenWidth, screenHeight int
	maintOverlayStable        bool
	devMenuRight              bool
	currentCellCharsMenu      bool
	useVectorAA               bool
}
