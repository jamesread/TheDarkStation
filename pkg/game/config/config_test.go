package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig_audioDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.SFXVolume != 100 {
		t.Fatalf("SFXVolume = %d, want 100", cfg.SFXVolume)
	}
	if cfg.MusicVolume != 70 {
		t.Fatalf("MusicVolume = %d, want 70", cfg.MusicVolume)
	}
	if cfg.Mute {
		t.Fatal("Mute default should be false")
	}
	if cfg.MonitorIndex != -1 {
		t.Fatalf("MonitorIndex = %d, want -1", cfg.MonitorIndex)
	}
}

func TestConfig_SaveLoadMonitorPreference(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfg := DefaultConfig()
	if err := cfg.SetMonitorPreference("DP-2", 1, 2560, 1440); err != nil {
		t.Fatalf("SetMonitorPreference: %v", err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.MonitorName != "DP-2" || got.MonitorIndex != 1 || got.MonitorWidth != 2560 || got.MonitorHeight != 1440 {
		t.Fatalf("loaded monitor pref = %+v", got)
	}
}

func TestConfig_SaveLoadAudio(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.ini")
	cfg := DefaultConfig()
	cfg.configPath = path
	cfg.SFXVolume = 40
	cfg.MusicVolume = 55
	cfg.Mute = true
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded := DefaultConfig()
	loaded.configPath = path
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	_ = data

	// Load via path override: point XDG is awkward; parse by reusing Load after SetCurrent path.
	// Instead, write through Save and re-open with a helper path set on a fresh Load after swapping env.
	t.Setenv("XDG_CONFIG_HOME", dir)
	// getConfigPath uses DarkStation/settings.ini under XDG_CONFIG_HOME
	cfgDir := filepath.Join(dir, "DarkStation")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "settings.ini"), data, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.SFXVolume != 40 {
		t.Fatalf("SFXVolume = %d, want 40", got.SFXVolume)
	}
	if got.MusicVolume != 55 {
		t.Fatalf("MusicVolume = %d, want 55", got.MusicVolume)
	}
	if !got.Mute {
		t.Fatal("Mute = false, want true")
	}
}

func TestClampPercent(t *testing.T) {
	cases := []struct {
		in, want int
	}{
		{-10, 0},
		{0, 0},
		{50, 50},
		{100, 100},
		{150, 100},
	}
	for _, tc := range cases {
		if got := clampPercent(tc.in); got != tc.want {
			t.Fatalf("clampPercent(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
