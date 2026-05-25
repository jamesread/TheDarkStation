package ebiten

import "strings"

func cvarEnabled(name string) bool {
	cvarMutex.RLock()
	v := strings.ToLower(strings.TrimSpace(cvarMap[name]))
	cvarMutex.RUnlock()
	return v == "1" || v == "true" || v == "yes"
}

func setCvarBool(name string, on bool) {
	if on {
		setCvar(name, "1")
	} else {
		setCvar(name, "0")
	}
}

func toggleCvarBool(name string) bool {
	on := !cvarEnabled(name)
	setCvarBool(name, on)
	return on
}
