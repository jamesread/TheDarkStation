package terminal

import (
	"os"

	"golang.org/x/term"
)

const (
	DefaultWidth  = 80
	DefaultHeight = 24
)

// GetSize returns the current terminal width and height.
// Falls back to defaults if the size cannot be determined.
func GetSize() (width, height int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return DefaultWidth, DefaultHeight
	}
	return width, height
}

// GetWidth returns the current terminal width.
// Falls back to DefaultWidth if the width cannot be determined.
func GetWidth() int {
	width, _ := GetSize()
	return width
}

// GetHeight returns the current terminal height.
// Falls back to DefaultHeight if the height cannot be determined.
func GetHeight() int {
	_, height := GetSize()
	return height
}
