package audio

import "testing"

func TestSetPowerGridPulseActiveBeforeInitIsNoOp(t *testing.T) {
	if ready {
		t.Skip("audio already initialized")
	}
	SetPowerGridPulseActive(true)
	SetPowerGridPulseActive(false)
	SetLiftWhirActive(true)
	SetLiftWhirActive(false)
}

func TestPowerGridPulsePhase_wraps(t *testing.T) {
	period := PowerGridPulsePeriodMs()
	if period < 100 {
		t.Fatalf("period = %v, want >= 100", period)
	}
	p0 := PowerGridPulsePhase(0)
	p1 := PowerGridPulsePhase(int64(period))
	if p0 < 0 || p0 >= 1 {
		t.Fatalf("phase(0) = %v, want in [0,1)", p0)
	}
	if p1 < 0 || p1 >= 1 {
		t.Fatalf("phase(period) = %v, want in [0,1)", p1)
	}
}
