package audio

import (
	"testing"
	"time"

	"darkstation/pkg/resources"
)

func TestDecodeEmbeddedTitleAmbient(t *testing.T) {
	if len(resources.TitleAmbientMP3) == 0 {
		t.Fatal("TitleAmbientMP3 is empty")
	}
	length, err := decodeMusicMP3(resources.TitleAmbientMP3)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if length <= 0 {
		t.Fatalf("decoded length = %d, want > 0", length)
	}
}

func TestPlayMusicBeforeInitIsNoOp(t *testing.T) {
	if ready {
		t.Skip("audio already initialized; cannot assert pre-init no-op")
	}
	PlayMusic(TrackTitleAmbient)
	if CurrentMusicTrack() != "" {
		t.Fatalf("CurrentMusicTrack = %q before init, want empty", CurrentMusicTrack())
	}
	StopMusic()
	UpdateMusic()
}

func TestMusicVolumeAccessor(t *testing.T) {
	SetMusicVolumePercent(55)
	if MusicVolumePercent() != 55 {
		t.Fatalf("MusicVolumePercent = %d, want 55", MusicVolumePercent())
	}
	SetMusicVolumePercent(200)
	if MusicVolumePercent() != 100 {
		t.Fatalf("MusicVolumePercent = %d, want 100 after clamp", MusicVolumePercent())
	}
}

func TestSmoothstep(t *testing.T) {
	if smoothstep(0) != 0 || smoothstep(1) != 1 {
		t.Fatalf("smoothstep endpoints wrong: %v %v", smoothstep(0), smoothstep(1))
	}
	mid := smoothstep(0.5)
	if mid < 0.4 || mid > 0.6 {
		t.Fatalf("smoothstep(0.5) = %v, want ~0.5", mid)
	}
}

func TestSetMusicCrossfadeDuration(t *testing.T) {
	SetMusicCrossfadeDuration(time.Second)
	mu.Lock()
	got := musicFadeDur
	mu.Unlock()
	if got != time.Second {
		t.Fatalf("musicFadeDur = %v, want 1s", got)
	}
	SetMusicCrossfadeDuration(time.Millisecond) // below minimum
	mu.Lock()
	got = musicFadeDur
	mu.Unlock()
	if got != 50*time.Millisecond {
		t.Fatalf("musicFadeDur = %v, want 50ms minimum", got)
	}
	SetMusicCrossfadeDuration(defaultMusicCrossfade)
}
