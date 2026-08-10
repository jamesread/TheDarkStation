package audio

import (
	"bytes"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// Named looping SFX clips.
const (
	PowerGridPulse = "power_grid_pulse"
	LiftWhir       = "lift_whir"
)

// Stereo 16-bit PCM sample size used by ebiten InfiniteLoop.
const pcmBytesPerSample = 4

const defaultPowerGridPulsePeriodMs = 1650.0

type loopClip struct {
	pcm      []byte
	player   *audio.Player
	active   bool
	epochMs  int64
	periodMs float64
	gain     float64
}

var loops map[string]*loopClip

func registerLoop(name string, pcm []byte, gain float64) {
	if loops == nil {
		loops = make(map[string]*loopClip)
	}
	periodMs := 1000.0 * float64(len(pcm)) / float64(sampleRate*pcmBytesPerSample)
	if periodMs < 100 {
		periodMs = 100
	}
	if gain <= 0 {
		gain = 1
	}
	loops[name] = &loopClip{pcm: pcm, periodMs: periodMs, gain: gain}
}

func closeLoopPlayerLocked(lp *loopClip) {
	if lp == nil || lp.player == nil {
		return
	}
	lp.player.Pause()
	_ = lp.player.Close()
	lp.player = nil
}

func ensureLoopPlayerLocked(name string, lp *loopClip) {
	if lp == nil || len(lp.pcm) == 0 || ctx == nil || lp.player != nil {
		return
	}
	loop := audio.NewInfiniteLoop(bytes.NewReader(lp.pcm), int64(len(lp.pcm)))
	player, err := ctx.NewPlayer(loop)
	if err != nil {
		log.Printf("audio: loop %s: %v", name, err)
		return
	}
	lp.player = player
	lp.player.SetVolume(effectiveVolume() * lp.gain)
	lp.player.Play()
}

func applyLoopVolumeLocked() {
	vol := effectiveVolume()
	for name, lp := range loops {
		if lp == nil || !lp.active {
			continue
		}
		if vol <= 0 {
			if lp.player != nil {
				lp.player.SetVolume(0)
			}
			continue
		}
		ensureLoopPlayerLocked(name, lp)
		if lp.player != nil {
			lp.player.SetVolume(vol * lp.gain)
		}
	}
}

// SetPowerGridPulseActive starts or stops the looping power-grid pulse SFX.
// Visual pulse timing uses the same epoch/period while the overlay is active.
func SetPowerGridPulseActive(on bool) {
	setLoopActive(PowerGridPulse, on)
}

// SetLiftWhirActive starts or stops the looping lift mechanical whir SFX.
func SetLiftWhirActive(on bool) {
	setLoopActive(LiftWhir, on)
}

func setLoopActive(name string, on bool) {
	mu.Lock()
	defer mu.Unlock()
	if !ready {
		return
	}
	lp := loops[name]
	if lp == nil {
		return
	}
	if on {
		if !lp.active {
			lp.active = true
			lp.epochMs = time.Now().UnixMilli()
		}
		if effectiveVolume() > 0 {
			ensureLoopPlayerLocked(name, lp)
		}
		return
	}
	closeLoopPlayerLocked(lp)
	lp.active = false
	lp.epochMs = 0
}

// PowerGridPulsePeriodMs returns the loop period used for visual sync.
func PowerGridPulsePeriodMs() float64 {
	mu.Lock()
	defer mu.Unlock()
	if lp := loops[PowerGridPulse]; lp != nil && lp.periodMs > 0 {
		return lp.periodMs
	}
	return defaultPowerGridPulsePeriodMs
}

// PowerGridPulseEpochMs is when the current pulse loop started (0 if inactive).
func PowerGridPulseEpochMs() int64 {
	mu.Lock()
	defer mu.Unlock()
	if lp := loops[PowerGridPulse]; lp != nil {
		return lp.epochMs
	}
	return 0
}

// PowerGridPulsePhase returns 0..1 progress through the current pulse cycle.
func PowerGridPulsePhase(nowMs int64) float64 {
	mu.Lock()
	defer mu.Unlock()
	period := defaultPowerGridPulsePeriodMs
	epoch := int64(0)
	if lp := loops[PowerGridPulse]; lp != nil {
		if lp.periodMs > 0 {
			period = lp.periodMs
		}
		epoch = lp.epochMs
	}
	elapsed := nowMs
	if epoch > 0 {
		elapsed = nowMs - epoch
	}
	if elapsed < 0 {
		elapsed = 0
	}
	periodMs := int64(period)
	if periodMs < 1 {
		periodMs = 1
	}
	return float64(elapsed%periodMs) / float64(periodMs)
}
