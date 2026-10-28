package phosphor

import (
	"fmt"
	"strings"
	"testing"
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
