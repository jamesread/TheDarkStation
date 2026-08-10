package audio

// Named USE / interact SFX clips.
const (
	UseGenerator = "use_generator"
	UseDoor      = "use_door"
	UseRepair    = "use_repair"
	UseCoupler   = "use_coupler"
	UseFurniture = "use_furniture"
	UseTerminal  = "use_terminal"
	UseMaint     = "use_maint"
	UseRelay     = "use_relay"
	UseHazard    = "use_hazard"
	UseLift      = "use_lift"
	UseDeny      = "use_deny"
	UsePickup    = "use_pickup"
	UseBattery   = "use_battery"

	SignalLockUp    = "signal_lock_up"
	SignalLockDown  = "signal_lock_down"
	SignalLockReady = "signal_lock_ready"
	HazardClear     = "hazard_clear"
)

// PlayUseGenerator plays generator inspect / power-up feedback.
func PlayUseGenerator() { Play(UseGenerator) }

// PlayUseDoor plays manual egress door release feedback.
func PlayUseDoor() { Play(UseDoor) }

// PlayUseRepair plays repair-device feedback.
func PlayUseRepair() { Play(UseRepair) }

// PlayUseCoupler plays a coupler crank pump tick.
func PlayUseCoupler() { Play(UseCoupler) }

// PlayUseFurniture plays furniture search feedback.
func PlayUseFurniture() { Play(UseFurniture) }

// PlayUseTerminal plays CCTV / puzzle terminal feedback.
func PlayUseTerminal() { Play(UseTerminal) }

// PlayUseMaint plays maintenance terminal open feedback.
func PlayUseMaint() { Play(UseMaint) }

// PlayUseRelay plays power relay toggle feedback.
func PlayUseRelay() { Play(UseRelay) }

// PlayUseHazard plays hazard control / clear feedback.
func PlayUseHazard() { Play(UseHazard) }

// PlayUseLift plays lift panel feedback.
func PlayUseLift() { Play(UseLift) }

// PlayUseDeny plays unpowered / blocked interaction feedback.
func PlayUseDeny() { Play(UseDeny) }

// PlayUsePickup plays floor item pickup feedback.
func PlayUsePickup() { Play(UsePickup) }

// PlayUseBattery plays battery insert into generator feedback.
func PlayUseBattery() { Play(UseBattery) }

// PlaySignalLockUp plays positive reinforcement when routing signal lock improves.
func PlaySignalLockUp() { Play(SignalLockUp) }

// PlaySignalLockDown plays negative reinforcement when routing signal lock worsens.
func PlaySignalLockDown() { Play(SignalLockDown) }

// PlaySignalLockReady plays when signal lock first reaches the commit threshold.
func PlaySignalLockReady() { Play(SignalLockReady) }

// PlayHazardClear plays as an environmental hazard fades out after being solved.
func PlayHazardClear() {
	setClipGain(HazardClear, 1)
	Play(HazardClear)
}

// SetHazardClearFadeProgress synchronizes the sound's fade with the visual
// clearing animation. Progress is clamped to 0..1.
func SetHazardClearFadeProgress(progress float64) {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	setClipGain(HazardClear, 1-progress)
}
