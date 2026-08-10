package menu

import (
	"strings"
	"testing"

	"darkstation/pkg/game/audio"
	"darkstation/pkg/game/config"
)

func useTempAudioConfig(t *testing.T, volume int, mute bool) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg := config.DefaultConfig()
	cfg.SFXVolume = volume
	cfg.MusicVolume = 70
	cfg.Mute = mute
	config.SetCurrent(cfg)
	audio.SetVolumePercent(volume)
	audio.SetMusicVolumePercent(70)
	audio.SetMute(mute)
}

func TestSFXVolumeMenuItem_HandleCycle(t *testing.T) {
	useTempAudioConfig(t, 50, false)

	item := &SFXVolumeMenuItem{}
	consumed, help := item.HandleCycle(1)
	if !consumed {
		t.Fatal("cycle should be consumed")
	}
	if audio.VolumePercent() != 60 {
		t.Fatalf("volume = %d, want 60", audio.VolumePercent())
	}
	if !strings.Contains(help, "60%") {
		t.Fatalf("help = %q, want 60%%", help)
	}

	consumed, _ = item.HandleCycle(-1)
	if !consumed || audio.VolumePercent() != 50 {
		t.Fatalf("volume after down = %d, want 50", audio.VolumePercent())
	}

	audio.SetVolumePercent(100)
	_, _ = item.HandleCycle(1)
	if audio.VolumePercent() != 100 {
		t.Fatalf("volume should clamp at 100, got %d", audio.VolumePercent())
	}

	audio.SetVolumePercent(0)
	_, _ = item.HandleCycle(-1)
	if audio.VolumePercent() != 0 {
		t.Fatalf("volume should clamp at 0, got %d", audio.VolumePercent())
	}
}

func TestMuteMenuItem_HandleCycle(t *testing.T) {
	useTempAudioConfig(t, 80, false)

	item := &MuteMenuItem{}
	consumed, help := item.HandleCycle(1)
	if !consumed || !audio.Muted() {
		t.Fatalf("mute on failed: muted=%v help=%q", audio.Muted(), help)
	}
	consumed, help = item.HandleCycle(1)
	if !consumed || audio.Muted() {
		t.Fatalf("mute off failed: muted=%v help=%q", audio.Muted(), help)
	}
}

func TestMusicVolumeMenuItem_HandleCycle(t *testing.T) {
	useTempAudioConfig(t, 50, false)
	audio.SetMusicVolumePercent(40)

	item := &MusicVolumeMenuItem{}
	consumed, help := item.HandleCycle(1)
	if !consumed {
		t.Fatal("cycle should be consumed")
	}
	if audio.MusicVolumePercent() != 50 {
		t.Fatalf("music volume = %d, want 50", audio.MusicVolumePercent())
	}
	if !strings.Contains(help, "50%") {
		t.Fatalf("help = %q, want 50%%", help)
	}
}

func TestSFXVolumeMenuItem_unmutesWhenRaised(t *testing.T) {
	useTempAudioConfig(t, 20, true)

	item := &SFXVolumeMenuItem{}
	_, _ = item.HandleCycle(1)
	if audio.Muted() {
		t.Fatal("raising volume should unmute")
	}
	if audio.VolumePercent() != 30 {
		t.Fatalf("volume = %d, want 30", audio.VolumePercent())
	}
}
