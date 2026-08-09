// Package audio plays short UI and game sound effects via Ebiten's audio API.
package audio

import (
	"bytes"
	"io"
	"log"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"

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
}

var (
	mu      sync.Mutex
	ctx     *audio.Context
	clips   map[string]*clip
	ready   bool
	initErr error
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

	ctx = audio.CurrentContext()
	if ctx == nil {
		ctx = audio.NewContext(sampleRate)
	}

	clips = make(map[string]*clip)
	sources := map[string][]byte{
		MenuConfirm: resources.MenuConfirmMP3,
		MenuUp:      resources.MenuUpMP3,
		MenuDown:    resources.MenuDownMP3,
	}
	for name, data := range sources {
		pcm, err := decodeMP3(data)
		if err != nil {
			initErr = err
			log.Printf("audio: decode %s: %v", name, err)
			return
		}
		clips[name] = &clip{pcm: pcm}
	}
	ready = true
}

func decodeMP3(data []byte) ([]byte, error) {
	stream, err := mp3.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(stream)
}

// Play starts (or restarts) a named clip. No-op if audio is not initialized.
func Play(name string) {
	mu.Lock()
	defer mu.Unlock()
	if !ready {
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
	_ = c.player.Rewind()
	c.player.Play()
}

// PlayMenuConfirm plays the menu selection/confirm beep.
func PlayMenuConfirm() { Play(MenuConfirm) }

// PlayMenuUp plays the menu navigate-up tick.
func PlayMenuUp() { Play(MenuUp) }

// PlayMenuDown plays the menu navigate-down tick.
func PlayMenuDown() { Play(MenuDown) }
