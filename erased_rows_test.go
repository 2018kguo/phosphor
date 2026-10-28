package phosphor

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

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
