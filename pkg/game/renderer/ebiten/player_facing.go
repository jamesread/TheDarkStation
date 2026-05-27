package ebiten

import (
	"math"
	"sync"
	"time"

	"darkstation/pkg/game/state"
)

const (
	IconPlayerArrow      = "↑" // Stemmed arrow; rotates smoothly and stays readable at a glance (unlike ▲)
	playerFacingRotateMs = 180
)

// playerFacingRotation tracks interpolated arrow rotation on the draw thread.
type playerFacingRotation struct {
	mu sync.Mutex

	initialized    bool
	angle          float64
	lastSnapFacing state.PlayerFacing
	animStartMs    int64
	animFromAngle  float64
	animToAngle    float64
}

func playerFacingAngle(facing state.PlayerFacing) float64 {
	switch facing {
	case state.FaceSouth:
		return math.Pi
	case state.FaceEast:
		return math.Pi / 2
	case state.FaceWest:
		return -math.Pi / 2
	default:
		return 0
	}
}

func lerpAngleShortest(from, to, t float64) float64 {
	delta := math.Mod(to-from+math.Pi, 2*math.Pi) - math.Pi
	return from + delta*t
}

func easeOutCubic(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	inv := 1 - t
	return 1 - inv*inv*inv
}

func (r *playerFacingRotation) drawAngle(snapFacing state.PlayerFacing) float64 {
	target := playerFacingAngle(snapFacing)
	now := time.Now().UnixMilli()

	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.initialized {
		r.initialized = true
		r.angle = target
		r.lastSnapFacing = snapFacing
		return r.angle
	}

	if snapFacing != r.lastSnapFacing {
		r.lastSnapFacing = snapFacing
		r.animFromAngle = r.angle
		r.animToAngle = target
		r.animStartMs = now
	}

	if r.animStartMs != 0 {
		elapsed := now - r.animStartMs
		if elapsed >= playerFacingRotateMs {
			r.angle = r.animToAngle
			r.animStartMs = 0
		} else {
			t := easeOutCubic(float64(elapsed) / float64(playerFacingRotateMs))
			r.angle = lerpAngleShortest(r.animFromAngle, r.animToAngle, t)
		}
	} else {
		r.angle = target
	}

	return r.angle
}
