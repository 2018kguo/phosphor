package phosphor

// dirtySet is a bit per screen row: whether its cells changed since the
// last ClearDirty.
type dirtySet struct{ bits []uint64 }

// touch marks row y dirty. Until the first ClearDirty there is nothing to
// record: every row counts as dirty.
func (t *Terminal) touch(y int) {
	if d := t.dirty; d != nil {
		d.bits[y>>6] |= 1 << (y & 63)
	}
}

// touchRows marks rows [from, to) dirty.
func (t *Terminal) touchRows(from, to int) {
	if d := t.dirty; d != nil && from < to {
		d.markRows(from, to)
	}
}

// Keep word-mask work out of the inline headless damage check.
//
//go:noinline
func (d *dirtySet) markRows(from, to int) {
	first, last := from>>6, (to-1)>>6
	start := ^uint64(0) << (from & 63)
	end := ^uint64(0) >> (63 - ((to - 1) & 63))
	if first == last {
		d.bits[first] |= start & end
		return
	}
	d.bits[first] |= start
	for i := first + 1; i < last; i++ {
		d.bits[i] = ^uint64(0)
	}
	d.bits[last] |= end
}

// touchAll marks every row dirty.
func (t *Terminal) touchAll() {
	if d := t.dirty; d != nil {
		for i := range d.bits {
			d.bits[i] = ^uint64(0)
		}
	}
}

// resizeDirty fits the dirty rows to a new height, all dirty.
func (t *Terminal) resizeDirty() {
	if t.dirty != nil {
		t.dirty.bits = make([]uint64, (t.rows+63)/64)
		t.touchAll()
	}
}

// RowDirty reports whether the cells of row y of the visible screen may
// have changed since the last ClearDirty: characters, styles or rare, by
// output, scrolling, erasing, switching screens, Resize or a change of
// palette or reverse video. Before the first ClearDirty every row is dirty.
//
// The cursor is not part of it: a renderer compares Cursor with what it
// drew. Rows out of range are never dirty.
func (t *Terminal) RowDirty(y int) bool {
	if uint(y) >= uint(t.rows) {
		return false
	}
	return t.dirty == nil || t.dirty.bits[y>>6]&(1<<(y&63)) != 0
}

// Dirty reports whether any row is dirty (see RowDirty).
func (t *Terminal) Dirty() bool {
	if t.dirty == nil {
		return true
	}
	for _, w := range t.dirty.bits {
		if w != 0 {
			return true
		}
	}
	return false
}

// ClearDirty marks every row clean, typically after a renderer has drawn
// the dirty ones. Damage tracking starts with the first call; until then
// it costs nothing.
func (t *Terminal) ClearDirty() {
	if t.dirty == nil {
		t.dirty = &dirtySet{bits: make([]uint64, (t.rows+63)/64)}
		return
	}
	clear(t.dirty.bits)
}
