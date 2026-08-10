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

//go:embed sounds/power_grid_pulse.mp3
var PowerGridPulseMP3 []byte

//go:embed sounds/lift_whir.mp3
var LiftWhirMP3 []byte

//go:embed sounds/use_generator.mp3
var UseGeneratorMP3 []byte

//go:embed sounds/use_door.mp3
var UseDoorMP3 []byte

//go:embed sounds/use_repair.mp3
var UseRepairMP3 []byte

//go:embed sounds/use_coupler.mp3
var UseCouplerMP3 []byte

//go:embed sounds/use_furniture.mp3
var UseFurnitureMP3 []byte

//go:embed sounds/use_terminal.mp3
var UseTerminalMP3 []byte

//go:embed sounds/use_maint.mp3
var UseMaintMP3 []byte

//go:embed sounds/use_relay.mp3
var UseRelayMP3 []byte

//go:embed sounds/use_hazard.mp3
var UseHazardMP3 []byte

//go:embed sounds/use_lift.mp3
var UseLiftMP3 []byte

//go:embed sounds/use_deny.mp3
var UseDenyMP3 []byte

//go:embed sounds/use_pickup.mp3
var UsePickupMP3 []byte

//go:embed sounds/use_battery.mp3
var UseBatteryMP3 []byte

//go:embed sounds/signal_lock_up.mp3
var SignalLockUpMP3 []byte

//go:embed sounds/signal_lock_down.mp3
var SignalLockDownMP3 []byte

//go:embed sounds/signal_lock_ready.mp3
var SignalLockReadyMP3 []byte

//go:embed sounds/hazard_clear.mp3
var HazardClearMP3 []byte
