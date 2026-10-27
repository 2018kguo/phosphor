package phosphor

import (
	"errors"
	"runtime"
	"strings"
	"testing"
	"unsafe"
)

func TestCompactOwnsOnlyNonzeroRowPrefix(t *testing.T) {
	term := New(80, 24)
	const prompt = "hello world $ "
	term.Write([]byte(prompt))
	term.Compact()
	row := term.main[0]
	if len(row) != len(prompt) || cap(row) != len(prompt) || term.Cell(79, 0) != (Cell{Width: 1}) {
		t.Fatalf("owned row len=%d cap=%d, virtual tail=%+v", len(row), cap(row), term.Cell(79, 0))
	}
	for _, row := range term.main[1:] {
		if row != nil {
			t.Fatal("blank row retained storage")
		}
	}
	if allocs := testing.AllocsPerRun(100, term.Compact); allocs != 0 {
		t.Fatalf("repeated prefix compaction allocated %g times", allocs)
	}
	if unsafe.Sizeof(cell{}) != 16 || unsafe.Sizeof(Terminal{}) > 560 {
		t.Fatal("prefix storage enlarged fixed terminal or cell storage")
	}
	term.Write([]byte("\rhello world $ ")) // first post-idle write materializes the row
	if n := testing.AllocsPerRun(100, func() { term.Write([]byte("\rhello world $ ")) }); n != 0 {
		t.Fatalf("steady ASCII after resuming allocated %g times", n)
	}
}

func TestSnapshotRowPrefixBudget(t *testing.T) {
	term := New(80, 24)
	term.Write([]byte("short"))
	// MaxBytes counts actual decoded storage, while MaxCells continues to
	// bound the full virtual geometry available to future output.
	budget := int(unsafe.Sizeof(Terminal{})) + 24*int(unsafe.Sizeof([]cell{})) + 5*int(unsafe.Sizeof(cell{}))
	data := term.Snapshot()
	var restored Terminal
	if err := restored.RestoreSnapshot(data, SnapshotLimits{MaxBytes: budget}); err != nil {
		t.Fatal(err)
	}
	if len(restored.main[0]) != 5 || cap(restored.main[0]) != 5 {
		t.Fatal("snapshot decoder padded the stored prefix")
	}
	assertSnapshotState(t, term, &restored)
	if err := restored.RestoreSnapshot(data, SnapshotLimits{MaxBytes: budget - 1}); !errors.Is(err, ErrSnapshotLimit) {
		t.Fatalf("decoded row budget was not enforced: %v", err)
	}
	if err := restored.RestoreSnapshot(data, SnapshotLimits{MaxCells: 80*24 - 1}); !errors.Is(err, ErrSnapshotLimit) {
		t.Fatalf("virtual cell budget was not enforced: %v", err)
	}
}

func TestPrefixRowsHideDiscardedCellsOnGrowth(t *testing.T) {
	term := New(10, 3)
	term.Write([]byte("\x1b[?7lABCDEFGHIJ"))
	term.Resize(4, 3)
	if len(term.main[0]) != 4 || cap(term.main[0]) != 10 {
		t.Fatal("truncation hid backing capacity from Compact")
	}
	term.Resize(10, 3)
	term.Write([]byte("\x1b[10GZ"))
	for x := 4; x < 9; x++ {
		if c := term.Cell(x, 0); c != (Cell{Width: 1}) {
			t.Fatalf("discarded cell reappeared at %d: %+v", x, c)
		}
	}
	if term.Line(0) != "ABCD     Z" {
		t.Fatal(term.Line(0))
	}
	term.Resize(4, 3)
	term.Compact()
	if cap(term.main[0]) != 4 {
		t.Fatal("Compact retained the old wider allocation")
	}
}

func TestPrefixRowsKeepPhysicalWrappedExtent(t *testing.T) {
	a, b := New(8, 3), New(8, 3)
	for _, term := range []*Terminal{a, b} {
		term.main[0], term.main[1] = make([]cell, 8), make([]cell, 8)
		term.main[0][0].R, term.main[1][0].R = 'A', 'B'
		term.mainWrap = []bool{true, false, false}
		term.cur.x, term.cur.y = 1, 1
	}
	b.Compact()
	painted := New(8, 3)
	painted.Write(b.Repaint(false))
	for y := range 3 {
		for x := range 8 {
			if a.Cell(x, y) != painted.Cell(x, y) {
				t.Fatalf("wrapped implicit zero changed at %d,%d", x, y)
			}
		}
	}
	for _, size := range [][2]int{{4, 6}, {12, 3}, {8, 3}} {
		a.Resize(size[0], size[1])
		b.Resize(size[0], size[1])
		b.Compact()
		assertSnapshotState(t, a, b)
	}
}

func TestPrefixRowsKeepRareStyledSpaces(t *testing.T) {
	for _, alternate := range []bool{false, true} {
		term := New(8, 3)
		if alternate {
			term.Write([]byte("\x1b[?47h"))
		}
		term.Write([]byte("\x1b[58;5;3m\x1b]8;;https://space\a \x1b]8;;\a\x1b[0m"))
		term.Compact()
		if alternate {
			term.Write([]byte("\x1b[?47l"))
			term.Compact()
			term.Write([]byte("\x1b[?47h"))
		}
		assertSpace := func() {
			if c := term.Cell(0, 0); c.Rune != ' ' || c.Style.UnderlineColor != IndexedColor(3) || c.Link.URI != "https://space" {
				t.Fatalf("styled space was mistaken for an empty row: %+v", c)
			}
		}
		assertSpace()
		term.Restart()
		term.Compact()
		assertSpace()
		if term.Cursor().Y != 1 || term.Modes().AltScreen {
			t.Fatal("Restart did not keep the styled-space row above the fresh prompt")
		}
	}
}

func TestPrefixRowsFutureOperations(t *testing.T) {
	for _, input := range []string{
		"prompt $ ",
		"short\x1b#8", // alignment test covers implicit columns too
		"\x1b[?2027h❤\x1b[58;5;3m️\x1b[?2027l á",
		"\x1b[1\"qA\x1b[0\"q\x1b[80G\x1b[?2K",
		"\x1b[1;3HA\x1b[2;5HB\x1b[3;7HC\x1b[?69h\x1b[3;8s\x1b[H\x1b[S\x1b[T",
		"\x1b[58;5;4m\x1b]8;;https://saved\aX\x1b7\x1b[?1049hALT\x1b7\x1b[?1049l\x1b8",
		strings.Repeat("\x1b[41mhistory 漢 á\x1b[0m\r\n", 20) + "\x1b[2J\x1b[H$ ",
	} {
		t.Run(input[:min(24, len(input))], func(t *testing.T) {
			a, b := New(80, 6), New(80, 6)
			for _, v := range []byte(input) {
				a.Write([]byte{v})
				b.Compact()
				b.Write([]byte{v})
				b.Compact()
				assertSnapshotState(t, a, b)
			}
			for _, size := range [][2]int{{4, 8}, {100, 10}, {80, 6}} {
				a.Resize(size[0], size[1])
				b.Resize(size[0], size[1])
				b.Compact()
				assertSnapshotState(t, a, b)
			}
			for _, suffix := range []string{"\x1b8", "\x1b[3b", "\x1b[?47h\x1b8Q\x1b[?47l", "\x1b[2;2H\x1b[@\x1b[P\x1b[5X"} {
				a.Write([]byte(suffix))
				b.Write([]byte(suffix))
				b.Compact()
				assertSnapshotState(t, a, b)
			}
		})
	}
}

func BenchmarkPrefixRowsIdleMemory(b *testing.B) {
	for _, compact := range []bool{false, true} {
		name := "active"
		if compact {
			name = "compact"
		}
		b.Run(name, func(b *testing.B) {
			const count = 512
			for b.Loop() {
				terms := make([]*Terminal, count)
				runtime.GC()
				var before, after runtime.MemStats
				runtime.ReadMemStats(&before)
				for i := range terms {
					term := New(80, 24)
					term.Write([]byte("hello world $ "))
					if compact {
						term.Compact()
					}
					terms[i] = term
				}
				runtime.GC()
				runtime.GC()
				runtime.ReadMemStats(&after)
				b.ReportMetric(float64(int64(after.HeapAlloc)-int64(before.HeapAlloc))/count, "heap-B/term")
				runtime.KeepAlive(terms)
			}
		})
	}
}

func FuzzPrefixRows(f *testing.F) {
	for _, input := range []string{
		"short\x1b[80GX\x00\x03\x05\x1b[?7l\x1b[10GZ",
		"\x1b[?2027h❤\x1b[58;5;3m️\x1b7\x1b[?1049hALT\x1b[?1049l\x1b8",
		"\x1b[1\"qA\x1b[0\"q\x1b[?2K\x1b[2J\x1b[Hprompt",
	} {
		f.Add([]byte(input), uint8(12), uint8(4))
	}
	f.Fuzz(func(t *testing.T, input []byte, c, r uint8) {
		if len(input) > 4096 {
			input = input[:4096]
		}
		a, b := New(1+int(c%40), 1+int(r%12)), New(1+int(c%40), 1+int(r%12))
		fuzzOps(input, 40, 12, func(data []byte) {
			a.Write(data)
			b.Compact()
			b.Write(data)
			b.Compact()
			assertSnapshotState(t, a, b)
		}, func(cols, rows int) {
			a.Resize(cols, rows)
			b.Resize(cols, rows)
			b.Compact()
			assertSnapshotState(t, a, b)
		})
	})
}
