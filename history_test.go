package phosphor

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand/v2"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"unsafe"
)

func TestColdPrefixWrites(t *testing.T) {
	term := New(80, 24)
	const prompt = "user@host:~$ "
	term.Write([]byte(prompt))
	if row := term.main[0]; len(row) != len(prompt) || cap(row) != len(prompt) {
		t.Fatalf("cold prompt allocated a larger row: len=%d cap=%d", len(row), cap(row))
	}
	if n := testing.AllocsPerRun(100, term.Compact); n != 0 {
		t.Fatalf("an exact cold prefix needs another allocation to compact: %g", n)
	}
	if n := testing.AllocsPerRun(100, func() { term.Write([]byte("\x1b[K")) }); n != 0 || len(term.main[0]) != len(prompt) {
		t.Fatalf("default background EL materialized implicit zeros: allocs=%g len=%d", n, len(term.main[0]))
	}
	term = New(80, 24)
	changes, previous := 0, 0
	for i := range 80 {
		term.Write([]byte{'a'})
		row := term.main[0]
		if len(row) != i+1 || cap(row) > term.cols {
			t.Fatalf("byte %d: len=%d cap=%d", i, len(row), cap(row))
		}
		if cap(row) != previous {
			previous = cap(row)
			changes++
		}
	}
	if changes > 8 {
		t.Fatalf("byte-sized writes grew %d times instead of geometrically", changes)
	}
	term = New(80, 24)
	term.Write([]byte("界á"))
	if row := term.main[0]; len(row) != 3 || cap(row) > 4 {
		t.Fatalf("Unicode prompt allocated a full row: len=%d cap=%d", len(row), cap(row))
	}
	term.Write([]byte("\x1b[H\x1b[4hZ"))
	if len(term.main[0]) != 80 || term.Cell(0, 0).Rune != 'Z' || term.Cell(1, 0).Rune != '界' {
		t.Fatal("insert mode failed to materialize and shift the row")
	}
	term = New(80, 24)
	term.Write([]byte("\x1b[?47hshort"))
	if len(term.alt[0]) != 80 {
		t.Fatal("alternate-screen prints did not retain the dense row policy")
	}
}

func TestColdPrefixEraseWideAndStyledCells(t *testing.T) {
	for _, input := range []string{
		"\x1b[58;5;3m\x1b]8;;https://wide\a界界\x1b[0m\x1b]8;;\a\x1b[2G\x1b[X",
		"\x1b[31;4m界界\x1b[0m\x1b[3G\x1b[2X",
		"\x1b[31;58;5;3mshort\x1b[0m\x1b[K",
		"\x1b[44mshort\x1b[K",
		"\x1b[1\"q界A\x1b[0\"qB\x1b[H\x1b[?K",
	} {
		a, b := New(80, 3), New(80, 3)
		for _, v := range []byte(input) {
			materializeColdControl(a)
			a.Write([]byte{v})
			b.Write([]byte{v})
			assertSnapshotState(t, a, b)
		}
	}
}

func TestColdPrefixEraseDamageDoesNotDependOnStorage(t *testing.T) {
	for _, empty := range []bool{false, true} {
		for _, wrapped := range []bool{false, true} {
			a := New(80, 3)
			a.Write([]byte("long previously owned row\x1b[2J\x1b[Hshort"))
			if empty {
				a.Write([]byte("\x1b[2J\x1b[H"))
			}
			a.setWrap(0, wrapped)
			a.ClearDirty()
			var b Terminal
			if err := b.RestoreSnapshot(a.Snapshot(), SnapshotLimits{}); err != nil {
				t.Fatal(err)
			}
			b.Compact()
			a.Write([]byte("\x1b[K"))
			b.Write([]byte("\x1b[K"))
			assertSnapshotState(t, a, &b)
			if a.RowDirty(0) != wrapped {
				t.Fatalf("empty=%v wrapped=%v: no-op erase damage=%v", empty, wrapped, a.RowDirty(0))
			}
		}
	}
}

func TestColdPrefixSteadyFortyRows(t *testing.T) {
	for _, tui := range []bool{false, true} {
		name := "logs"
		frame := []byte(strings.Repeat("log line with a fixed ASCII message\r\n", 40))
		if tui {
			name = "tui"
			frame = nil
			for y := range 40 {
				frame = fmt.Appendf(frame, "\x1b[%d;1H\x1b[32mrow content with fixed ASCII width\x1b[0m", y+1)
			}
		}
		t.Run(name, func(t *testing.T) {
			term := New(120, 40, WithScrollback(0))
			term.Write(frame)
			term.Write(frame)
			if n := testing.AllocsPerRun(100, func() { term.Write(frame) }); n != 0 {
				t.Fatalf("warm forty-row output allocated %g times", n)
			}
		})
	}
}

func materializeColdControl(term *Terminal) {
	// The oracle holds the same cells in full-width arrays. Do not mark
	// damage: this changes storage only, like Compact in the other direction.
	for _, screen := range [][][]cell{term.main, term.alt} {
		for y, row := range screen {
			screen[y] = growRow(row, term.cols, term.cols)
		}
	}
}

func FuzzColdPrefixWrites(f *testing.F) {
	for _, input := range []string{
		"\x1b[32muser@host\x1b[0m:~$ ",
		"界á\x1b[?2027h❤\x1b[58;5;3m️\x1b[4hINSERT\x1b[4l",
		"\x1b[44mshort\r\nnext\r\nthird\r\n\x1b[2J\x1b[H",
		"\x1b[58;5;3m\x1b]8;;https://wide\a界界\x1b[0m\x1b]8;;\a\x1b[2G\x1b[X\x1b[80G\x1b[K",
		"0\x0030\x1b#8\x1b[?7lABCDEFGHIJ\x00\x03\x05\x00\x20\x05\x1b[30GZ",
	} {
		f.Add([]byte(input), uint8(12), uint8(4), uint8(1))
	}
	f.Fuzz(func(t *testing.T, input []byte, c, r, chunk uint8) {
		if len(input) > 1024 {
			input = input[:1024]
		}
		a, b := New(1+int(c%40), 1+int(r%12)), New(1+int(c%40), 1+int(r%12))
		fuzzOps(input, 40, 12, func(data []byte) {
			for len(data) > 0 {
				n := min(1+int(chunk%17), len(data))
				materializeColdControl(a)
				a.Write(data[:n])
				b.Write(data[:n])
				assertSnapshotState(t, a, b)
				data = data[n:]
			}
		}, func(cols, rows int) {
			a.Resize(cols, rows)
			b.Resize(cols, rows)
			assertSnapshotState(t, a, b)
		})
	})
}

func TestErasedRowsHideRecycledCells(t *testing.T) {
	for _, alternate := range []bool{false, true} {
		for _, background := range []string{"", "\x1b[44m"} {
			term := New(12, 3)
			if alternate {
				term.Write([]byte("\x1b[?47h"))
			}
			term.Write([]byte("\x1b[31;58;5;3m\x1b]8;;https://old\a界áOLD CELLS\x1b]8;;\a\x1b[0m"))
			owned := term.screen()[0]
			term.Write([]byte(background + "\x1b[2J\x1b[H"))
			if row := term.screen()[0]; cap(row) != cap(owned) {
				t.Fatal("erasure discarded reusable row capacity")
			}
			for y := range 3 {
				for x := range 12 {
					want := Cell{Width: 1}
					if background != "" {
						want.Style.Bg = IndexedColor(4)
					}
					if got := term.Cell(x, y); got != want {
						t.Fatalf("alternate=%v background=%q cell %d,%d = %+v, want %+v", alternate, background, x, y, got, want)
					}
				}
			}
			term.Write([]byte("\x1b[4Gnew\x1b[10GZ"))
			if got := term.Line(0); got != "   new   Z" {
				t.Fatalf("gap exposed old cells: %q", got)
			}
			term.Write([]byte("\x1b[0m\x1b[2J\x1b[H"))
			term.Compact()
			for _, row := range term.screen() {
				if row != nil {
					t.Fatal("erased idle row retained storage after Compact")
				}
			}
		}
	}
}

func TestErasedRowsMatchEagerStorage(t *testing.T) {
	traces := []string{
		"\x1b[31;58;5;3m\x1b]8;;https://old\a界áOLD CELLS\x1b]8;;\a\x1b[0m\x1b[2J\x1b[H\x1b[4Gabc\x1b[10GZ",
		"\x1b[1\"q界PROTECTED\x1b[0\"q\x1b[2J\x1b[H\x1b[3GAB界CD\x1b[4Gxyz\x1b[?2K\x1b[2GQ",
		"\x1b[?2027h❤\x1b[58;5;3m️\x1b[0m\x1b[2J\x1b[H\x1b[6G界á\x1b[HXY\x1b[4hZ\x1b[4l\x1b[P",
		strings.Repeat("\x1b[32mhistory 界 á\x1b[0m\r\n", 14) + "\x1b[2J\x1b[H$ \x1b[7Gend",
		"\x1b#8\x1b[2;3r\x1b[3;1H\n\x1b[4Gshort\x1b[2;1H\x1bM\x1b[9GZ\x1b[r",
		"\x1b[?47h\x1b#8\x1b[2J\x1b[Halt\x1b[2;4Hgap\x1b[?47lmain\x1b[2J\x1b[H\x1b[?47h\x1b[6GX\x1b[?47l",
		"\x1b#8\x1b[?69h\x1b[3;10s\x1b[2J\x1b[H\x1b[S\x1b[3GX\x1b[2;8H界\x1b[T\x1b[?69l",
	}
	for i, input := range traces {
		for _, cols := range []int{1, 4, 12, 30} {
			for _, chunk := range []int{1, 7, len(input)} {
				t.Run(fmt.Sprintf("trace%d/%dcols/%dbytes", i, cols, chunk), func(t *testing.T) {
					a, b := New(cols, 3), New(cols, 3)
					for p := []byte(input); len(p) > 0; {
						n := min(chunk, len(p))
						writeErasedRowsOracle(a, b, p[:n])
						assertSnapshotState(t, a, b)
						if !bytes.Equal(a.Snapshot(), b.Snapshot()) {
							t.Fatal("exact checkpoint depends on erased backing cells")
						}
						a.ClearDirty()
						b.ClearDirty()
						p = p[n:]
					}
					for _, size := range [][2]int{{3, 6}, {18, 2}, {cols, 3}} {
						a.Resize(size[0], size[1])
						b.Resize(size[0], size[1])
						a.Compact()
						assertSnapshotState(t, a, b)
						var restored Terminal
						if err := restored.RestoreSnapshot(a.Snapshot(), SnapshotLimits{}); err != nil {
							t.Fatal(err)
						}
						assertSnapshotState(t, a, &restored)
						writeErasedRowsOracle(a, b, []byte("\x1b[H\x1b[2J\x1b[2GX界\r\nnext\x1b[5GZ"))
						assertSnapshotState(t, a, b)
					}
				})
			}
		}
	}
}

func writeErasedRowsOracle(a, b *Terminal, input []byte) {
	a.Write(input)
	for _, v := range input {
		// Full-width storage clears discarded capacity before any parser
		// operation can expose it; single-byte writes also avoid ASCII runs.
		materializeColdControl(b)
		b.Write([]byte{v})
	}
}

func FuzzErasedRows(f *testing.F) {
	for _, input := range []string{
		"\x1b#8\x1b[2J\x1b[H\x1b[4Gabc\x1b[9GZ",
		"\x1b[1\"q界OLD\x1b[0\"q\x1b[2J\x1b[H\x1b[3GXY\x1b[?2K",
		strings.Repeat("\x1b[31m界á cells\x1b[0m\r\n", 12),
	} {
		f.Add([]byte(input), uint8(12), uint8(3))
	}
	f.Fuzz(func(t *testing.T, input []byte, c, r uint8) {
		if len(input) > 1024 {
			input = input[:1024]
		}
		a, b := New(1+int(c%40), 1+int(r%12)), New(1+int(c%40), 1+int(r%12))
		fuzzOps(input, 40, 12, func(data []byte) {
			writeErasedRowsOracle(a, b, data)
			assertSnapshotState(t, a, b)
		}, func(cols, rows int) {
			a.Resize(cols, rows)
			b.Resize(cols, rows)
			a.Compact()
			assertSnapshotState(t, a, b)
		})
	})
}

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
		term.mainWrap = []rowFlags{rowWrap, 0, 0}
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

func TestHistoryASCIICopyMatchesScalar(t *testing.T) {
	rng := rand.New(rand.NewPCG(13, 57))
	for trial := range 10000 {
		length := 1 + rng.IntN(256)
		row := make([]cell, length)
		style := cell{Fg: Color(rng.Uint32()), Bg: Color(rng.Uint32()), A: Attr(rng.Uint32()), E: uint16(rng.Uint32())}
		for i := range row {
			row[i] = style
			row[i].R = rune(1 + rng.IntN(127))
		}
		if boundary := trial % (length + 1); boundary < length {
			c := &row[boundary]
			switch trial % 6 {
			case 0:
				c.R = []rune{0, 128, '界', wideTail, spacerHead, clusterBase}[trial/6%6]
			case 1:
				c.Fg ^= 1 << uint(trial%32)
			case 2:
				c.Bg ^= 1 << uint(trial%32)
			case 3:
				c.A ^= 1 << uint(trial%16)
			case 4:
				c.E ^= 1 << uint(trial%16)
			case 5:
				c.R = rune(rng.Uint32())
			}
		}
		got, want := bytes.Repeat([]byte{0xcc}, length), bytes.Repeat([]byte{0xcc}, length)
		n := copyASCIICells(got, row)
		k := 0
		for k < len(row) && row[k].R > 0 && row[k].R < 128 && styleOf(row[k]) == styleOf(row[0]) {
			want[k] = byte(row[k].R)
			k++
		}
		if n != k || !bytes.Equal(got, want) {
			t.Fatalf("trial %d copied %d cells, want %d", trial, n, k)
		}
	}
}

func TestHistoryASCIICopyWordBoundaries(t *testing.T) {
	for _, fg := range []Color{0, 1, 0x80000000, 0xffffffff} {
		for _, r := range []rune{-1 << 31, wideTail, 0, 1, 127, 128, 1<<31 - 1} {
			for _, delta := range []Color{0, 1, 0xffffffff} {
				row := []cell{{R: 'A', Fg: fg}, {R: r, Fg: fg + delta}, {R: 'Z', Fg: fg}}
				want := 1
				if r > 0 && r < 128 && delta == 0 {
					want = 3
				}
				dst := []byte{0xcc, 0xcc, 0xcc}
				if n := copyASCIICells(dst, row); n != want || dst[0] != 'A' || want == 1 && dst[1] != 0xcc {
					t.Fatalf("foreground=%08x delta=%08x rune=%d: copied %d, want %d", fg, delta, r, n, want)
				}
			}
		}
	}
}

func TestHistoryNarrowedStyledTailExtent(t *testing.T) {
	term := New(3, 2)
	term.Write([]byte("\x1b[?2027hA\x1b[31;4m\x1b]8;;https://narrow\x1b\\☔︎\x1b]8;;\x1b\\\x1b[0m\x1b[K\r\nX\r\nY"))
	rows := term.ScrollbackCells(0)
	if len(rows) != 1 || rows[0][1].Text() != "☔︎" || rows[0][2] != (Cell{Width: 1}) {
		t.Fatalf("history invented a styled tail: %+v", rows)
	}
	term.Resize(2, 2)
	rows = term.ScrollbackCells(0)
	if len(rows) != 1 || rows[0][0].Text() != "A" || rows[0][1].Text() != "☔︎" || rows[0][1].Width != 1 {
		t.Fatalf("history invented an extra reflow row: %+v", rows)
	}
}

func TestHistoryFormattingRestoredContinuation(t *testing.T) {
	source := New(24, 3, WithScrollback(4), WithScrollbackBytes(4096))
	source.Write([]byte("\x1b[?2027h" + strings.Repeat("\x1b[2G\x1b]8;;https://history.invalid/\x1b\\\x1b[31m界 á 👩‍💻 \x1b[1\"qP\x1b[0\"q\x1b[0m\x1b]8;;\x1b\\\r\n", 40)))
	data := source.Snapshot()
	var aReplies, bReplies bytes.Buffer
	a, b := New(1, 1, WithReply(&aReplies)), New(1, 1, WithReply(&bReplies))
	for _, term := range []*Terminal{a, b} {
		if err := term.RestoreSnapshot(data, SnapshotLimits{}); err != nil {
			t.Fatal(err)
		}
		if term.ScrollbackLen() != term.sbMaxLines {
			t.Fatal("fixture did not restore full history")
		}
	}
	for i := range 100 {
		input := fmt.Appendf(nil, "\x1b[31mline-%d 界 á 👩‍💻\x1b[0m\r\n", i)
		if i%7 == 0 {
			input = append(input, "\x1b]133;A\a\x1b]8;;https://future.invalid/\x1b\\linked\x1b]8;;\x1b\\\r\n"...)
		}
		input = append(input, "\x1b[5n\x1b[6n"...)
		a.Write(input)
		for _, c := range input {
			b.Write([]byte{c})
		}
		assertSnapshotState(t, a, b)
		if !bytes.Equal(a.Snapshot(), b.Snapshot()) || !bytes.Equal(aReplies.Bytes(), bReplies.Bytes()) {
			t.Fatal("restored continuation changed history, checkpoint or replies across chunk sizes")
		}
		if i == 25 || i == 75 {
			for _, term := range []*Terminal{a, b} {
				term.Compact()
				term.Resize(18+i%12, 6)
			}
			assertSnapshotState(t, a, b)
		}
	}
	if aReplies.Len() == 0 {
		t.Fatal("fixture produced no query replies")
	}
}

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

func TestScrollbackRecyclesEvictedLines(t *testing.T) {
	for _, opts := range [][]Option{{WithScrollback(8)}, {WithScrollbackBytes(256)}} {
		term := New(40, 2, opts...)
		line := []byte("\x1b[32mINFO\x1b[0m steady bounded output\r\n")
		for range 100 {
			term.Write(line)
		}
		if allocs := testing.AllocsPerRun(100, func() {
			for range 100 {
				term.Write(line)
			}
		}); allocs > 1 {
			t.Fatalf("steady scrolling allocated %.1f times per 100 lines", allocs)
		}
		for i := range 30 {
			term.Write([]byte(fmt.Sprintf("line %02d\r\n", i)))
		}
		got := term.ScrollbackLines()
		if len(got) == 0 || got[len(got)-1] != "line 28" {
			t.Fatalf("scrollback order: %q", got)
		}
		fresh := New(40, 2, opts...)
		fresh.Write(term.Repaint(true))
		if strings.Join(fresh.ScrollbackLines(), "\n") != strings.Join(got, "\n") {
			t.Fatalf("repaint changed recycled history")
		}
	}
}

func TestShortScrollbackLinesDoNotKeepLargeBuffers(t *testing.T) {
	term := New(100, 2, WithScrollback(8), WithScrollbackBytes(1<<20))
	term.Write([]byte(strings.Repeat("x", 8192) + "\r\n"))
	for range 40 {
		term.Write([]byte("tiny\r\n"))
	}
	for _, line := range term.sb[term.sbStart:] {
		if cap(line) > 128 {
			t.Fatalf("short line retained %d-byte buffer", cap(line))
		}
	}
}

func TestScrollClear(t *testing.T) {
	cases := []struct {
		name       string
		cols, rows int
		input      string
		history    []string
		x, y       int
	}{
		{"partial", 8, 4, "A\r\nB", []string{"A", "B"}, 0, 0},
		{"below-content", 8, 4, "A\x1b[4;4H", []string{"A"}, 3, 2},
		{"empty", 8, 4, "\x1b[3;4H", nil, 3, 2},
		{"spaces", 8, 4, "  \r\n", []string{""}, 0, 0},
		{"soft-wrap", 3, 4, "abcdefg", []string{"abcdefg"}, 0, 0},
		{"wide-wrap", 3, 4, "ab漢", []string{"ab漢"}, 0, 0},
		{"colored-blanks", 8, 4, "\x1b[44m\x1b[2J", []string{"", "", "", ""}, 0, 0},
		{"protected", 8, 4, "\x1b[1\"qP\x1bVQ", []string{"PQ"}, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			term := New(c.cols, c.rows)
			term.Write([]byte(c.input))
			pen := term.cur.pen
			term.ClearDirty()
			term.Write([]byte("\x1b[22J"))
			if got := term.ScrollbackLines(); !slices.Equal(got, c.history) {
				t.Fatalf("history%q want%q", got, c.history)
			}
			if got := strings.TrimSpace(strings.Join(term.Lines(), "\n")); got != "" {
				t.Fatalf("screen not cleared:%q", got)
			}
			if cur := term.Cursor(); cur.X != c.x || cur.Y != c.y || cur.PendingWrap {
				t.Fatalf("cursor%+v want%d,%d", cur, c.x, c.y)
			}
			if term.cur.pen != pen {
				t.Fatalf("pen changed:%+v want%+v", term.cur.pen, pen)
			}
			for y := range c.rows {
				for x := range c.cols {
					if got := term.Cell(x, y); got.Style.Bg != 0 || got.Rune != 0 {
						t.Fatalf("cell%d,%d:%+v", x, y, got)
					}
				}
			}
			if len(c.history) > 0 && !term.Dirty() {
				t.Fatal("clear did not mark damage")
			}
			checkRepaint(t, term)
			repaint := term.Repaint(true)
			term.Write([]byte("\x1b[22J"))
			if got := term.Repaint(true); !bytes.Equal(got, repaint) {
				t.Fatal("clearing an empty screen changed state")
			}
		})
	}
}

func TestScrollClearMarginsAlternateAndBounds(t *testing.T) {
	t.Run("margins", func(t *testing.T) {
		term := feed(8, 4, "A\r\nB\r\nC\r\nD\x1b[2;3r\x1b[?69h\x1b[3;6s\x1b[22J")
		if got := term.ScrollbackLines(); !slices.Equal(got, []string{"A", "B", "C", "D"}) {
			t.Fatalf("history%q", got)
		}
		if term.top != 1 || term.bot != 2 || term.left != 2 || term.right != 5 || !term.lrmm {
			t.Fatal("margins changed")
		}
		checkRepaint(t, term)
	})
	t.Run("alternate", func(t *testing.T) {
		term := feed(8, 4, "MAIN\x1b[?1049h\x1b[HALT\x1b[22J")
		if term.ScrollbackLen() != 0 || strings.TrimSpace(strings.Join(term.Lines(), "\n")) != "" {
			t.Fatal("alternate screen not cleared")
		}
		term.Write([]byte("\x1b[?1049l"))
		if term.Line(0) != "MAIN" {
			t.Fatal("main screen changed")
		}
		checkRepaint(t, term)
	})
	t.Run("bounded", func(t *testing.T) {
		term := New(8, 4, WithScrollback(1))
		term.Write([]byte("A\r\nB\x1b[22J"))
		if got := term.ScrollbackLines(); !slices.Equal(got, []string{"B"}) {
			t.Fatalf("history budget not honored:%q", got)
		}
		checkRepaint(t, term)
	})
	t.Run("no-history", func(t *testing.T) {
		term := New(8, 4, WithScrollback(0))
		term.Write([]byte("A\r\nB\x1b[22J"))
		if term.ScrollbackLen() != 0 || term.Cursor().X != 0 || term.Cursor().Y != 0 {
			t.Fatal("history-disabled clear")
		}
		checkRepaint(t, term)
	})
	t.Run("selective", func(t *testing.T) {
		term := feed(8, 4, "\x1b[1\"qP\x1b[?22J")
		if term.Line(0) != "" || !slices.Equal(term.ScrollbackLines(), []string{"P"}) {
			t.Fatal("selective scroll-and-clear did not move protected content")
		}
		checkRepaint(t, term)
	})
	t.Run("saved-cursor", func(t *testing.T) {
		term := feed(8, 4, "\x1b[3;4H\x1b[31m\x1b7\x1b[HA\x1b[22J\x1b8X")
		if term.Line(2) != "   X" || term.Cell(3, 2).Style.Fg != IndexedColor(1) {
			t.Fatalf("saved cursor changed: %q, %+v", term.Lines(), term.Cell(3, 2))
		}
		checkRepaint(t, term)
	})
	t.Run("full-history-wrap-and-background", func(t *testing.T) {
		term := New(3, 2, WithScrollback(2))
		term.Write([]byte("old\r\nold\r\n\x1b[44mabcdefg\x1b[22J"))
		if got := term.ScrollbackLines(); !slices.Equal(got, []string{"old", "abcdefg"}) {
			t.Fatalf("history %q", got)
		}
		term.Write([]byte("X"))
		if got := term.Cell(0, 0); got.Rune != 'X' || got.Style.Bg != IndexedColor(4) {
			t.Fatalf("subsequent background changed: %+v", got)
		}
		for _, size := range [][2]int{{1, 4}, {6, 2}, {3, 2}} {
			term.Resize(size[0], size[1])
			if msg := term.checkInvariants(); msg != "" {
				t.Fatal(msg)
			}
			checkRepaint(t, term)
		}
	})
}

// DECRC can already leave the cursor outside an enabled origin region.
// Scroll-and-clear reaches the same state when it homes a scrolled-away
// cursor. Check physical coordinates and subsequent output directly.
func TestScrollClearOutsideOriginReplay(t *testing.T) {
	original := feed(8, 4, "A\x1b[2;3r\x1b[?69h\x1b[3;6s\x1b[?6hB\x1b[22J")
	if cur := original.Cursor(); cur.X != 0 || cur.Y != 0 || !original.cur.origin {
		t.Fatalf("scroll-and-clear cursor %+v, origin %v", cur, original.cur.origin)
	}
	restored := New(8, 4)
	restored.Write(original.Repaint(true))
	if cur := restored.Cursor(); cur != original.Cursor() || !restored.cur.origin {
		t.Fatalf("repaint moved cursor: %+v, origin %v", cur, restored.cur.origin)
	}
	original.Write([]byte("X"))
	restored.Write([]byte("X"))
	if original.Line(0) != "X" || restored.Line(0) != "X" || original.Cursor() != restored.Cursor() {
		t.Fatalf("repaint changed continuation: original %q, restored %q", original.Lines(), restored.Lines())
	}
}

func TestScrollClearContinuation(t *testing.T) {
	for _, cols := range []int{1, 2, 3, 8} {
		for _, rows := range []int{1, 2, 4} {
			t.Run(fmt.Sprintf("%dx%d", cols, rows), func(t *testing.T) {
				input := []byte("ÁB\r\n漢\x1b[22JX\x1b[2;3H\x1b[22JY")
				for cut := 0; cut <= len(input); cut++ {
					a, b := New(cols, rows), New(cols, rows)
					a.Write(input[:cut])
					b.Write(input[:cut])
					b.Compact()
					restored := New(cols, rows)
					restored.Write(b.Repaint(true))
					if restored.Cursor() != a.Cursor() {
						t.Fatalf("cut %d: cursor changed across checkpoint", cut)
					}
					a.Write(input[cut:])
					b.Write(input[cut:])
					restored.Write(input[cut:])
					assertContinuationState(t, a, b, "compact+scroll-clear")
					assertContinuationState(t, a, restored, "scroll-clear checkpoint")
					if restored.Cursor() != a.Cursor() {
						t.Fatalf("cut %d: cursor changed after continuation", cut)
					}
					if msg := a.checkInvariants(); msg != "" {
						t.Fatal(msg)
					}
					checkRepaint(t, a)
					a.Resize(2, 3)
					restored.Resize(2, 3)
					assertContinuationState(t, a, restored, "scroll-clear checkpoint resize")
				}
			})
		}
	}
}

func TestScrollClearPreservesREP(t *testing.T) {
	input := []byte("AB\x1b[22J\x1b[3b")
	for cut := 0; cut <= len(input); cut++ {
		term := New(8, 4)
		term.Write(input[:cut])
		term.Compact()
		term.Write(input[cut:])
		if term.Line(0) != "BBB" || term.Cursor().X != 3 || term.Cursor().Y != 0 {
			t.Fatalf("cut %d: REP changed: %q, %+v", cut, term.Lines(), term.Cursor())
		}
	}
}
