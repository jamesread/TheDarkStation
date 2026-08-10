// Package audio plays short UI and game sound effects via Ebiten's audio API.
package audio

import (
	"bytes"
	"io"
	"log"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"

	"darkstation/pkg/game/config"
	"darkstation/pkg/resources"
)

const sampleRate = 44100

// Named menu SFX clips.
const (
	MenuConfirm = "menu_confirm"
	MenuUp      = "menu_up"
	MenuDown    = "menu_down"
)

type clip struct {
	pcm    []byte
	player *audio.Player
	gain   float64
}

var (
	mu      sync.Mutex
	ctx     *audio.Context
	clips   map[string]*clip
	ready   bool
	initErr error

	volumePercent = 100
	muted         bool
)

// Init loads embedded SFX and creates the audio context. Safe to call once;
// subsequent calls are no-ops. Failures are logged and Play becomes a no-op
// so headless tests and machines without audio keep working.
func Init() {
	mu.Lock()
	defer mu.Unlock()
	if ready || initErr != nil {
		return
	}

	cfg := config.Current()
	volumePercent = clampPercent(cfg.SFXVolume)
	musicVolumePercent = clampPercent(cfg.MusicVolume)
	muted = cfg.Mute

	ctx = audio.CurrentContext()
	if ctx == nil {
		ctx = audio.NewContext(sampleRate)
	}

	clips = make(map[string]*clip)
	sources := map[string][]byte{
		MenuConfirm:     resources.MenuConfirmMP3,
		MenuUp:          resources.MenuUpMP3,
		MenuDown:        resources.MenuDownMP3,
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
		pcm, err := decodeMP3(data)
		if err != nil {
			initErr = err
			log.Printf("audio: decode %s: %v", name, err)
			return
		}
		clips[name] = &clip{pcm: pcm, gain: 1}
	}
	pulsePCM, err := decodeMP3(resources.PowerGridPulseMP3)
	if err != nil {
		initErr = err
		log.Printf("audio: decode %s: %v", PowerGridPulse, err)
		return
	}
	registerLoop(PowerGridPulse, pulsePCM, 0.55)
	whirPCM, err := decodeMP3(resources.LiftWhirMP3)
	if err != nil {
		initErr = err
		log.Printf("audio: decode %s: %v", LiftWhir, err)
		return
	}
	registerLoop(LiftWhir, whirPCM, 0.5)
	initMusicTracks()
	ready = true
}

func decodeMP3(data []byte) ([]byte, error) {
	stream, err := mp3.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(stream)
}

func clampPercent(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func effectiveVolume() float64 {
	if muted {
		return 0
	}
	return float64(volumePercent) / 100
}

func applyVolumeLocked() {
	vol := effectiveVolume()
	for _, c := range clips {
		if c != nil && c.player != nil {
			c.player.SetVolume(vol * c.gain)
		}
	}
	applyLoopVolumeLocked()
	applyMusicVolumeLocked()
}

// VolumePercent returns the configured SFX volume (0–100), ignoring mute.
func VolumePercent() int {
	mu.Lock()
	defer mu.Unlock()
	return volumePercent
}

// SetVolumePercent sets SFX volume (0–100) and updates active players.
func SetVolumePercent(percent int) {
	mu.Lock()
	defer mu.Unlock()
	volumePercent = clampPercent(percent)
	applyVolumeLocked()
}

// Muted reports whether audio output is muted.
func Muted() bool {
	mu.Lock()
	defer mu.Unlock()
	return muted
}

// SetMute enables or disables mute and updates active players.
func SetMute(on bool) {
	mu.Lock()
	defer mu.Unlock()
	muted = on
	applyVolumeLocked()
}

// Play starts (or restarts) a named clip. No-op if audio is not initialized
// or the effective volume is zero.
func Play(name string) {
	mu.Lock()
	defer mu.Unlock()
	if !ready {
		return
	}
	vol := effectiveVolume()
	if vol <= 0 {
		return
	}
	c, ok := clips[name]
	if !ok || len(c.pcm) == 0 {
		return
	}
	if c.player == nil {
		p := audio.NewPlayerFromBytes(ctx, c.pcm)
		c.player = p
	}
	c.player.SetVolume(vol * c.gain)
	_ = c.player.Rewind()
	c.player.Play()
}

// setClipGain changes one playing clip's gain relative to the configured SFX
// volume. It is used for effects whose loudness follows an animation.
func setClipGain(name string, gain float64) {
	if gain < 0 {
		gain = 0
	}
	if gain > 1 {
		gain = 1
	}
	mu.Lock()
	defer mu.Unlock()
	c := clips[name]
	if c == nil {
		return
	}
	c.gain = gain
	if c.player != nil {
		c.player.SetVolume(effectiveVolume() * gain)
	}
}

// PlayMenuConfirm plays the menu selection/confirm beep.
func PlayMenuConfirm() { Play(MenuConfirm) }

// PlayMenuUp plays the menu navigate-up tick.
func PlayMenuUp() { Play(MenuUp) }

// PlayMenuDown plays the menu navigate-down tick.
func PlayMenuDown() { Play(MenuDown) }
