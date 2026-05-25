package ebiten

import "testing"

func TestToggleDrawFOVRays(t *testing.T) {
	e := &EbitenRenderer{}
	if e.DrawFOVRaysEnabled() {
		t.Fatal("FOV rays should start off")
	}
	if !e.ToggleDrawFOVRays() {
		t.Fatal("first toggle should enable FOV rays")
	}
	if !e.DrawFOVRaysEnabled() {
		t.Fatal("FOV rays should be on")
	}
	e.SetDrawFOVRays(false)
	if e.DrawFOVRaysEnabled() {
		t.Fatal("SetDrawFOVRays(false) should disable FOV rays")
	}
}

func TestToggleShowFPSCounter(t *testing.T) {
	initCvars()
	e := &EbitenRenderer{}
	if !e.ShowFPSCounterEnabled() {
		t.Fatal("FPS counter should start on (draw.fps=1)")
	}
	if e.ToggleShowFPSCounter() {
		t.Fatal("first toggle should disable FPS counter")
	}
	if e.ShowFPSCounterEnabled() {
		t.Fatal("FPS counter should be off")
	}
	e.SetShowFPSCounter(true)
	if !e.ShowFPSCounterEnabled() {
		t.Fatal("SetShowFPSCounter(true) should enable FPS counter")
	}
}

func TestToggleDrawMapAreaBorder(t *testing.T) {
	e := &EbitenRenderer{}
	if e.DrawMapAreaBorderEnabled() {
		t.Fatal("border should start off")
	}
	if !e.ToggleDrawMapAreaBorder() {
		t.Fatal("first toggle should enable border")
	}
	if !e.DrawMapAreaBorderEnabled() {
		t.Fatal("border should be on")
	}
	e.SetDrawMapAreaBorder(false)
	if e.DrawMapAreaBorderEnabled() {
		t.Fatal("SetDrawMapAreaBorder(false) should disable border")
	}
}
