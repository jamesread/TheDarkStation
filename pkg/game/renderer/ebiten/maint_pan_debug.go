package ebiten

import (
	"log"
	"strings"
	"time"
)

func maintPanDebugOn() bool {
	cvarMutex.RLock()
	v := strings.ToLower(strings.TrimSpace(cvarMap["debug.maint_pan"]))
	cvarMutex.RUnlock()
	return v == "1" || v == "true" || v == "yes"
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
