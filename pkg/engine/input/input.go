package input

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/term"
)

var stdinReader *bufio.Reader

// GetInput reads a line of input from stdin
func GetInput() string {
	if stdinReader == nil {
		stdinReader = bufio.NewReader(os.Stdin)
	}

	chr, err := stdinReader.ReadString('\n')

	if err != nil {
		log.Fatalf("Cannot read stdin: %v", err)
		return ""
	}

	return strings.Trim(chr, "\n")
}

// readByte reads a single byte from stdin in raw mode
func readByte() (byte, error) {
	buf := make([]byte, 1)
	_, err := os.Stdin.Read(buf)
	return buf[0], err
}

// tryReadArrowKey attempts to read an arrow key escape sequence.
// Returns the arrow direction string if successful, empty string otherwise.
func tryReadArrowKey(firstByte byte) (string, []byte) {
	if firstByte != 0x1b {
		return "", []byte{firstByte}
	}

	// Read second byte
	b2, err := readByte()
	if err != nil {
		return "", nil
	}

	// Handle both CSI sequences (ESC [) and SS3 sequences (ESC O)
	if b2 == '[' || b2 == 'O' {
		// Read third byte (the actual key code)
		b3, err := readByte()
		if err != nil {
			return "", nil
		}

		switch b3 {
		case 'A':
			return "arrow_up", nil
		case 'B':
			return "arrow_down", nil
		case 'C':
			return "arrow_right", nil
		case 'D':
			return "arrow_left", nil
		}
		// Unknown escape sequence - discard it
		return "", nil
	}

	// Not an arrow sequence, return the bytes we read
	return "", []byte{firstByte, b2}
}

// GetInputWithArrows reads input with support for arrow keys.
// Arrow keys return immediately without needing Enter.
// For text input, user types and presses Enter as normal.
func GetInputWithArrows() string {
	// Reset the buffered reader to avoid conflicts with raw mode
	stdinReader = nil

	// Put terminal into raw mode to detect arrow keys
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatalf("Cannot set terminal to raw mode: %v", err)
	}

	// Read first byte
	b1, err := readByte()
	if err != nil {
		term.Restore(int(os.Stdin.Fd()), oldState)
		log.Fatalf("Cannot read stdin: %v", err)
		return ""
	}

	// Check for arrow key
	if arrowKey, _ := tryReadArrowKey(b1); arrowKey != "" {
		term.Restore(int(os.Stdin.Fd()), oldState)
		fmt.Println() // Move to next line
		return arrowKey
	}

	// Handle Ctrl+C
	if b1 == 3 {
		term.Restore(int(os.Stdin.Fd()), oldState)
		fmt.Println()
		os.Exit(0)
	}

	// Handle newline/enter - just return empty
	if b1 == '\n' || b1 == '\r' {
		term.Restore(int(os.Stdin.Fd()), oldState)
		return ""
	}

	// For regular characters, collect input until Enter
	var input []byte
	// Only add printable characters
	if b1 >= 32 && b1 < 127 {
		input = append(input, b1)
		fmt.Print(string(b1)) // Echo the character
	}

	for {
		b, err := readByte()
		if err != nil {
			break
		}

		// Check for escape sequence (arrow keys pressed during text entry)
		if b == 0x1b {
			// Try to read as arrow key, discard if it is one
			tryReadArrowKey(b)
			continue
		}

		// Handle backspace
		if b == 127 || b == 8 {
			if len(input) > 0 {
				input = input[:len(input)-1]
				fmt.Print("\b \b") // Erase character from display
			}
			continue
		}

		// Handle Enter
		if b == '\n' || b == '\r' {
			fmt.Println()
			break
		}

		// Handle Ctrl+C
		if b == 3 {
			term.Restore(int(os.Stdin.Fd()), oldState)
			fmt.Println()
			os.Exit(0)
		}

		// Only add printable characters
		if b >= 32 && b < 127 {
			input = append(input, b)
			fmt.Print(string(b)) // Echo the character
		}
	}

	term.Restore(int(os.Stdin.Fd()), oldState)
	return string(input)
}
