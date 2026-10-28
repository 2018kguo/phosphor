package phosphor

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
