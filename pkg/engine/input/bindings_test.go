package input

import "testing"

func TestSetSingleBindingDeviceScoped(t *testing.T) {
	orig := make(map[string]Action, len(bindings))
	for k, v := range bindings {
		orig[k] = v
	}
	t.Cleanup(func() {
		for k := range bindings {
			delete(bindings, k)
		}
		for k, v := range orig {
			bindings[k] = v
		}
	})

	SetSingleBinding(ActionMoveNorth, "x")
	if bindings["x"] != ActionMoveNorth {
		t.Fatal("keyboard binding not set")
	}
	if bindings["gamepad_dpad_up"] != ActionMoveNorth {
		t.Fatal("gamepad binding should remain when rebinding keyboard")
	}
	if bindings["arrow_up"] != ActionMoveNorth {
		t.Fatal("reserved arrow binding should remain")
	}

	SetSingleBinding(ActionMoveNorth, "gamepad_y")
	if bindings["gamepad_y"] != ActionMoveNorth {
		t.Fatal("gamepad binding not set")
	}
	if bindings["gamepad_dpad_up"] == ActionMoveNorth {
		t.Fatal("previous gamepad binding should be replaced")
	}
	if bindings["x"] != ActionMoveNorth {
		t.Fatal("keyboard binding should remain when rebinding gamepad")
	}
}

func TestSetSingleBindingReservedCodes(t *testing.T) {
	orig := make(map[string]Action, len(bindings))
	for k, v := range bindings {
		orig[k] = v
	}
	t.Cleanup(func() {
		for k := range bindings {
			delete(bindings, k)
		}
		for k, v := range orig {
			bindings[k] = v
		}
	})

	SetSingleBinding(ActionMoveNorth, "gamepad_a")
	if bindings["gamepad_a"] != ActionInteract {
		t.Fatal("gamepad_a must stay reserved for interact")
	}
	SetSingleBinding(ActionHint, "arrow_up")
	if bindings["arrow_up"] != ActionMoveNorth {
		t.Fatal("arrow_up must stay reserved for movement")
	}
}

func TestFormatBindingCode(t *testing.T) {
	if FormatBindingCode("gamepad_b") != "B" {
		t.Fatalf("got %q", FormatBindingCode("gamepad_b"))
	}
	if FormatBindingCode("w") != "w" {
		t.Fatalf("got %q", FormatBindingCode("w"))
	}
}
