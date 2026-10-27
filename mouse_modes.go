package phosphor

// The low four bits preserve DECRQM/XTSAVE state. The next three bits
// preserve the effective mode, which changes on every set or reset even
// when other mode bits remain enabled. This keeps the idle Terminal small.
const mouseModeBits uint16 = 15

var mouseTrackingNumbers = [4]int{9, 1000, 1002, 1003}
var mouseEncodingNumbers = [4]int{1005, 1006, 1015, 1016}

func mouseModeBit(mode int, numbers [4]int) uint16 {
	for i, n := range numbers {
		if mode == n {
			return 1 << i
		}
	}
	return 0
}

func mouseModeNumber(state uint16, numbers [4]int) int {
	if selected := state >> 4; selected >= 1 && selected <= 4 {
		return numbers[selected-1]
	}
	return 0
}

func setMouseMode(state uint16, mode int, on bool, numbers [4]int) uint16 {
	for i, n := range numbers {
		if mode != n {
			continue
		}
		bits := state & mouseModeBits
		if on {
			return bits | 1<<i | uint16(i+1)<<4
		}
		return bits &^ (1 << i)
	}
	return state
}

// validMouseModes rejects states no VT sequence can produce: an effective
// mode must be enabled, and all four enabled bits imply an effective mode.
func validMouseModes(state uint16) bool {
	selected := state >> 4
	if selected == 0 {
		return state != mouseModeBits
	}
	return selected <= 4 && state&(1<<(selected-1)) != 0
}

func appendMouseModes(b []byte, state uint16, numbers [4]int) []byte {
	selected := mouseModeNumber(state, numbers)
	for i, mode := range numbers {
		if state&(1<<i) != 0 && mode != selected {
			b = appendMode(b, mode, true)
		}
	}
	if selected != 0 {
		return appendMode(b, selected, true)
	}
	if state != 0 {
		// Resetting any disabled mode clears the effective selection while
		// preserving every enabled bit. A valid state always has one.
		for i, mode := range numbers {
			if state&(1<<i) == 0 {
				return appendMode(b, mode, false)
			}
		}
	}
	return b
}
