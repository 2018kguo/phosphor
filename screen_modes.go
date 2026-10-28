package phosphor

import "strconv"

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
