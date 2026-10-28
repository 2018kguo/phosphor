package phosphor

import "testing"

// A final MAIN with only 1047 reset must leave ALT through an erasing
// transition. It cannot retain uniformly erased old rows plus default rows
// introduced by height or width growth. The allowed shape is one top-left
// erased rectangle with zero padding. Text, protection, other styles, and row flags
// are excluded; those still require repaint equality. Width-only growth at 80/132 remains strict: DECCOLM may stage
// those widths. Native Terminal.zig deccolm only reaches 80/132 and cannot
// restore other host widths; switchScreenMode 1047 erases the entire ALT.
// The raw native fixture cursor-1047-resize-background-limit records
// the ANSI failure. FuzzSnapshotContinuation is the universal exact checkpoint
// oracle, including states outside portable ANSI's reconstruction contract.
func repaintRectangleGrowthLimit(src *Terminal) bool {
	if src.altActive || src.privateModes()&(privateMode47|privateMode1047|privateMode1049) != privateMode47|privateMode1049 || len(src.alt) != src.rows || src.rows < 1 {
		return false
	}
	bg := rowCell(src.alt[0], 0).Bg
	if bg == 0 {
		return false
	}
	width := 0
	for width < src.cols && rowCell(src.alt[0], width) == (cell{Bg: bg}) {
		width++
	}
	if width == 0 {
		return false
	}
	height := 0
	for height < src.rows && rowCell(src.alt[height], 0) == (cell{Bg: bg}) {
		height++
	}
	if width == src.cols && height == src.rows {
		return false
	}
	if height == src.rows && (src.cols == 80 || src.cols == 132) {
		return false
	}
	for y, row := range src.alt {
		if rowFlagsAt(src.altWrap, y) != 0 {
			return false
		}
		for x := range src.cols {
			want := cell{}
			if x < width && y < height {
				want.Bg = bg
			}
			if rowCell(row, x) != want {
				return false
			}
		}
	}
	return true
}

// Portable repaint's declared loss is only the default background in zero
// ALT padding. Keep every other assertion active against this explicit export.
func repaintRectangleGrowthExpectation(t *testing.T, src *Terminal) *Terminal {
	t.Helper()
	expected := New(1, 1)
	if err := expected.RestoreSnapshot(src.Snapshot(), SnapshotLimits{}); err != nil {
		t.Fatal(err)
	}
	bg := rowCell(src.alt[0], 0).Bg
	for y, row := range expected.alt {
		full := make([]cell, src.cols)
		copy(full, row)
		for x := range full {
			if full[x] == (cell{}) {
				full[x].Bg = bg
			}
		}
		expected.alt[y] = full
	}
	return expected
}

func checkRectangleGrowthSnapshot(t *testing.T, src *Terminal) {
	t.Helper()
	before := src.Snapshot()
	defer func() {
		if err := src.RestoreSnapshot(before, SnapshotLimits{}); err != nil {
			t.Fatal(err)
		}
	}()
	a, b := src, New(1, 1)
	if err := b.RestoreSnapshot(before, SnapshotLimits{}); err != nil {
		t.Fatal(err)
	}
	assertSnapshotState(t, a, b)
	for _, suffix := range []string{"\x1b[?47h", "\x1b8X\r\n\x1b[31mY", "\x1b[?1049l\x1b8Z"} {
		a.Write([]byte(suffix))
		b.Write([]byte(suffix))
		assertSnapshotState(t, a, b)
	}
	cols, rows := src.cols+1, src.rows+1
	a.Resize(cols, rows)
	b.Resize(cols, rows)
	a.Write([]byte("\x1b[?47h\x1b8Q"))
	b.Write([]byte("\x1b[?47h\x1b8Q"))
	assertSnapshotState(t, a, b)
}

func TestRepaintHeightGrowthLimit(t *testing.T) {
	source := New(3, 11, WithScrollback(20))
	source.Write([]byte("\x1b[?1049h\x1b[?47h\x1b[48;2;1;2;3m\x1b[?1047l"))
	source.Resize(3, 12)
	if !repaintRectangleGrowthLimit(source) {
		t.Fatal("height growth not recognized")
	}
	checkRectangleGrowthSnapshot(t, source)
	replay := New(3, 12)
	replay.Write(source.Repaint(false))
	if stateOf(source, false) == stateOf(replay, false) {
		t.Fatal("ANSI limit changed; reassess fuzz boundary")
	}
	checkRepaint(t, source)       // all facts except the declared ALT background loss
	source.Write([]byte("\x1bV")) // live protection does not protect existing ALT cells
	if !repaintRectangleGrowthLimit(source) {
		t.Fatal("live ISO protection changed rectangle classification")
	}
	checkRepaint(t, source)
	source.Write([]byte("\x1bW"))
	for name, change := range map[string]func(*Terminal){
		"active alternate":       func(s *Terminal) { s.Write([]byte("\x1b[?47h")) },
		"non-erasing exit":       func(s *Terminal) { s.Write([]byte("\x1b[?47l")) },
		"text":                   func(s *Terminal) { s.alt[0][0].R = 'X' },
		"protected cell":         func(s *Terminal) { s.alt[0][0].A = AttrProtected },
		"foreground":             func(s *Terminal) { s.alt[0][0].Fg = IndexedColor(1) },
		"row flags":              func(s *Terminal) { s.altWrap = make([]rowFlags, s.rows); s.altWrap[0] = rowWrap },
		"uneven row widths":      func(s *Terminal) { s.alt[0] = s.alt[0][:2] },
		"no new rows":            func(s *Terminal) { s.Resize(3, 11) },
		"colored row after zero": func(s *Terminal) { s.alt[5] = nil },
	} {
		t.Run(name, func(t *testing.T) {
			var s Terminal
			if err := s.RestoreSnapshot(source.Snapshot(), SnapshotLimits{}); err != nil {
				t.Fatal(err)
			}
			change(&s)
			if repaintRectangleGrowthLimit(&s) {
				t.Fatal("near miss excluded from strict repaint checks")
			}
		})
	}
}

func TestRepaintWidthGrowthLimit(t *testing.T) {
	for _, dims := range [][3]int{{13, 47, 8}, {80, 132, 8}, {13, 80, 8}, {1, 24, 1}} {
		source := New(dims[0], 12, WithScrollback(20))
		source.Write([]byte("\x1b[?47h\x1b[?1049h\x1b[48;2;1;2;3m\x1b[?1047l"))
		source.Resize(dims[1], dims[2])
		want := dims[1] != 80 && dims[1] != 132
		if repaintRectangleGrowthLimit(source) != want {
			t.Fatalf("width %d to%d boundary", dims[0], dims[1])
		}
		if want {
			checkRepaint(t, source)
		}
	}
}
