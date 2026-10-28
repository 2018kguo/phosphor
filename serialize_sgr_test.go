package phosphor

import (
	"fmt"
	"testing"
)

func TestForegroundHistoryUsesCanonicalSGR(t *testing.T) {
	for index := 0; index < 16; index++ {
		code := 30 + index
		if index >= 8 {
			code += 52
		}
		term := New(40, 2)
		term.Write(fmt.Appendf(nil, "\x1b[%dmcolor\x1b[m default\r\none\r\ntwo", code))
		got := string(term.sb[term.sbStart])
		want := fmt.Sprintf("\x1b[0;%dmcolor\x1b[0m default", code)
		if got != want {
			t.Fatalf("palette %d history = %q, want %q", index, got, want)
		}
		cells := term.ScrollbackCells(0)[0]
		if cells[0].Style.Fg != IndexedColor(uint8(index)) || cells[6].Style.Fg != DefaultColor {
			t.Fatalf("palette %d lost the style boundary: %+v", index, cells)
		}
	}
}

func TestSGRForegroundFastPathBoundaries(t *testing.T) {
	term := New(10, 2)
	underline := term.internExtra(cellExtra{underline: IndexedColor(5)})
	for _, tc := range []struct {
		name string
		cell cell
		want string
	}{
		{"default", cell{}, "\x1b[0m"},
		{"protected", cell{A: AttrProtected, Fg: IndexedColor(1)}, "\x1b[0;31m"},
		{"bold", cell{A: AttrBold, Fg: IndexedColor(1)}, "\x1b[0;1;31m"},
		{"background", cell{Fg: IndexedColor(1), Bg: IndexedColor(4)}, "\x1b[0;31;44m"},
		{"indexed", cell{Fg: IndexedColor(16)}, "\x1b[0;38;5;16m"},
		{"rgb", cell{Fg: RGBColor(1, 2, 3)}, "\x1b[0;38;2;1;2;3m"},
		{"underline", cell{Fg: IndexedColor(1), E: underline}, "\x1b[0;31;58;5;5m"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := string(term.appendSGR(nil, tc.cell)); got != tc.want {
				t.Fatalf("SGR = %q, want %q", got, tc.want)
			}
		})
	}
}
