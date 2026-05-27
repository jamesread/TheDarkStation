package ebiten

import "testing"

func TestReportLevelGenProgress_clampsStepAndComputesRatio(t *testing.T) {
	e := New()

	e.BeginLevelGen(3, 10)
	load := e.levelGenSnapshot()
	if !load.active || load.level != 3 || load.total != 10 {
		t.Fatalf("BeginLevelGen: got active=%v level=%d total=%d", load.active, load.level, load.total)
	}

	e.ReportLevelGenProgress(4, 10, "Furnishing rooms")
	load = e.levelGenSnapshot()
	if load.step != 4 || load.label != "Furnishing rooms" {
		t.Fatalf("step/label: got step=%d label=%q", load.step, load.label)
	}
	if load.progress < 0.39 || load.progress > 0.41 {
		t.Fatalf("progress for step 4/10: got %v", load.progress)
	}

	e.ReportLevelGenProgress(99, 10, "Done")
	load = e.levelGenSnapshot()
	if load.step != 10 {
		t.Fatalf("expected clamped step 10, got %d", load.step)
	}
	if load.progress != 1 {
		t.Fatalf("expected progress 1, got %v", load.progress)
	}

	e.ClearLevelGenProgress()
	load = e.levelGenSnapshot()
	if load.active {
		t.Fatal("expected loading inactive after clear")
	}
}
