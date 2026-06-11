package entities

import "fmt"

// RepairType identifies a deck repair mechanic.
type RepairType string

const (
	RepairPressureValve    RepairType = "pressure_valve"
	RepairSignalCalibrator RepairType = "signal_calibrator"
	RepairPowerCoupler     RepairType = "power_coupler"
	RepairWastePump        RepairType = "waste_pump"
)

// RepairStatus tracks a repair objective's lifecycle.
type RepairStatus string

const (
	RepairPending  RepairStatus = "pending"
	RepairDraining RepairStatus = "draining"
	RepairComplete RepairStatus = "complete"
)

const (
	// WastePumpDrainDurationMs is long enough to be felt but short enough not to stall play.
	WastePumpDrainDurationMs int64 = 6000
	// SlimePopDurationMs is how long a drained slime cell plays its pop-off animation.
	SlimePopDurationMs     int64 = 420
	SignalCalibrationSteps       = 3
)

// BlockerCell is one toxic-slime tile gating access near the exit lift.
type BlockerCell struct {
	Row int
	Col int
}

// RepairObjective is a deck-local objective placed as a physical device, with
// optional blockers such as toxic slime tied to completion.
type RepairObjective struct {
	ID            string
	Type          RepairType
	Name          string
	RoomName      string
	Description   string
	PrereqIDs     []string
	RequiresPower bool
	SkipExitGate  bool // When true, does not block the local lift (e.g. cross-deck routing repairs)

	DeviceRow int
	DeviceCol int

	BlockerName string
	BlockerRow  int
	BlockerCol  int
	// BlockerCells lists every slime tile (drain order: farthest from exit first).
	BlockerCells []BlockerCell

	Status          RepairStatus
	DrainCleared    int // slime cells cleared during timed drain (indices into BlockerCells)
	StartedAtMs     int64
	CompleteAtMs    int64
	CalibrationStep int
}

// NewRepairObjective creates a repair objective with defaults for its type.
func NewRepairObjective(id string, typ RepairType, roomName string, row, col int) *RepairObjective {
	name, desc := RepairDisplay(typ)
	return &RepairObjective{
		ID:          id,
		Type:        typ,
		Name:        name,
		RoomName:    roomName,
		Description: desc,
		DeviceRow:   row,
		DeviceCol:   col,
		BlockerRow:  -1,
		BlockerCol:  -1,
		Status:      RepairPending,
	}
}

// RepairDisplay returns the user-facing name and description for a repair type.
func RepairDisplay(typ RepairType) (name, description string) {
	switch typ {
	case RepairPressureValve:
		return "Pressure Valve", "Bleed pressure from the auxiliary pipework."
	case RepairSignalCalibrator:
		return "Signal Calibrator", "Align the routing signal sequence."
	case RepairPowerCoupler:
		return "Power Coupler", "Re-seat the live power coupler."
	case RepairWastePump:
		return "Waste Pump", "Restore the pump and drain toxic slime."
	default:
		return "Repair Station", "Restore a damaged station system."
	}
}

// IsComplete reports whether this objective no longer blocks progress.
func (r *RepairObjective) IsComplete() bool {
	return r != nil && r.Status == RepairComplete
}

// IsRoutingCoupler reports cross-deck lift routing payoff devices (SkipExitGate repairs).
func (r *RepairObjective) IsRoutingCoupler() bool {
	return r != nil && r.SkipExitGate
}

// IsDraining reports whether this objective is waiting on a timed drain.
func (r *RepairObjective) IsDraining() bool {
	return r != nil && r.Status == RepairDraining
}

// BlocksMovement reports whether any linked blocker tile is still impassable.
func (r *RepairObjective) BlocksMovement() bool {
	if r == nil || r.BlockerName == "" || r.Status == RepairComplete {
		return false
	}
	for _, bc := range r.BlockerCellList() {
		if r.BlockerBlocksCell(bc.Row, bc.Col) {
			return true
		}
	}
	return false
}

// BlockerCellList returns all slime tiles for this objective.
func (r *RepairObjective) BlockerCellList() []BlockerCell {
	if r == nil {
		return nil
	}
	if len(r.BlockerCells) > 0 {
		return r.BlockerCells
	}
	if r.BlockerRow >= 0 && r.BlockerCol >= 0 {
		return []BlockerCell{{Row: r.BlockerRow, Col: r.BlockerCol}}
	}
	return nil
}

// BlockerBlocksCell reports whether a specific slime tile still blocks movement.
func (r *RepairObjective) BlockerBlocksCell(row, col int) bool {
	if r == nil || r.BlockerName == "" || r.Status == RepairComplete {
		return false
	}
	idx := r.blockerIndex(row, col)
	if idx < 0 {
		return false
	}
	if r.Status != RepairDraining {
		return true
	}
	return idx >= r.DrainCleared
}

func (r *RepairObjective) blockerIndex(row, col int) int {
	for i, bc := range r.BlockerCellList() {
		if bc.Row == row && bc.Col == col {
			return i
		}
	}
	return -1
}

// DrainProgress returns 0..1 for the waste-pump drain timer.
func (r *RepairObjective) DrainProgress(nowMs int64) float64 {
	if r == nil || !r.IsDraining() || r.CompleteAtMs <= r.StartedAtMs {
		return 0
	}
	elapsed := nowMs - r.StartedAtMs
	dur := r.CompleteAtMs - r.StartedAtMs
	if elapsed <= 0 {
		return 0
	}
	if elapsed >= dur {
		return 1
	}
	return float64(elapsed) / float64(dur)
}

// TargetDrainCleared returns how many slime tiles should be gone at nowMs.
func (r *RepairObjective) TargetDrainCleared(nowMs int64) int {
	cells := r.BlockerCellList()
	if len(cells) == 0 || !r.IsDraining() {
		return 0
	}
	if nowMs >= r.CompleteAtMs {
		return len(cells)
	}
	elapsed := nowMs - r.StartedAtMs
	dur := r.CompleteAtMs - r.StartedAtMs
	if dur <= 0 {
		return len(cells)
	}
	cleared := int(float64(len(cells)) * float64(elapsed) / float64(dur))
	if cleared > len(cells) {
		return len(cells)
	}
	return cleared
}

// NeedsLongUse reports whether this repair is completed by a hold-to-use action.
func (r *RepairObjective) NeedsLongUse() bool {
	if r == nil || r.Status != RepairPending {
		return false
	}
	return r.Type != RepairSignalCalibrator
}

// BeginTimedCompletion starts the timed completion phase for repairs that do
// not finish instantly after physical repair.
func (r *RepairObjective) BeginTimedCompletion(nowMs int64) {
	if r == nil || r.Status != RepairPending {
		return
	}
	if r.Type == RepairWastePump && r.BlockerName != "" {
		r.Status = RepairDraining
		r.StartedAtMs = nowMs
		r.CompleteAtMs = nowMs + WastePumpDrainDurationMs
		r.DrainCleared = 0
		return
	}
	r.Status = RepairComplete
}

// Complete marks a repair and its linked blocker as fully resolved.
func (r *RepairObjective) Complete() {
	if r == nil {
		return
	}
	r.Status = RepairComplete
	r.CompleteAtMs = 0
}

// CalibrationLabel returns the next calibration step label.
func (r *RepairObjective) CalibrationLabel() string {
	if r == nil {
		return ""
	}
	labels := []string{"A", "B", "C"}
	if r.CalibrationStep < 0 || r.CalibrationStep >= len(labels) {
		return fmt.Sprintf("%d", r.CalibrationStep+1)
	}
	return labels[r.CalibrationStep]
}
