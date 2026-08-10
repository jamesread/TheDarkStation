package menu

import (
	"fmt"

	"darkstation/pkg/engine/input"
	"darkstation/pkg/game/audio"
	"darkstation/pkg/game/config"
)

const sfxVolumeStep = 10

// AudioMenuHandler handles audio settings (standalone; also used via Settings Audio tab).
type AudioMenuHandler struct{}

// NewAudioMenuHandler creates a new audio settings menu handler.
func NewAudioMenuHandler() *AudioMenuHandler {
	return &AudioMenuHandler{}
}

func (h *AudioMenuHandler) GetTitle() string {
	return "Audio"
}

func (h *AudioMenuHandler) GetInstructions(selected MenuItem) string {
	return input.HintMenuSelect() + ", " + input.HintMenuActivate() + ", D-pad left/right or A/D to cycle, " + input.HintMenuClose() + "."
}

func (h *AudioMenuHandler) OnSelect(item MenuItem, index int) {}

func (h *AudioMenuHandler) OnActivate(item MenuItem, index int) (shouldClose bool, helpText string) {
	if _, isClose := item.(*CloseMenuItem); isClose {
		return true, ""
	}
	if cycler, ok := item.(CycleMenuItem); ok && cycler.CanCycle() {
		_, helpText := cycler.HandleCycle(1)
		return false, helpText
	}
	return false, ""
}

func (h *AudioMenuHandler) OnExit() {}

func (h *AudioMenuHandler) ShouldCloseOnAnyAction() bool {
	return false
}

func (h *AudioMenuHandler) GetMenuItems() []MenuItem {
	return []MenuItem{
		&SFXVolumeMenuItem{},
		&MusicVolumeMenuItem{},
		&MuteMenuItem{},
		&CloseMenuItem{Label: "Back"},
	}
}

// SFXVolumeMenuItem cycles SFX volume in 10% steps.
type SFXVolumeMenuItem struct{}

func (s *SFXVolumeMenuItem) GetLabel() string {
	return fmt.Sprintf("SFX Volume\tACTION{%d%%}\tSUBTLE{< left/right >}", audio.VolumePercent())
}

func (s *SFXVolumeMenuItem) IsSelectable() bool { return true }

func (s *SFXVolumeMenuItem) GetHelpText() string {
	return "Menu and sound effect volume"
}

func (s *SFXVolumeMenuItem) CanCycle() bool { return true }

func (s *SFXVolumeMenuItem) HandleCycle(delta int) (bool, string) {
	cur := audio.VolumePercent()
	next := cur + delta*sfxVolumeStep
	if next < 0 {
		next = 0
	}
	if next > 100 {
		next = 100
	}
	if next != cur {
		audio.SetVolumePercent(next)
		if err := config.Current().SetSFXVolume(next); err != nil {
			return true, fmt.Sprintf("SFX volume: %d%% (could not save)", next)
		}
	}
	if audio.Muted() && next > 0 {
		audio.SetMute(false)
		_ = config.Current().SetMute(false)
	}
	if next > 0 && !audio.Muted() {
		audio.PlayMenuConfirm()
	}
	return true, fmt.Sprintf("SFX volume: %d%%", next)
}

// MusicVolumeMenuItem cycles music volume in 10% steps.
type MusicVolumeMenuItem struct{}

func (m *MusicVolumeMenuItem) GetLabel() string {
	return fmt.Sprintf("Music Volume\tACTION{%d%%}\tSUBTLE{< left/right >}", audio.MusicVolumePercent())
}

func (m *MusicVolumeMenuItem) IsSelectable() bool { return true }

func (m *MusicVolumeMenuItem) GetHelpText() string {
	return "Background music volume"
}

func (m *MusicVolumeMenuItem) CanCycle() bool { return true }

func (m *MusicVolumeMenuItem) HandleCycle(delta int) (bool, string) {
	cur := audio.MusicVolumePercent()
	next := cur + delta*sfxVolumeStep
	if next < 0 {
		next = 0
	}
	if next > 100 {
		next = 100
	}
	if next != cur {
		audio.SetMusicVolumePercent(next)
		if err := config.Current().SetMusicVolume(next); err != nil {
			return true, fmt.Sprintf("Music volume: %d%% (could not save)", next)
		}
	}
	if audio.Muted() && next > 0 {
		audio.SetMute(false)
		_ = config.Current().SetMute(false)
	}
	return true, fmt.Sprintf("Music volume: %d%%", next)
}

// MuteMenuItem toggles audio mute.
type MuteMenuItem struct{}

func (m *MuteMenuItem) GetLabel() string {
	state := "off"
	if audio.Muted() {
		state = "on"
	}
	return "Mute\tACTION{" + state + "}\tSUBTLE{< left/right >}"
}

func (m *MuteMenuItem) IsSelectable() bool { return true }

func (m *MuteMenuItem) GetHelpText() string {
	return "Silence music and sound effects without changing volume levels"
}

func (m *MuteMenuItem) CanCycle() bool { return true }

func (m *MuteMenuItem) HandleCycle(delta int) (bool, string) {
	on := !audio.Muted()
	audio.SetMute(on)
	if err := config.Current().SetMute(on); err != nil {
		if on {
			return true, "Mute: on (could not save)"
		}
		return true, "Mute: off (could not save)"
	}
	if !on && audio.VolumePercent() > 0 {
		audio.PlayMenuConfirm()
	}
	if on {
		return true, "Mute: on"
	}
	return true, "Mute: off"
}
