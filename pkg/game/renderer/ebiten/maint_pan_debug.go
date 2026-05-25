package ebiten

import (
	"log"
	"time"
)

func maintPanDebugOn() bool {
	return cvarEnabled("debug.maint_pan")
}

func maintPanLogf(format string, args ...any) {
	if maintPanDebugOn() {
		log.Printf("[MaintPan] "+format, args...)
	}
}

// maintPanLogfThrottled avoids flooding stderr when debug is on (Updates run at TPS).
var maintPanLogNextUnixMilli int64

func maintPanLogfThrottled(format string, args ...any) {
	if !maintPanDebugOn() {
		return
	}
	now := time.Now().UnixMilli()
	if now < maintPanLogNextUnixMilli {
		return
	}
	maintPanLogNextUnixMilli = now + 250
	log.Printf("[MaintPan] "+format, args...)
}
