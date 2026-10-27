package phosphor

// Compact releases unused storage after a terminal becomes idle. It drops
// rows containing only zero cells, unused wrap flags, dead Unicode and
// hyperlink entries, invalid saved cursor storage and temporary buffers.
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
	if saved := t.saved; saved != nil && !saved.screen[0].valid && !saved.screen[1].valid && !saved.alternate.valid {
		t.saved = nil
	}
	t.compactPools()
}

// Pool indexes remain stable when everything is still live. The small
// mark set stays on the stack for up to 256 pooled strings, so an already
// compact prompt does not allocate or rebuild its tables on every call.
func (t *Terminal) compactPools() {
	x := t.x
	if x == nil {
		return
	}
	nClusters, nLinks := len(x.clusters), len(x.links)
	if n := nClusters + nLinks; n > 0 {
		var small [4]uint64
		words := (n + 63) / 64
		marks := small[:]
		if words > len(small) {
			marks = make([]uint64, words)
		}
		liveClusters, liveLinks := 0, 0
		mark := func(i int) {
			bit := uint64(1) << uint(i%64)
			if marks[i/64]&bit != 0 {
				return
			}
			marks[i/64] |= bit
			if i < nClusters {
				liveClusters++
			} else {
				liveLinks++
			}
		}
		t.visitClusterRefs(func(r *rune) {
			if *r >= clusterBase {
				mark(int(*r - clusterBase))
			}
		})
		t.visitLinkRefs(func(l *uint16) {
			if *l != 0 {
				mark(nClusters + int(*l) - 1)
			}
		})
		if liveClusters < nClusters {
			t.compactClusters()
		}
		if liveLinks < nLinks {
			t.compactLinks()
		}
	}
	if x.tabs == nil && len(x.savedModes) == 0 && len(x.clusters) == 0 && len(x.links) == 0 {
		t.x = nil
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
