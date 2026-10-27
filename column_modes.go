package phosphor

const (
	columnMode132 uint8 = 1 << iota
	columnModeAllow
)

func (t *Terminal) columnModes() uint8 {
	if t.x != nil {
		return t.x.columnModes
	}
	return 0
}

func (t *Terminal) setColumnMode(on bool) {
	if t.columnModes()&columnModeAllow == 0 {
		if t.x != nil {
			t.x.columnModes &^= columnMode132
		}
		return
	}
	cols := 80
	if on {
		t.x.columnModes |= columnMode132
		cols = 132
	} else {
		t.x.columnModes &^= columnMode132
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
	if on := t.columnModes()&columnMode132 != 0; on != current132 {
		b = appendMode(b, 40, true)
		b = appendMode(b, 3, on)
	}
	b = appendMode(b, 40, t.columnModes()&columnModeAllow != 0)
	// DECCOLM's resize ends a synchronized update, even at the same size.
	return append(b, "\x1b[?2026h"...)
}
