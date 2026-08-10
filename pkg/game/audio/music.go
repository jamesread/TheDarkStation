package audio

import (
	"bytes"
	"io"
	"log"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"

	"darkstation/pkg/resources"
)

// Named music tracks.
const (
	TrackTitleAmbient = "title_ambient"
)

const defaultMusicCrossfade = 1500 * time.Millisecond

type musicSlot struct {
	player *audio.Player
	track  string
}

var (
	musicTracks map[string][]byte
	musicReady  bool

	musicActive   *musicSlot
	musicOutgoing *musicSlot
	musicFading   bool
	musicFadeAt   time.Duration
	musicFadeDur  = defaultMusicCrossfade

	musicVolumePercent = 70
	currentMusicTrack  string
	lastMusicUpdate    time.Time
)

func initMusicTracks() {
	musicTracks = map[string][]byte{
		TrackTitleAmbient: resources.TitleAmbientMP3,
	}
	for name, data := range musicTracks {
		if len(data) == 0 {
			log.Printf("audio: music track %s is empty", name)
		}
	}
	musicReady = true
}

func newLoopingMusicPlayer(data []byte) (*audio.Player, error) {
	stream, err := mp3.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	loop := audio.NewInfiniteLoop(stream, stream.Length())
	return ctx.NewPlayer(loop)
}

func closeMusicSlot(slot *musicSlot) {
	if slot == nil || slot.player == nil {
		return
	}
	slot.player.Pause()
	_ = slot.player.Close()
	slot.player = nil
}

func effectiveMusicVolume() float64 {
	if muted {
		return 0
	}
	return float64(musicVolumePercent) / 100
}

func applyMusicVolumeLocked() {
	base := effectiveMusicVolume()
	if musicFading && musicFadeDur > 0 {
		t := float64(musicFadeAt) / float64(musicFadeDur)
		if t > 1 {
			t = 1
		}
		t = smoothstep(t)
		if musicOutgoing != nil && musicOutgoing.player != nil {
			musicOutgoing.player.SetVolume(base * (1 - t))
		}
		if musicActive != nil && musicActive.player != nil {
			musicActive.player.SetVolume(base * t)
		}
		return
	}
	if musicActive != nil && musicActive.player != nil {
		musicActive.player.SetVolume(base)
	}
	if musicOutgoing != nil && musicOutgoing.player != nil {
		musicOutgoing.player.SetVolume(0)
	}
}

func smoothstep(t float64) float64 {
	t = math.Max(0, math.Min(1, t))
	return t * t * (3 - 2*t)
}

// MusicVolumePercent returns the configured music volume (0–100), ignoring mute.
func MusicVolumePercent() int {
	mu.Lock()
	defer mu.Unlock()
	return musicVolumePercent
}

// SetMusicVolumePercent sets music volume (0–100) and updates active players.
func SetMusicVolumePercent(percent int) {
	mu.Lock()
	defer mu.Unlock()
	musicVolumePercent = clampPercent(percent)
	applyMusicVolumeLocked()
}

// CurrentMusicTrack returns the track name currently targeted for playback.
func CurrentMusicTrack() string {
	mu.Lock()
	defer mu.Unlock()
	return currentMusicTrack
}

// PlayMusic starts or crossfades to the named looping track. No-op if audio is
// not ready, the track is unknown, or the same track is already current.
func PlayMusic(name string) {
	mu.Lock()
	defer mu.Unlock()
	if !ready || !musicReady || ctx == nil {
		return
	}
	if name == "" || name == currentMusicTrack {
		return
	}
	data, ok := musicTracks[name]
	if !ok || len(data) == 0 {
		log.Printf("audio: unknown music track %q", name)
		return
	}

	player, err := newLoopingMusicPlayer(data)
	if err != nil {
		log.Printf("audio: music decode %s: %v", name, err)
		return
	}

	incoming := &musicSlot{player: player, track: name}
	currentMusicTrack = name

	if musicActive == nil || musicActive.player == nil {
		closeMusicSlot(musicOutgoing)
		musicOutgoing = nil
		musicActive = incoming
		musicFading = false
		musicFadeAt = 0
		musicActive.player.SetVolume(effectiveMusicVolume())
		musicActive.player.Play()
		return
	}

	// Crossfade: previous active becomes outgoing.
	closeMusicSlot(musicOutgoing)
	musicOutgoing = musicActive
	musicActive = incoming
	musicFading = true
	musicFadeAt = 0
	musicActive.player.SetVolume(0)
	musicActive.player.Play()
	applyMusicVolumeLocked()
}

// StopMusic fades out and stops the current track.
func StopMusic() {
	mu.Lock()
	defer mu.Unlock()
	if musicActive == nil && musicOutgoing == nil {
		currentMusicTrack = ""
		return
	}
	closeMusicSlot(musicOutgoing)
	musicOutgoing = musicActive
	musicActive = nil
	currentMusicTrack = ""
	if musicOutgoing == nil {
		musicFading = false
		musicFadeAt = 0
		return
	}
	musicFading = true
	musicFadeAt = 0
	applyMusicVolumeLocked()
}

// UpdateMusic advances crossfades. Call once per frame from the game Update loop.
func UpdateMusic() {
	now := time.Now()
	mu.Lock()
	defer mu.Unlock()
	dt := now.Sub(lastMusicUpdate)
	if lastMusicUpdate.IsZero() || dt < 0 || dt > time.Second {
		dt = time.Second / 60
	}
	lastMusicUpdate = now
	if !musicFading {
		return
	}
	musicFadeAt += dt
	if musicFadeAt >= musicFadeDur {
		closeMusicSlot(musicOutgoing)
		musicOutgoing = nil
		musicFading = false
		musicFadeAt = 0
		if musicActive != nil && musicActive.player != nil {
			musicActive.player.SetVolume(effectiveMusicVolume())
		}
		return
	}
	applyMusicVolumeLocked()
}

// SetMusicCrossfadeDuration overrides the default crossfade length (tests / tuning).
func SetMusicCrossfadeDuration(d time.Duration) {
	mu.Lock()
	defer mu.Unlock()
	if d < 50*time.Millisecond {
		d = 50 * time.Millisecond
	}
	musicFadeDur = d
}

// decodeMusicMP3 is used by tests to verify embedded tracks decode.
func decodeMusicMP3(data []byte) (int64, error) {
	stream, err := mp3.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(io.Discard, io.LimitReader(stream, 64*1024))
	if err != nil {
		return stream.Length(), err
	}
	if n == 0 {
		return stream.Length(), io.ErrUnexpectedEOF
	}
	return stream.Length(), nil
}
