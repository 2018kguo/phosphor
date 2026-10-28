package phosphor

import (
	"strconv"
)

// Configure destructive mode changes before painting. Navigation prefers a
// mode whose final bit already matches the destination, so screen switches
// do not silently change the independently reported mode bits.
func (t *Terminal) appendScreenModes(b []byte, saved []savedMode) ([]byte, int, int) {
	for _, m := range saved {
		if screenModeBit(m.mode) == 0 {
			continue
		}
		b = appendMode(b, m.mode, m.on)
		b = append(b, "\x1b[?"...)
		b = strconv.AppendInt(b, int64(m.mode), 10)
		b = append(b, 's')
	}
	entry, exit := 0, 0
	for _, mode := range [...]int{47, 1047, 1049} {
		on, _ := t.modeValue(mode)
		b = appendMode(b, mode, on)
		if on && entry == 0 {
			entry = mode
		}
	}
	// Prefer a non-erasing exit: the active cursor is restored later.
	for _, mode := range [...]int{47, 1049, 1047} {
		if on, _ := t.modeValue(mode); !on {
			exit = mode
			break
		}
	}
	if entry == 0 {
		entry = 47
	}
	if exit == 0 {
		exit = 47
	}
	return appendMode(b, exit, false), entry, exit
}

// A 1047 exit recreates the background of its erased cells. Protected
// cells are kept by ISO protection; a later host resize can introduce a
// different background, which portable mode switching cannot reproduce.
func (t *Terminal) alternateEraseBackground() Color {
	for _, row := range t.alt {
		for x := range t.cols {
			c := rowCell(row, x)
			if !t.protISO || c.A&AttrProtected == 0 {
				return c.Bg
			}
		}
	}
	return 0
}

const (
	privateMode132 uint8 = 1 << iota
	privateModeAllowColumns
	privateModeSlowScroll
	privateMode47
	privateMode1047
	privateMode1049
	privateModeCount = iota
)

const privateModeMask uint8 = 1<<privateModeCount - 1

var savedModeNumbers = [...]int{1, 3, 4, 5, 6, 7, 9, 12, 25, 40, 45, 47, 69, 1000, 1002, 1003, 1004, 1005, 1006, 1015, 1016, 1045, 1047, 1049, 2004, 2027}

func screenModeBit(mode int) uint8 {
	switch mode {
	case 47:
		return privateMode47
	case 1047:
		return privateMode1047
	case 1049:
		return privateMode1049
	}
	return 0
}

func (t *Terminal) privateModes() uint8 {
	if t.x != nil {
		return t.x.privateModes
	}
	return 0
}

func (t *Terminal) setColumnMode(on bool) {
	if t.privateModes()&privateModeAllowColumns == 0 {
		if t.x != nil {
			t.x.privateModes &^= privateMode132
		}
		return
	}
	cols := 80
	if on {
		t.x.privateModes |= privateMode132
		cols = 132
	} else {
		t.x.privateModes &^= privateMode132
	}
	t.Resize(cols, t.rows)
	// A same-size resize preserves margins and tabs. A width change resets
	// them through Resize, matching the native terminal's DECCOLM behavior.
	t.eraseDisplay(2, false)
	t.moveTo(0, 0)
}

// Recreate column modes and their saved values before painting: setting
// mode3 can resize and erase the receiving screen. A selected 132-column
// mode after a host resize to an arbitrary width cannot be represented by
// portable VT alone; Snapshot preserves that state exactly.
func (t *Terminal) appendColumnModes(b []byte, saved []savedMode) []byte {
	current132 := false
	for _, mode := range saved {
		if mode.mode != 3 && mode.mode != 40 {
			continue
		}
		if mode.mode == 3 && mode.on != current132 {
			b = appendMode(b, 40, true)
			b = appendMode(b, 3, mode.on)
			current132 = mode.on
		} else if mode.mode == 40 {
			b = appendMode(b, 40, mode.on)
		}
		b = append(b, "\x1b[?"...)
		if mode.mode == 3 {
			b = append(b, '3')
		} else {
			b = append(b, '4', '0')
		}
		b = append(b, 's')
	}
	if on := t.privateModes()&privateMode132 != 0; on != current132 {
		b = appendMode(b, 40, true)
		b = appendMode(b, 3, on)
	}
	b = appendMode(b, 40, t.privateModes()&privateModeAllowColumns != 0)
	// DECCOLM's resize ends a synchronized update, even at the same size.
	return append(b, "\x1b[?2026h"...)
}

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
