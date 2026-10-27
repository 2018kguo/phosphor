package phosphor

// Compact releases unused storage after a terminal becomes idle. It drops
// rows containing only zero cells, unused wrap flags and temporary buffers.
// It keeps screen contents, scrollback, cursors, modes, damage and unfinished
// parser input unchanged. Later output may allocate those buffers again.
//
// Like Write and Resize, Compact requires exclusive access to the Terminal.
// The caller decides when to run it; output never compacts itself.
func (t *Terminal) Compact() {
	for _, screen := range [][][]cell{t.main, t.alt} {
		for y, row := range screen {
			if compactZeroRow(row) {
				screen[y] = nil
			}
		}
	}
	if !t.anyWrap(t.mainWrap) {
		t.mainWrap = nil
	}
	if !t.anyWrap(t.altWrap) {
		t.altWrap = nil
	}
	t.scratch = nil
	if t.state < stOSC {
		t.osc = nil
	}
}

// Space characters, spacer heads and styled blanks carry information even
// when they look empty. Only literal zero cells can be replaced by a nil row.
func compactZeroRow(row []cell) bool {
	for _, c := range row {
		if c != (cell{}) {
			return false
		}
	}
	return true
}
