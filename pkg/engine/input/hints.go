package input

import "fmt"

// HintMove returns the movement control hint for the active primary device.
func HintMove() string {
	if GetPrimaryDevice() == PrimaryGamepad {
		return "Use left stick or D-pad to move"
	}
	return "Press WASD or arrow keys to move"
}

// HintInteractPrefix returns "Press … to interact" for callouts and tooltips.
func HintInteractPrefix() string {
	if GetPrimaryDevice() == PrimaryGamepad {
		return "Press A to interact"
	}
	return "Press E or Enter to interact"
}

// HintMenuSelect returns navigation text for menus (without trailing period).
func HintMenuSelect() string {
	if GetPrimaryDevice() == PrimaryGamepad {
		return "D-pad up/down to select"
	}
	return "Use up/down to select"
}

// HintMenuActivate returns activate/confirm text for menus.
func HintMenuActivate() string {
	if GetPrimaryDevice() == PrimaryGamepad {
		return "A to activate"
	}
	return "Enter to activate"
}

// HintMenuEditBinding returns edit binding text for the bindings menu.
func HintMenuEditBinding() string {
	if GetPrimaryDevice() == PrimaryGamepad {
		return "A to edit"
	}
	return "Enter to edit"
}

// HintMenuBackToMain returns back navigation for bindings from the title menu.
func HintMenuBackToMain() string {
	if GetPrimaryDevice() == PrimaryGamepad {
		return "A to return to the main menu"
	}
	return "Press Enter to return to the main menu"
}

// HintMenuQuit returns quit/exit shortcut text (main menu).
func HintMenuQuit() string {
	if GetPrimaryDevice() == PrimaryGamepad {
		return "B to quit"
	}
	return "Esc to quit"
}

// HintMenuClose returns close-menu shortcut text (in-game menus).
func HintMenuClose() string {
	if GetPrimaryDevice() == PrimaryGamepad {
		return "Start or B to close"
	}
	return "F10/Start or q to close"
}

// HintMenuCloseShort returns a shorter close hint (Escape variant).
func HintMenuCloseShort() string {
	if GetPrimaryDevice() == PrimaryGamepad {
		return "B or Start to close"
	}
	return "Escape or Menu to close"
}

// HintConfirm returns confirm/OK text.
func HintConfirm() string {
	if GetPrimaryDevice() == PrimaryGamepad {
		return "A"
	}
	return "Enter"
}

// HintMenuInstructionsMain returns main menu footer instructions.
func HintMenuInstructionsMain() string {
	return fmt.Sprintf("%s, %s, %s", HintMenuSelect(), HintMenuActivate(), HintMenuQuit())
}

// HintMenuInstructionsGameplay returns gameplay pause menu instructions.
func HintMenuInstructionsGameplay() string {
	return fmt.Sprintf("%s, %s, %s", HintMenuSelect(), HintMenuActivate(), HintMenuClose())
}

// HintBindingsExit returns exit/back hints for the bindings menu footer.
func HintBindingsExit(fromMainMenu bool) string {
	if fromMainMenu {
		return fmt.Sprintf("%s, %s", HintMenuBackToMain(), HintMenuClose())
	}
	return HintMenuClose()
}

// HintPressConfirm returns "Press …" phrasing for menu help lines.
func HintPressConfirm() string {
	if GetPrimaryDevice() == PrimaryGamepad {
		return "Press A"
	}
	return "Press Enter"
}

// HintPressConfirmTo returns confirm phrasing plus an action (e.g. "Press A to toggle power").
func HintPressConfirmTo(action string) string {
	return HintPressConfirm() + " to " + action
}

// HintPressConfirmOrTab returns panel switch help for maintenance mode toggle.
func HintPressConfirmOrTab() string {
	if GetPrimaryDevice() == PrimaryGamepad {
		return "Press A to switch panel"
	}
	return "Press Enter or Tab to switch panel"
}

// HintMaintMenuInstructions returns maintenance terminal menu footer text.
func HintMaintMenuInstructions() string {
	if GetPrimaryDevice() == PrimaryGamepad {
		return "D-pad up/down: select | A: activate | D-pad left/right: cycle | B: close"
	}
	return "Up/Down: select | Enter: activate | A/D: cycle option | 1/2/3: circuit preset | Tab: mode | Esc: close"
}

// HintMaintCycle returns inline cycle hint for maintenance labels.
func HintMaintCycle() string {
	if GetPrimaryDevice() == PrimaryGamepad {
		return "< left/right >"
	}
	return "< A/D >"
}

// HintDevMenuInstructions returns developer menu instructions.
func HintDevMenuInstructions() string {
	if GetPrimaryDevice() == PrimaryGamepad {
		return "D-pad up/down: select | A: activate | B: close"
	}
	return "Up/Down: select | Enter: activate | q or Esc: close"
}

// IsMovementHintMessage reports whether a callout uses the movement tutorial text.
func IsMovementHintMessage(msg string) bool {
	return msg == "Press WASD or arrow keys to move" || msg == "Use left stick or D-pad to move"
}

// IsInteractHintMessage reports whether a callout uses the interact tutorial text.
func IsInteractHintMessage(msg string) bool {
	return msg == "Press E or Enter to interact" || msg == "Press A to interact"
}

// HintConfirmYesNo returns yes/no labels for confirmation dialogs.
func HintConfirmYesNo() string {
	if GetPrimaryDevice() == PrimaryGamepad {
		return "A: Yes | B: No"
	}
	return "Y or Enter: Yes | N or Esc: No"
}
