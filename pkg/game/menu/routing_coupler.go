package menu

import (
	"fmt"
	"math"
	"strings"

	engineinput "darkstation/pkg/engine/input"
	"darkstation/pkg/engine/world"
	"darkstation/pkg/game/audio"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/entities"
	"darkstation/pkg/game/state"
)

const routingCouplerMeterWidth = 11

// RoutingCouplerMenuHandler runs the lift routing coupler alignment mini-game.
type RoutingCouplerMenuHandler struct {
	g          *state.Game
	cell       *world.Cell
	repair     *entities.RepairObjective
	onComplete func()
	targets    []int
	values     []int
	params     entities.RoutingCouplerParams
	targetDeck int
}

// NewRoutingCouplerMenuHandler builds a handler for one routing coupler repair.
func NewRoutingCouplerMenuHandler(g *state.Game, cell *world.Cell, repair *entities.RepairObjective, onComplete func()) *RoutingCouplerMenuHandler {
	targetDeck := repair.RoutingTargetLevel()
	params := entities.RoutingCouplerDifficulty(targetDeck)
	seed := entities.RoutingCouplerSeed(g.LevelSeed, repair.ID)
	targets := entities.RoutingCouplerTargets(seed, params.Axes)
	values := entities.RoutingCouplerInitialValues(seed, targets, targetDeck)
	return &RoutingCouplerMenuHandler{
		g:          g,
		cell:       cell,
		repair:     repair,
		onComplete: onComplete,
		targets:    targets,
		values:     values,
		params:     params,
		targetDeck: targetDeck,
	}
}

func (h *RoutingCouplerMenuHandler) GetTitle() string {
	if h.repair != nil && h.repair.Name != "" {
		return h.repair.Name
	}
	return "Lift Routing Coupler"
}

func (h *RoutingCouplerMenuHandler) GetInstructions(selected MenuItem) string {
	if _, ok := selected.(*RoutingCouplerAxisItem); ok {
		return fmt.Sprintf("%s to tune, %s to lock when signal is strong enough, %s to close",
			engineinput.HintMaintCycle(), engineinput.HintConfirm(), engineinput.HintMenuClose())
	}
	return engineinput.HintMenuInstructionsGameplay()
}

func (h *RoutingCouplerMenuHandler) OnSelect(item MenuItem, index int) {}

func (h *RoutingCouplerMenuHandler) OnActivate(item MenuItem, index int) (shouldClose bool, helpText string) {
	switch item.(type) {
	case *RoutingCouplerCommitItem:
		if !h.canLock() {
			return false, fmt.Sprintf("Signal lock too weak — center the tuners above %.0f%%", h.params.LockThreshold*100)
		}
		if h.onComplete != nil {
			h.onComplete()
		}
		return true, ""
	default:
		return false, ""
	}
}

func (h *RoutingCouplerMenuHandler) OnExit() {}

func (h *RoutingCouplerMenuHandler) ShouldCloseOnAnyAction() bool {
	return false
}

func (h *RoutingCouplerMenuHandler) HandleCancelShortcut(g *state.Game) bool {
	return true
}

func (h *RoutingCouplerMenuHandler) GetMenuItems() []MenuItem {
	items := []MenuItem{
		&RoutingCouplerInfoItem{Handler: h},
		&RoutingCouplerLockItem{Handler: h},
	}
	for i, name := range entities.RoutingCouplerAxisNames(h.params.Axes) {
		items = append(items, &RoutingCouplerAxisItem{
			Handler: h,
			Index:   i,
			Name:    name,
		})
	}
	items = append(items, &RoutingCouplerCommitItem{Handler: h})
	return items
}

func (h *RoutingCouplerMenuHandler) signalLock() float64 {
	return entities.RoutingCouplerSignalLock(h.targets, h.values, h.params.MaxDist)
}

func (h *RoutingCouplerMenuHandler) canLock() bool {
	return entities.RoutingCouplerLocked(h.targets, h.values, h.params)
}

func (h *RoutingCouplerMenuHandler) adjustAxis(index, delta int) {
	if index < 0 || index >= len(h.values) {
		return
	}
	h.values[index] = entities.RoutingCouplerAdjustValue(h.values[index], delta, h.params.Step)
}

func routingCouplerBar(value int, width int) string {
	if width <= 0 {
		return ""
	}
	filled := value * width / 100
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// routingCouplerTuningMeter draws a bipolar null meter: center │ is the sweet spot,
// ● shows error left/right of target (reveals alignment quality without numeric targets).
func routingCouplerTuningMeter(target, value, maxDist, width int) string {
	if width < 5 {
		width = 5
	}
	if width%2 == 0 {
		width++
	}
	center := width / 2
	cells := []rune(strings.Repeat("─", width))
	cells[center] = '│'
	if maxDist < 1 {
		maxDist = 1
	}
	err := value - target
	offset := int(math.Round(float64(err) / float64(maxDist) * float64(center)))
	if offset < -center {
		offset = -center
	}
	if offset > center {
		offset = center
	}
	pos := center + offset
	cells[pos] = '●'
	return string(cells)
}

// routingCouplerLockColored wraps content in markup that tracks alignment progress:
// red (UNPOWERED) at 0%, warming yellow (DOOR) as lock approaches threshold, green (POWERED) when lockable.
func routingCouplerLockColored(lock, threshold float64, content string) string {
	if lock <= 0.001 {
		return fmt.Sprintf("UNPOWERED{%s}", content)
	}
	if threshold <= 0 {
		return fmt.Sprintf("POWERED{%s}", content)
	}
	ratio := lock / threshold
	switch {
	case ratio >= 1.0:
		return fmt.Sprintf("POWERED{%s}", content)
	case ratio >= 0.45:
		return fmt.Sprintf("DOOR{%s}", content)
	default:
		return fmt.Sprintf("UNPOWERED{%s}", content)
	}
}

func routingCouplerLockLabel(lock, threshold float64) string {
	pct := int(lock*100 + 0.5)
	need := int(threshold*100 + 0.5)
	bar := routingCouplerBar(pct, 12)
	body := fmt.Sprintf("%s  %d%%", bar, pct)
	return fmt.Sprintf("Signal lock\t%s\tSUBTLE{need %d%%}", routingCouplerLockColored(lock, threshold, body), need)
}

func playRoutingCouplerLockFeedback(before, after, threshold float64) {
	const eps = 0.0005
	if after > before+eps {
		if before < threshold && after >= threshold {
			audio.PlaySignalLockReady()
			return
		}
		audio.PlaySignalLockUp()
		return
	}
	if after < before-eps {
		audio.PlaySignalLockDown()
	}
}

// RoutingCouplerInfoItem shows which deck this coupler routes to.
type RoutingCouplerInfoItem struct {
	Handler *RoutingCouplerMenuHandler
}

func (i *RoutingCouplerInfoItem) GetLabel() string {
	h := i.Handler
	if h == nil {
		return "Lift routing alignment"
	}
	title := ""
	if h.g != nil {
		deckID := h.targetDeck - 1
		if deckID >= 0 && deckID < deck.TotalDecks {
			theme := deck.ThemeDisplayName(h.g.ThemeForDeck(deckID))
			if theme != "" {
				title = fmt.Sprintf(" — %s", theme)
			}
		}
	}
	return fmt.Sprintf("Route lift to\tACTION{Deck %d%s}", h.targetDeck, title)
}

func (i *RoutingCouplerInfoItem) IsSelectable() bool { return false }

func (i *RoutingCouplerInfoItem) GetHelpText() string {
	return "Center each tuner on the null mark until signal lock is strong enough"
}

// RoutingCouplerLockItem shows current signal lock strength.
type RoutingCouplerLockItem struct {
	Handler *RoutingCouplerMenuHandler
}

func (l *RoutingCouplerLockItem) GetLabel() string {
	if l.Handler == nil {
		return "Signal lock\tUNPOWERED{0%}"
	}
	return routingCouplerLockLabel(l.Handler.signalLock(), l.Handler.params.LockThreshold)
}

func (l *RoutingCouplerLockItem) IsSelectable() bool { return false }

func (l *RoutingCouplerLockItem) GetHelpText() string {
	if l.Handler != nil && l.Handler.canLock() {
		return "POWERED{Coupling aligned — lock when ready}"
	}
	return "Tune phase, gain, and bias toward the center null until lock is ready"
}

// RoutingCouplerAxisItem is one adjustable alignment axis.
type RoutingCouplerAxisItem struct {
	Handler *RoutingCouplerMenuHandler
	Index   int
	Name    string
}

func (a *RoutingCouplerAxisItem) GetLabel() string {
	if a.Handler == nil || a.Index < 0 || a.Index >= len(a.Handler.values) {
		return a.Name
	}
	value := a.Handler.values[a.Index]
	target := a.Handler.targets[a.Index]
	meter := routingCouplerTuningMeter(target, value, a.Handler.params.MaxDist, routingCouplerMeterWidth)
	score := entities.RoutingCouplerAxisScore(target, value, a.Handler.params.MaxDist)
	return fmt.Sprintf("%s\t%s\tSUBTLE{%s}", a.Name, routingCouplerLockColored(score, 0.85, meter), engineinput.HintMaintCycle())
}

func (a *RoutingCouplerAxisItem) IsSelectable() bool { return true }

func (a *RoutingCouplerAxisItem) GetHelpText() string {
	return fmt.Sprintf("%s: move the marker onto the center null", a.Name)
}

func (a *RoutingCouplerAxisItem) CanCycle() bool { return true }

// HandlesOwnCycleSound skips the generic menu confirm beep; lock-quality SFX is played instead.
func (a *RoutingCouplerAxisItem) HandlesOwnCycleSound() bool { return true }

func (a *RoutingCouplerAxisItem) HandleCycle(delta int) (bool, string) {
	if a.Handler == nil {
		return false, ""
	}
	before := a.Handler.signalLock()
	threshold := a.Handler.params.LockThreshold
	a.Handler.adjustAxis(a.Index, delta)
	after := a.Handler.signalLock()
	playRoutingCouplerLockFeedback(before, after, threshold)
	pct := int(after*100 + 0.5)
	body := fmt.Sprintf("Signal lock %d%%", pct)
	return true, routingCouplerLockColored(after, threshold, body)
}

// RoutingCouplerCommitItem locks the coupling when alignment is sufficient.
type RoutingCouplerCommitItem struct {
	Handler *RoutingCouplerMenuHandler
}

func (c *RoutingCouplerCommitItem) GetLabel() string {
	if c.Handler != nil && c.Handler.canLock() {
		return fmt.Sprintf("Lock coupling\tPOWERED{ready}\tSUBTLE{%s}", engineinput.HintConfirm())
	}
	return "Lock coupling\tSUBTLE{signal too weak}"
}

func (c *RoutingCouplerCommitItem) IsSelectable() bool {
	return c.Handler != nil && c.Handler.canLock()
}

func (c *RoutingCouplerCommitItem) GetHelpText() string {
	if c.Handler != nil && c.Handler.canLock() {
		return engineinput.HintPressConfirmTo("lock the coupling")
	}
	if c.Handler != nil {
		return fmt.Sprintf("Raise signal lock to at least %.0f%%", c.Handler.params.LockThreshold*100)
	}
	return ""
}
