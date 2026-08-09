package audio

import (
	"testing"

	"darkstation/pkg/resources"
)

func TestDecodeEmbeddedMenuSFX(t *testing.T) {
	sources := map[string][]byte{
		MenuConfirm: resources.MenuConfirmMP3,
		MenuUp:      resources.MenuUpMP3,
		MenuDown:    resources.MenuDownMP3,
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
}
