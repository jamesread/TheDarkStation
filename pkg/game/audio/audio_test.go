package audio

import (
	"testing"

	"darkstation/pkg/resources"
)

func TestDecodeEmbeddedMenuSFX(t *testing.T) {
	sources := map[string][]byte{
		MenuConfirm:     resources.MenuConfirmMP3,
		MenuUp:          resources.MenuUpMP3,
		MenuDown:        resources.MenuDownMP3,
		PowerGridPulse:  resources.PowerGridPulseMP3,
		LiftWhir:        resources.LiftWhirMP3,
		UseGenerator:    resources.UseGeneratorMP3,
		UseDoor:         resources.UseDoorMP3,
		UseRepair:       resources.UseRepairMP3,
		UseCoupler:      resources.UseCouplerMP3,
		UseFurniture:    resources.UseFurnitureMP3,
		UseTerminal:     resources.UseTerminalMP3,
		UseMaint:        resources.UseMaintMP3,
		UseRelay:        resources.UseRelayMP3,
		UseHazard:       resources.UseHazardMP3,
		UseLift:         resources.UseLiftMP3,
		UseDeny:         resources.UseDenyMP3,
		UsePickup:       resources.UsePickupMP3,
		UseBattery:      resources.UseBatteryMP3,
		SignalLockUp:    resources.SignalLockUpMP3,
		SignalLockDown:  resources.SignalLockDownMP3,
		SignalLockReady: resources.SignalLockReadyMP3,
		HazardClear:     resources.HazardClearMP3,
	}
	for name, data := range sources {
		if len(data) == 0 {
			t.Fatalf("%s: embedded MP3 is empty", name)
		}
		pcm, err := decodeMP3(data)
		if err != nil {
			t.Fatalf("%s: decode: %v", name, err)
		}
		if len(pcm) == 0 {
			t.Fatalf("%s: decoded PCM is empty", name)
		}
	}
}

func TestPlayBeforeInitIsNoOp(t *testing.T) {
	// Must not panic when audio context was never created (unit tests, headless CI).
	PlayMenuConfirm()
	PlayMenuUp()
	PlayMenuDown()
	PlayUseGenerator()
	PlayUseDoor()
	PlayUseRepair()
	PlayUseCoupler()
	PlayUseFurniture()
	PlayUseTerminal()
	PlayUseMaint()
	PlayUseRelay()
	PlayUseHazard()
	PlayUseLift()
	PlayUseDeny()
	PlayUsePickup()
	PlayUseBattery()
	PlaySignalLockUp()
	PlaySignalLockDown()
	PlaySignalLockReady()
	PlayHazardClear()
}

func TestHazardClearSoundSpansFadeAnimation(t *testing.T) {
	pcm, err := decodeMP3(resources.HazardClearMP3)
	if err != nil {
		t.Fatalf("decode hazard clear: %v", err)
	}
	const bytesPerStereoFrame = 4
	minBytes := sampleRate * bytesPerStereoFrame * 3
	if len(pcm) < minBytes {
		t.Fatalf("hazard-clear PCM = %d bytes, want at least three seconds (%d)", len(pcm), minBytes)
	}
}

func TestSetHazardClearFadeProgressControlsClipGain(t *testing.T) {
	mu.Lock()
	previous := clips
	clips = map[string]*clip{HazardClear: {gain: 1}}
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		clips = previous
		mu.Unlock()
	})

	SetHazardClearFadeProgress(0.4)
	mu.Lock()
	got := clips[HazardClear].gain
	mu.Unlock()
	if got < 0.599 || got > 0.601 {
		t.Fatalf("gain at 40%% progress = %v, want 0.6", got)
	}
	SetHazardClearFadeProgress(2)
	mu.Lock()
	got = clips[HazardClear].gain
	mu.Unlock()
	if got != 0 {
		t.Fatalf("gain past fade end = %v, want 0", got)
	}
}

func TestVolumeAndMuteAccessors(t *testing.T) {
	SetVolumePercent(70)
	if VolumePercent() != 70 {
		t.Fatalf("VolumePercent = %d, want 70", VolumePercent())
	}
	SetVolumePercent(200)
	if VolumePercent() != 100 {
		t.Fatalf("VolumePercent = %d, want 100 after clamp", VolumePercent())
	}
	SetMute(true)
	if !Muted() {
		t.Fatal("Muted = false, want true")
	}
	SetMute(false)
	if Muted() {
		t.Fatal("Muted = true, want false")
	}
}
