package phosphor

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
	"unsafe"
)

func TestGraphemeStreamingPresentation(t *testing.T) {
	cases := []struct {
		name, text, rendered string
		width                int
	}{
		{"family", "👨‍👩‍👧", "👨‍👩‍👧", 2},
		{"skin tone", "👋🏿", "👋🏿", 2},
		{"text modifier base", "✍🏿", "✍🏿", 1},
		{"flag", "🇺🇸", "🇺🇸", 2},
		{"odd regional indicator", "🇺🇸🇨", "🇺🇸🇨", 4},
		{"emoji presentation", "❤️", "❤️", 2},
		{"text presentation", "☔︎", "☔︎", 1},
		{"keycap", "1️⃣", "1️⃣", 2},
		{"Indic conjunct", "क्‍ष", "क्‍ष", 2},
		{"Hangul jamo", "각", "각", 2},
		{"combining Latin", "é", "é", 1},
		{"invalid selector", "x️", "x", 1},
		{"invalid modifier", "\"🏿\"", "\"🏿\"", 4},
		{"ASCII after Prepend", "_؀A", "_؀A", 3},
		{"non-ASCII after Prepend", "؀å", "؀å", 2},
		{"two Prepends", "؀؀", "؀؀", 1},
		{"ignored selector leaves boundary state", "n️̃", "ñ", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			input := []byte("\x1b[?2027h" + c.text)
			for cut := 0; cut <= len(input); cut++ {
				term := New(20, 3)
				term.Write(input[:cut])
				term.Write(input[cut:])
				if got := term.Line(0); got != c.rendered {
					t.Fatalf("cut%d text%q want%q", cut, got, c.rendered)
				}
				if got := term.Cursor().X; got != c.width {
					t.Fatalf("cut%d cursor%d want%d", cut, got, c.width)
				}
				assertGraphemeInvariants(t, term)
				checkRepaint(t, term)
			}
		})
	}
}

func TestGraphemeWideningKeepsNativePens(t *testing.T) {
	for _, wrap := range []bool{false, true} {
		t.Run(fmt.Sprint(wrap), func(t *testing.T) {
			term := New(3, 3)
			prefix := "\x1b[?2027h"
			if wrap {
				prefix += "\x1b[2C"
			}
			term.Write([]byte(prefix + "\x1b]8;;https://base\x1b\\\x1b[31m❤\x1b]8;;https://selector\x1b\\\x1b[32m️"))
			y := 0
			wantHead, wantURI := idx(1), "https://base"
			if wrap {
				y, wantHead, wantURI = 1, idx(2), "https://selector"
			}
			assertPens := func(head, tail Cell) {
				t.Helper()
				if head.Style.Fg != wantHead || head.Link.URI != wantURI || tail.Style.Fg != idx(2) || tail.Link.URI != "https://selector" {
					t.Fatalf("base and selector pens: head%+v tail%+v", head, tail)
				}
			}
			assertPens(term.Cell(0, y), term.Cell(1, y))
			if wrap {
				spacer := term.Cell(2, 0)
				if spacer.Rune != 0 || spacer.Style.Fg != idx(2) || spacer.Link.URI != "https://selector" {
					t.Fatalf("wrapped spacer lost the current pen:%+v", spacer)
				}
			}
			replay := New(3, 3)
			replay.Write(term.Repaint(false))
			assertPens(replay.Cell(0, y), replay.Cell(1, y))
			term.Write([]byte("\r\nX\r\nY\r\nZ"))
			for i := range term.ScrollbackLen() {
				for _, row := range term.ScrollbackCells(i) {
					for x, c := range row {
						if c.Rune == '❤' {
							assertPens(c, row[x+1])
							return
						}
					}
				}
			}
			t.Fatal("wide cluster did not reach history")
		})
	}
}

func TestGraphemeModeTransitionsAndControls(t *testing.T) {
	cases := []struct {
		input, want string
		width       int
	}{
		{"\x1b[?2027la\u200b\x1b[?2027h́", "a\u200b́", 1},
		{"\x1b[?2027l👨‍\x1b[?2027h👩‍👧", "👨‍👩‍👧", 2},
		{"\x1b[?2027h❤\x1b[31m️", "❤️", 2},
		{"\x1b[?2027h👨\x1b]8;;https://example.com\x1b\\‍👩", "👨‍👩", 2},
		{"\x1b[?2027h❤\x1b[?2027l️\x1b[?2027hX", "❤️X", 2},
		{"\x1b[?2027h☔︎\x1b[?2027l☔︎", "☔︎☔︎", 3},
	}
	for _, c := range cases {
		term := New(20, 3)
		// One byte per write exercises parser, UTF-8 and grapheme boundaries together.
		for i := range len(c.input) {
			term.Write([]byte{c.input[i]})
		}
		if got := term.Line(0); got != c.want {
			t.Fatalf("%q =>%q want%q", c.input, got, c.want)
		}
		if got := term.Cursor().X; got != c.width {
			t.Fatalf("%q cursor%d want%d", c.input, got, c.width)
		}
		assertGraphemeInvariants(t, term)
	}
}

func TestREPRemembersIndependentScalar(t *testing.T) {
	for _, c := range []struct{ input, want string }{
		{"á\x1b[2b", "áaa"},
		{"\x1b[?2027h👨‍👩‍👧\x1b[2b", "👨‍👩‍👧👨👨"},
		{"\x1b[?2027h❤️\x1b[2b", "❤️❤❤"},
		{"\x1b[?2027h☔︎\x1b[2b", "☔︎☔☔"},
		{"\x1b(0q\x1b(B\x1b[2b", "─qq"},
		{"q\x1b(0\x1b[2b", "q──"},
	} {
		term := New(20, 3)
		term.Write([]byte(c.input))
		if got := term.Line(0); got != c.want {
			t.Fatalf("%q =>%q want%q", c.input, got, c.want)
		}
		assertGraphemeInvariants(t, term)
	}
	term := New(20, 3)
	term.Write([]byte("á\x1b[2K"))
	term.Compact()
	if term.x != nil {
		t.Fatal("erased cluster retained despite REP storing only its scalar")
	}
	term.Write([]byte("\x1b[2b"))
	if term.Line(0) != " aa" {
		t.Fatalf("REP after Compact:%q", term.Line(0))
	}
}

func TestPendingWrapIsIndependentOfDECAWM(t *testing.T) {
	for _, input := range []string{
		"\x1b[?2027h\x1b[?7lA❤️",
		"abc\x1b7\x1b[?7l\x1b[HX\x1b8",
		"\x1b[?2027h\x1b[?7lA❤️\x1b7\x1b[HX\x1b8",
	} {
		term := New(3, 3)
		term.Write([]byte(input))
		if term.cur.x != 2 || !term.cur.wrapPending || term.autowrap {
			t.Fatalf("%q cursor%+v autowrap%v", input, term.cur, term.autowrap)
		}
		assertGraphemeInvariants(t, term)
		replay := New(3, 3)
		replay.Write(term.Repaint(false))
		if !replay.cur.wrapPending || replay.autowrap {
			t.Fatalf("portable replay lost nonwrapping pending state:%+v", replay.cur)
		}
		term.Write([]byte("X"))
		if !term.cur.wrapPending || term.cur.y != 0 || term.Cell(2, 0).Rune != 'X' {
			t.Fatalf("edge print did not renew pending wrap without wrapping: cursor%+v screen%q", term.cur, term.Lines())
		}
		assertGraphemeInvariants(t, term)
	}
}

func TestCodepointWidthExamples(t *testing.T) {
	for _, c := range []struct {
		r     rune
		width int
	}{
		{'A', 1}, {'界', 2}, {0x1f1fa, 2}, {0x00ad, 1}, {0x070f, 1},
		{0x1160, 0}, {0x11ff, 0}, {0x3164, 0}, {0xe0fff, 0},
		{0x2e3b, 2}, {0x1faea, 2}, {0x0301, 0}, {0xfe0f, 0},
		{-1, 1}, {0xd800, 1}, {0x110000, 1},
	} {
		if got := CodepointWidth(c.r); got != c.width {
			t.Fatalf("%U width%d want%d", c.r, got, c.width)
		}
	}
}

func TestGraphemeHistoryNarrowReflow(t *testing.T) {
	for _, cols := range []int{1, 2, 3, 5} {
		t.Run(fmt.Sprint(cols), func(t *testing.T) {
			term := New(3, 2)
			term.Write([]byte("\x1b[?2027hA☔︎\r\nX\r\nY"))
			term.Resize(cols, 2)
			cells := term.ScrollbackCells(0)
			var text strings.Builder
			found := false
			for _, row := range cells {
				for _, c := range row {
					if c.Rune != 0 {
						text.WriteRune(c.Rune)
						text.WriteString(c.Combining)
					}
					if c.Rune == '☔' {
						found = true
						if c.Width != 1 {
							t.Fatal("narrowed history became wide")
						}
					}
				}
			}
			if !found || text.String() != "A☔︎" {
				t.Fatalf("history text:%q cells:%+v", text.String(), cells)
			}
			term.Resize(cols, 8)
			assertGraphemeInvariants(t, term)
			if !strings.Contains(strings.Join(term.Lines(), ""), "☔︎") {
				t.Fatalf("history restore lost narrowed emoji:%q", term.Lines())
			}
		})
	}
}

func TestGraphemePortableRepaintEdgeLimit(t *testing.T) {
	term := New(3, 4)
	term.Write([]byte("\x1b[?2027hA☔︎"))
	term.Resize(2, 4)
	if term.Cell(1, 0).Rune != '☔' || term.Cell(1, 0).Width != 1 {
		t.Fatalf("cell reflow lost narrowed geometry:%+v", term.Cell(1, 0))
	}
	replay := New(2, 4)
	replay.Write(term.Repaint(false))
	// Scalar VT paints the wide umbrella before seeing VS15, so a portable
	// receiver wraps it first. Exact internal persistence uses Snapshot;
	// Repaint cannot promise this geometry on arbitrary terminal hosts.
	if reflect.DeepEqual(term.Lines(), replay.Lines()) {
		t.Fatal("edge limit changed; update the documented portable VT boundary")
	}
	if replay.Cell(0, 1).Rune != '☔' {
		t.Fatalf("unexpected scalar VT boundary:%q", replay.Lines())
	}
}

func TestGraphemeDirtyBoundAndIdle(t *testing.T) {
	if unsafe.Sizeof(Terminal{}) > 560 || unsafe.Sizeof(cell{}) != 16 {
		t.Fatalf("fixed state grew:%d cell%d", unsafe.Sizeof(Terminal{}), unsafe.Sizeof(cell{}))
	}
	for _, mode := range []string{"", "\x1b[?2027h"} {
		term := New(80, 24)
		term.Write([]byte(mode))
		n := testing.AllocsPerRun(100, func() { term.Write([]byte("\ruser@host:~$ ")) })
		if n != 0 || term.x != nil {
			t.Fatalf("ASCII prompt allocated%.1f, extra%v", n, term.x)
		}
	}
	term := New(8, 3)
	term.Write([]byte("\x1b[?2027h👨"))
	term.ClearDirty()
	for _, s := range []string{"‍", "👩", "‍", "👧"} {
		term.Write([]byte(s))
		if !term.RowDirty(0) {
			t.Fatalf("%q changed without damage", s)
		}
		term.ClearDirty()
	}
	term.Write([]byte(strings.Repeat("́", 1000)))
	if len(term.cellString(term.cellAt(0, 0))) > maxCluster {
		t.Fatal("unbounded cluster")
	}
	assertGraphemeInvariants(t, term)
	term.ClearDirty()
	term.Write([]byte("️"))
	if term.RowDirty(0) {
		t.Fatal("ignored selector marked row dirty")
	}
}

func assertGraphemeInvariants(t *testing.T, term *Terminal) {
	t.Helper()
	if msg := term.checkInvariants(); msg != "" {
		t.Fatal(msg)
	}
	if term.x == nil {
		return
	}
	for _, c := range term.x.clusters {
		count := utf8.RuneCountInString(c.text)
		if count > maxClusterRunes || !utf8.ValidString(c.text) || c.width < 1 || c.width > 2 {
			t.Fatalf("invalid cluster:%+v", c)
		}
		for i := range 128 {
			if c.joinModes.has(i) && (i == 0 || i >= count) {
				t.Fatalf("join mode outside scalar boundary:%+v", c)
			}
		}
	}
}

func FuzzGraphemeTransitions(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, uint8(4))
	f.Add([]byte{0, 11, 3, 12, 13, 14, 5, 0xf3, 10, 2}, uint8(2))
	tokens := []string{"\x1b[?2027h", "👨", "‍", "👩", "👧", "❤", "️", "☔", "︎", "X", "\x1b[?2027l", "🇺", "🇸", "🏿", "👋", "\r\n", "\x1b[H", "\x1b[2J", "\x1b[?1049h", "\x1b[?1049l", "\x1b[31m", "́", "\x1b[?7l", "\x1b[?7h", "\x1b[4h", "\x1b[4l", "\x1b[b", "ᄀ", "ᅡ", "ᆨ", "\x1b[?69h\x1b[2;3s", "\x1b[?69l"}
	f.Fuzz(func(t *testing.T, sel []byte, width uint8) {
		if len(sel) > 256 {
			sel = sel[:256]
		}
		term := New(int(width%15)+1, 4, WithScrollback(20))
		for _, b := range sel {
			if b >= 0xf0 {
				term.Resize(int(b%15)+1, 4)
			} else {
				term.Write([]byte(tokens[int(b)%len(tokens)]))
			}
			assertGraphemeInvariants(t, term)
			if b%7 == 0 {
				term.Compact()
				assertGraphemeInvariants(t, term)
			}
		}
	})
}
