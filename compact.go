package phosphor

// Compact releases unused storage after a terminal becomes idle. It drops
// unused zero tails and rows, unused wrap flags, dead Unicode and
// cell attribute entries, invalid saved cursor storage and temporary buffers.
// It keeps screen contents, scrollback, cursors, modes, damage and unfinished
// parser input unchanged. Later output may allocate those buffers again.
//
// Like Write and Resize, Compact requires exclusive access to the Terminal.
// The caller decides when to run it; output never compacts itself.
func (t *Terminal) Compact() {
	for _, screen := range [][][]cell{t.main, t.alt} {
		for y, row := range screen {
			end := len(row)
			for end > 0 && row[end-1] == (cell{}) {
				end--
			}
			if end == 0 {
				screen[y] = nil
			} else if end < len(row) || end < cap(row) {
				// Exact capacity makes repeated idle compaction allocation-free.
				kept := make([]cell, end)
				copy(kept, row[:end])
				screen[y] = kept
			}
		}
	}
	if !t.anyRowFlags(t.mainWrap) {
		t.mainWrap = nil
	}
	if !t.anyRowFlags(t.altWrap) {
		t.altWrap = nil
	}
	t.scratch = nil
	if t.state < stOSC {
		t.osc = nil
	}
	if saved := t.saved; saved != nil && !saved.screen[0].valid && !saved.screen[1].valid {
		t.saved = nil
	}
	t.compactPools()
}

// Pool indexes remain stable when everything is still live. The small
// mark set stays on the stack for up to 256 pooled entries, so an already
// compact prompt does not allocate or rebuild its tables on every call.
func (t *Terminal) compactPools() {
	x := t.x
	if x == nil {
		return
	}
	nClusters, nExtras := len(x.clusters), len(x.rare)
	if n := nClusters + nExtras; n > 0 {
		var small [4]uint64
		words := (n + 63) / 64
		marks := small[:]
		if words > len(small) {
			marks = make([]uint64, words)
		}
		liveClusters, liveExtras := 0, 0
		mark := func(i int) {
			bit := uint64(1) << uint(i%64)
			if marks[i/64]&bit != 0 {
				return
			}
			marks[i/64] |= bit
			if i < nClusters {
				liveClusters++
			} else {
				liveExtras++
			}
		}
		t.visitClusterRefs(func(r *rune) {
			if *r >= clusterBase {
				mark(int(*r - clusterBase))
			}
		})
		t.visitExtraRefs(func(l *uint16) {
			if *l != 0 {
				mark(nClusters + int(*l) - 1)
			}
		})
		if liveClusters < nClusters {
			t.compactClusters()
		}
		if liveExtras < nExtras {
			t.compactExtras()
		}
	}
	if !t.altActive && x.tabs == nil && len(x.savedModes) == 0 && x.privateModes == 0 && len(x.clusters) == 0 && len(x.rare) == 0 {
		t.x = nil
	}
}
