package entities

import "testing"

func TestTypeRequiresPower(t *testing.T) {
	tests := []struct {
		typ  RepairType
		want bool
	}{
		{RepairPressureValve, true},
		{RepairSignalCalibrator, true},
		{RepairPowerCoupler, true},
		{RepairWastePump, true},
		{RepairConduitSplice, false},
	}
	for _, tc := range tests {
		if got := TypeRequiresPower(tc.typ); got != tc.want {
			t.Errorf("TypeRequiresPower(%q) = %v, want %v", tc.typ, got, tc.want)
		}
	}
}

func TestNeedsLongUse(t *testing.T) {
	valve := NewRepairObjective("v", RepairPressureValve, "A", 0, 0)
	if !valve.NeedsLongUse() {
		t.Fatal("pressure valve should need long use")
	}
	coupler := NewRepairObjective("c", RepairPowerCoupler, "A", 0, 0)
	if coupler.NeedsLongUse() {
		t.Fatal("power coupler should use tap crank, not long use")
	}
	signal := NewRepairObjective("s", RepairSignalCalibrator, "A", 0, 0)
	if signal.NeedsLongUse() {
		t.Fatal("signal calibrator should not need long use")
	}
}

func TestCouplerCrankHint(t *testing.T) {
	coupler := NewRepairObjective("c", RepairPowerCoupler, "A", 0, 0)
	if got := coupler.CouplerCrankHint(); got == "" {
		t.Fatal("expected crank hint text")
	}
}

func TestNeedsLivePower(t *testing.T) {
	valve := NewRepairObjective("v", RepairPressureValve, "A", 0, 0)
	if !valve.NeedsLivePower() {
		t.Fatal("pressure valve should need live power")
	}
	splice := NewRepairObjective("c", RepairConduitSplice, "Corridor", 0, 0)
	if splice.NeedsLivePower() {
		t.Fatal("conduit splice should not need live power")
	}
}

func TestRepairDependencyReport_ValveToCouplerIsCausal(t *testing.T) {
	valve := NewRepairObjective("v", RepairPressureValve, "Pressure", 0, 0)
	coupler := NewRepairObjective("c", RepairPowerCoupler, "Transfer", 0, 1)
	report := RepairDependencyReport(coupler, valve)
	if report.Symptom != "COUPLER DRIVE INHIBITED" {
		t.Fatalf("symptom = %q", report.Symptom)
	}
	if report.Diagnosis != "PRESSURE CONTROL INPUT OUT OF TOLERANCE" {
		t.Fatalf("diagnosis = %q", report.Diagnosis)
	}
	if report.Prediction == "" || report.Restored != "COUPLER INTERLOCK RELEASED" {
		t.Fatalf("incomplete causal report: %+v", report)
	}
}

func TestRepairInspectionPersistsOnObjective(t *testing.T) {
	repair := NewRepairObjective("v", RepairPressureValve, "Pressure", 0, 0)
	if repair.Inspected {
		t.Fatal("new repair should not start inspected")
	}
	repair.Inspect()
	if !repair.Inspected {
		t.Fatal("Inspect should persist knowledge on the repair objective")
	}
}
