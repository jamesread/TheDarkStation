package resources

import (
	_ "embed"
)

//go:embed sounds/menu_confirm.mp3
var MenuConfirmMP3 []byte

//go:embed sounds/menu_up.mp3
var MenuUpMP3 []byte

//go:embed sounds/menu_down.mp3
var MenuDownMP3 []byte
