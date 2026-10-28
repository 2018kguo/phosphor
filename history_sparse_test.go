package phosphor

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestSparseHistoryPreservesEmptyAndLiteralSpaces(t *testing.T) {
	for _, tc := range []struct {
		name, prefix string
		literal      bool
	}{
		{"cursor", "\x1b[3GA", false},
		{"literal", "  A", true},
		{"background", "\x1b[41m\x1b[2J\x1b[3GA\x1b[0m", false},
		{"hyperlink", "\x1b]8;;https://example.com\x1b\\\x1b[3GA\x1b]8;;\x1b\\", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			term := New(8, 2)
			term.Write([]byte(tc.prefix + "\r\nB\r\nC"))
			cells := term.ScrollbackCells(0)
			want := rune(0)
			if tc.literal {
				want = ' '
			}
			if len(cells) != 1 || cells[0][0].Rune != want || cells[0][1].Rune != want || cells[0][2].Rune != 'A' {
				t.Fatalf("history cells: %+v", cells)
			}
			if tc.literal && !bytes.Equal(term.sb[term.sbStart], []byte("  A")) {
				t.Fatalf("ordinary history bytes changed: %q", term.sb[term.sbStart])
			}
			for _, line := range term.sb[term.sbStart:] {
				if !validateHistoryLine(line) {
					t.Fatalf("invalid generated history: %q", line)
				}
			}
			restored := New(8, 2)
			restored.Write(term.Repaint(true))
			assertSparseObservations(t, term, restored)
			for _, suffix := range []string{"\x1b8Z", "\r\nnext\r\nlast"} {
				term.Write([]byte(suffix))
				restored.Write([]byte(suffix))
				assertSparseObservations(t, term, restored)
			}
			for _, size := range [][2]int{{4, 4}, {12, 2}, {3, 6}, {8, 2}} {
				term.Resize(size[0], size[1])
				restored.Resize(size[0], size[1])
				assertSparseObservations(t, term, restored)
			}
		})
	}
}

func assertSparseObservations(t *testing.T, a, b *Terminal) {
	t.Helper()
	if a.Cursor() != b.Cursor() || a.Modes() != b.Modes() || a.ScrollbackLen() != b.ScrollbackLen() {
		t.Fatalf("cursor/modes/history changed: %+v/%+v, %d/%d", a.Cursor(), b.Cursor(), a.ScrollbackLen(), b.ScrollbackLen())
	}
	for y := range a.rows {
		for x := range a.cols {
			if a.Cell(x, y) != b.Cell(x, y) {
				t.Fatalf("screen %d,%d: %+v/%+v", x, y, a.Cell(x, y), b.Cell(x, y))
			}
		}
	}
	for i := range a.ScrollbackLen() {
		if !reflect.DeepEqual(a.ScrollbackCells(i), b.ScrollbackCells(i)) {
			t.Fatalf("history %d: %+v/%+v", i, a.ScrollbackCells(i), b.ScrollbackCells(i))
		}
	}
}

func TestSparseHistoryLongWrappedLine(t *testing.T) {
	term := New(64, 2, WithScrollbackBytes(1<<20))
	row := make([]cell, 64)
	for i := range row {
		if i%2 != 0 {
			row[i].R = 'x'
		}
	}
	for range 1000 {
		term.pushScrollback(row, rowWrap, row)
	}
	columns := 0
	for _, line := range term.sb[term.sbStart:] {
		if len(line) > maxLogicalLine || !validateHistoryLine(line) {
			t.Fatalf("invalid bounded wrapped entry: %d bytes", len(line))
		}
		columns += len(stripSGR(line))
	}
	if columns != 64*1000 {
		t.Fatalf("wrapped history kept %d columns", columns)
	}
	for _, row := range term.ScrollbackCells(0) {
		for x, c := range row {
			want := rune(0)
			if x%2 != 0 {
				want = 'x'
			}
			if c.Rune != want {
				t.Fatalf("column %d: %+v", x, c)
			}
		}
	}
}

func TestSparseHistoryValidation(t *testing.T) {
	for _, line := range [][]byte{
		{0}, {0, 0}, {0, 1}, {0, 2, ' '}, {0, 1, 'x'}, {0, 0x81, 0, ' '},
		{0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 1},
		[]byte("prefix\x00\x01 \x1b[6n"), []byte("\x1b[31\x00\x01 m"),
	} {
		if validateHistoryLine(line) {
			t.Fatalf("accepted malformed history %q", line)
		}
	}
	for _, line := range [][]byte{[]byte("  A"), []byte("prefix\x00\x02  A"), []byte("\x1b[41m\x00\x01 \x1b[0m")} {
		if !validateHistoryLine(line) {
			t.Fatalf("rejected valid history %q", line)
		}
	}
	old := replayLine([]byte("  A"), 8)
	if old.Cell(0, 0).Rune != ' ' || old.Cell(1, 0).Rune != ' ' {
		t.Fatal("legacy history spaces were guessed to be empty")
	}
}

func sparseRichHistory() *Terminal {
	term := New(20, 2)
	term.Write([]byte("\x1b[?2027h\x1b[3G\x1b]8;;https://example.com/" + strings.Repeat("a", 3500) + "\x1b\\❤️\x1b]8;;\x1b\\\r\nX\r\nY"))
	return term
}

func TestSparseHistoryRichGeometry(t *testing.T) {
	term := sparseRichHistory()
	line := term.sb[term.sbStart]
	tmp := replayLine(line, 20)
	if tmp.rows != 1 || tmp.Cell(0, 0).Rune != 0 || tmp.Cell(2, 0).Rune != '❤' || tmp.Cell(2, 0).Width != 2 {
		t.Fatalf("rich history geometry: %dx%d %+v", tmp.cols, tmp.rows, tmp.main)
	}
	restored := New(20, 2)
	restored.Write(term.Repaint(true))
	assertSparseObservations(t, term, restored)
}

func BenchmarkSparseRichHistoryDecode(b *testing.B) {
	term := sparseRichHistory()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = term.ScrollbackCells(0)
	}
}

func FuzzSparseHistoryValidation(f *testing.F) {
	for _, line := range []string{"", "  A", "prefix\x00\x02  A", "\x1b[41m\x00\x01 \x1b[0m", "\x1b]8;;https://example.com\x1b\\A\x1b]8;;\x1b\\", "\x00\x00P\x01", "\x00\x00P\x01hello\x00\x00P\x02\x00\x02  "} {
		f.Add([]byte(line), uint8(8))
	}
	f.Fuzz(func(t *testing.T, line []byte, width uint8) {
		if len(line) > maxLogicalLine || !validateHistoryLine(line) {
			return
		}
		cols := int(width%64) + 1
		tmp := replayLine(line, cols)
		if tmp.cur.y < 0 || tmp.cur.y >= len(tmp.main) {
			t.Fatalf("history replay cursor outside rows: %+v", tmp.cur)
		}
		_ = stripSGR(line)
		usedSave := false
		_ = appendHistoryRepaint(nil, line, cols, false, &usedSave)
	})
}
