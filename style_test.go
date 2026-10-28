package phosphor

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"unsafe"
)

func TestExtendedColorMalformedAndContinuation(t *testing.T) {
	for _, code := range []string{"38", "48", "58"} {
		for _, tc := range []struct {
			suffix    string
			attrs     Attr
			underline Underline
		}{
			{"", 0, 0}, {";5", AttrBlink, 0}, {";2;1;2", AttrBold | AttrFaint, 0},
			{";3;4", AttrItalic, UnderlineSingle}, {":5", AttrBlink, 0},
			{":", 0, 0}, {":5:", 0, 0}, {":2:", 0, 0}, {":2:1:2", 0, 0},
			{":2:1:2:3:4:5", 0, 0}, {":3:4", 0, 0},
		} {
			t.Run(code+tc.suffix, func(t *testing.T) {
				in := []byte("\x1b[38;5;9;48;5;10;58;5;11m\x1b[" + code + tc.suffix + "mA\x1bP$qm\x1b\\")
				want := Style{Fg: IndexedColor(9), Bg: IndexedColor(10), UnderlineColor: IndexedColor(11), Attrs: tc.attrs, Underline: tc.underline}
				for cut := 0; cut <= len(in); cut++ {
					var reply bytes.Buffer
					term := New(8, 2, WithReply(&reply))
					term.Write(in[:cut])
					var restored Terminal
					if err := restored.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
						t.Fatalf("cut %d: %v", cut, err)
					}
					painted := New(8, 2)
					painted.Write(term.Repaint(true))
					for _, emu := range []*Terminal{term, &restored, painted} {
						emu.Write(in[cut:])
						if got := emu.Cell(0, 0).Style; got != want {
							t.Fatalf("cut %d: got %+v, want %+v", cut, got, want)
						}
					}
					// The response must describe underline color even though
					// Ghostty currently omits it from its DECRQSS response.
					if !bytes.Contains(reply.Bytes(), []byte(";58;5;11")) {
						t.Fatalf("cut %d: incomplete style reply %q", cut, reply.Bytes())
					}
				}
			})
		}
	}
}

func TestUnderlineColorForms(t *testing.T) {
	for _, tc := range []struct {
		sgr  string
		want Color
	}{
		{"58;5;0", IndexedColor(0)}, {"58;5;255", IndexedColor(255)},
		{"58:5:3", IndexedColor(3)}, {"58;2;1;2;3", RGBColor(1, 2, 3)},
		{"58:2::4:5:6", RGBColor(4, 5, 6)}, {"58:2:7:8:9", RGBColor(7, 8, 9)},
	} {
		t.Run(tc.sgr, func(t *testing.T) {
			term := New(8, 2)
			term.Write([]byte("\x1b[" + tc.sgr + "mA\x1b[4m漢"))
			for x := range 3 {
				c := term.Cell(x, 0)
				if c.Style.UnderlineColor != tc.want {
					t.Fatalf("cell %d: %+v, want color %v", x, c, tc.want)
				}
			}
			if term.Cell(0, 0).Style.Underline != UnderlineNone || term.Cell(1, 0).Style.Underline != UnderlineSingle {
				t.Fatal("color changed the underline style")
			}
			checkRepaint(t, term)
		})
	}
}

func TestUnderlineColorLinkAndResets(t *testing.T) {
	term := New(20, 2)
	term.Write([]byte("\x1b[58;5;1m\x1b]8;id=x;https://a\aA\x1b[24mB\x1b]8;;\aC" +
		"\x1b]8;id=y;https://b\aD\x1b[59mE\x1b[58;5;2mF\x1b[0mG\x1b[58;5;3mH\x1b[mI"))
	for x, color := range []Color{IndexedColor(1), IndexedColor(1), IndexedColor(1), IndexedColor(1), 0, IndexedColor(2), 0, IndexedColor(3), 0} {
		c := term.Cell(x, 0)
		uri := "https://b"
		if x < 2 {
			uri = "https://a"
		} else if x == 2 {
			uri = ""
		}
		if c.Style.UnderlineColor != color || c.Link.URI != uri {
			t.Fatalf("cell %d: %+v, want color %v and URI %q", x, c, color, uri)
		}
	}
	checkRepaint(t, term)
	term.Write([]byte("\x1b[58;5;5;48;5;6m\x1b[H\x1b[3X"))
	for x := range 3 {
		if c := term.Cell(x, 0); c.Style.UnderlineColor != 0 || c.Link != (Hyperlink{}) || c.Style.Bg != IndexedColor(6) {
			t.Fatalf("erase kept rare attributes: %+v", c)
		}
	}
	term.Write([]byte("\x1b[!pZ"))
	if c := term.Cell(0, 0); c.Style != (Style{}) || c.Link != (Hyperlink{}) {
		t.Fatalf("soft reset kept rare attributes: %+v", c)
	}
}

func TestUnderlineColorSavedPensAndCheckpoint(t *testing.T) {
	term := New(12, 5)
	term.Write([]byte("\x1b[2;3H\x1b[58;5;1m\x1b]8;;https://main\a\x1b7" +
		"\x1b[58;5;2m\x1b]8;;https://1049\a\x1b[?1049h" +
		"\x1b[3;4H\x1b[58;5;3m\x1b]8;;https://alt\a\x1b7"))
	for i := range 100 {
		term.Write(fmt.Appendf(nil, "\x1b[58;2;1;2;%dm\x1b]8;;https://dead/%d\a", i, i))
	}
	term.Compact()
	term.Resize(14, 6)
	var restored Terminal
	if err := restored.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
		t.Fatal(err)
	}
	for _, trace := range []struct {
		in    string
		x, y  int
		color Color
		uri   string
	}{
		{"\x1b8A", 3, 2, IndexedColor(3), "https://dead/99"},
		{"\x1b[?1049lB", 2, 1, IndexedColor(2), ""},
		{"\x1b8M", 2, 1, IndexedColor(2), ""},
	} {
		for _, emu := range []*Terminal{term, &restored} {
			emu.Write([]byte(trace.in))
			if c := emu.Cell(trace.x, trace.y); c.Style.UnderlineColor != trace.color || c.Link.URI != trace.uri {
				t.Fatalf("restored pen after %q: %+v", trace.in, c)
			}
		}
		assertSnapshotState(t, term, &restored)
	}
	checkRepaint(t, term)
}

func TestUnderlineColorHistoryResize(t *testing.T) {
	a := New(10, 3)
	for i := range 30 {
		a.Write(fmt.Appendf(nil, "\x1b[4:3;58;2;%d;2;3m\x1b]8;;https://%d\a漢á\r\n", i, i))
	}
	var b Terminal
	if err := b.RestoreSnapshot(a.Snapshot(), SnapshotLimits{}); err != nil {
		t.Fatal(err)
	}
	b.Compact()
	for _, size := range [][2]int{{4, 5}, {20, 20}, {8, 3}} {
		a.Resize(size[0], size[1])
		b.Resize(size[0], size[1])
		b.Compact()
		assertSnapshotState(t, a, &b)
		checkRepaint(t, a)
	}
}

func TestRareAttributesLimit(t *testing.T) {
	term := New(0xffff, 1)
	term.main[0] = make([]cell, term.cols)
	x := term.ext()
	x.rare = make([]cellExtra, term.cols)
	for i := range x.rare {
		x.rare[i] = cellExtra{link: ";https://kept", underline: RGBColor(1, uint8(i>>8), uint8(i))}
		term.main[0][i] = cell{R: 'a', E: uint16(i + 1)}
	}
	term.cur.pen.E = 0xffff
	before := term.extraAt(term.cur.pen.E)
	term.setUnderlineColor(RGBColor(2, 0, 0))
	if got := term.extraAt(term.cur.pen.E); got != before || len(x.rare) != 0xffff {
		t.Fatalf("saturated update changed existing attributes: %+v", got)
	}
	term.setUnderlineColor(x.rare[0].underline)
	if got := term.extraAt(term.cur.pen.E); got != x.rare[0] {
		t.Fatalf("saturated pool failed to reuse older pair: %+v", got)
	}
	term.main[0] = nil
	term.setUnderlineColor(RGBColor(2, 0, 0))
	if got := term.extraAt(term.cur.pen.E); got.underline != RGBColor(2, 0, 0) || got.link != before.link || len(x.rare) > 2 {
		t.Fatalf("collection failed to recover capacity: %+v, %d entries", got, len(x.rare))
	}
}

func TestRareAttributesLeaveOrdinaryStorageSmall(t *testing.T) {
	term := New(80, 24)
	term.Write([]byte("\x1b[59m\x1b[m\x1b]8;;\a$ "))
	if term.x != nil || unsafe.Sizeof(cell{}) != 16 || unsafe.Sizeof(Terminal{}) > 560 {
		t.Fatalf("ordinary terminal storage grew: extra=%v cell=%d terminal=%d", term.x != nil, unsafe.Sizeof(cell{}), unsafe.Sizeof(Terminal{}))
	}
}

func TestSingleSGRMatchesGenericParameterDispatch(t *testing.T) {
	for _, prelude := range []string{
		"",
		"\x1b[1;2;3;4:3;48;2;12;34;56m",
		"\x1b[1\"q\x1b]8;id=live;https://example.test\x1b\\\x1b[58;2;1;2;3m",
	} {
		for parameter := 0; parameter <= 107; parameter++ {
			fast, generic := New(20, 4), New(20, 4)
			fast.Write([]byte(prelude))
			generic.Write([]byte(prelude))
			fast.Write(fmt.Appendf(nil, "\x1b[%dm", parameter))
			// Unknown SGR999 leaves the pen unchanged and forces the
			// multi-parameter route for the following operation.
			generic.Write(fmt.Appendf(nil, "\x1b[999;%dm", parameter))
			if fast.cur.pen != generic.cur.pen {
				t.Fatalf("prelude %q SGR%d pen = %+v, generic %+v", prelude, parameter, fast.cur.pen, generic.cur.pen)
			}
			for _, term := range []*Terminal{fast, generic} {
				term.Write([]byte("same\x1b[H"))
			}
			if !bytes.Equal(fast.Snapshot(), generic.Snapshot()) {
				t.Fatalf("prelude %q SGR%d: continued state differs", prelude, parameter)
			}
		}
	}
}

func TestSingleSGRColonGroupsStayOnGenericPath(t *testing.T) {
	for _, input := range []string{"\x1b[0:m", "\x1b[31:m", "\x1b[0::m", "\x1b[31:1m"} {
		term := New(20, 4)
		term.Write([]byte("\x1b[1;34m"))
		before := term.cur.pen
		term.Write([]byte(input))
		if term.cur.pen != before {
			t.Fatalf("%q changed pen %+v to %+v", input, before, term.cur.pen)
		}
	}
}

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

func TestKittyColorsSharePalette(t *testing.T) {
	base := DefaultPalette()
	base.Colors[1], base.Foreground, base.Background, base.Cursor = RGB{1, 2, 3}, RGB{4, 5, 6}, RGB{7, 8, 9}, RGB{10, 11, 12}
	var replies bytes.Buffer
	term := New(8, 3, WithPalette(base), WithReply(&replies))
	term.ClearDirty()
	term.Write([]byte("\x1b]21;1=rgb:12/34/56;foreground=red;background=rgbi:0/0.5/1;cursor=#aabbcc\x1b\\"))
	p := term.Palette()
	if p.Colors[1] != (RGB{0x12, 0x34, 0x56}) || p.Foreground != (RGB{255, 0, 0}) || p.Background != (RGB{0, 127, 255}) || p.Cursor != (RGB{170, 187, 204}) || !term.Dirty() {
		t.Fatalf("palette changes missing: %v %v %v %v dirty=%v", p.Colors[1], p.Foreground, p.Background, p.Cursor, term.Dirty())
	}
	// Both adapters read and write the same entries, including query order
	// within a batch and canonical index formatting.
	term.Write([]byte("\x1b]4;1;rgb:01/02/03\a\x1b]21;001=?;1=blue;1=?;foreground=?;cursor=?\a"))
	if got, want := replies.String(), "\x1b]21;1=rgb:01/02/03;1=rgb:00/00/ff;foreground=rgb:ff/00/00;cursor=rgb:aa/bb/cc\a"; got != want {
		t.Fatalf("reply %q, want %q", got, want)
	}
	term.Write([]byte("\x1b]21;1=;foreground;background=;cursor\x1b\\"))
	if term.Palette() != base {
		t.Fatal("kitty reset did not restore base palette")
	}
	term.Write([]byte("\x1b]21;1=red;foreground=blue\x1b\\\x1b]104;1\a\x1b]110\a"))
	if term.Palette() != base {
		t.Fatal("xterm reset did not remove kitty changes")
	}
}

func TestKittyColorCheckpointAndBounds(t *testing.T) {
	input := []byte("\x1b]21;1=red;foreground=rgbi:1/0.5/0;cursor=#aabbcc\x1b\\\x1b]21;1=?;foreground=?;cursor=?\a")
	for cut := 0; cut <= len(input); cut++ {
		var aReply, bReply bytes.Buffer
		a := New(8, 3, WithReply(&aReply))
		a.Write(input[:cut])
		b := New(1, 1, WithReply(&bReply))
		if err := b.RestoreSnapshot(a.Snapshot(), SnapshotLimits{}); err != nil {
			t.Fatalf("cut %d: %v", cut, err)
		}
		aReply.Reset()
		a.Write(input[cut:])
		b.Write(input[cut:])
		b.Compact()
		assertSnapshotState(t, a, b)
		if !bytes.Equal(aReply.Bytes(), bReply.Bytes()) {
			t.Fatalf("cut %d: reply changed across snapshot", cut)
		}
	}
	term := New(8, 3)
	base := term.Palette()
	for _, body := range []string{
		"-1=red;256=red;1_0=red;foreground =red;1=bad-color;selection_foreground=red;cursor_text=?;unknown=?",
		"1=red;" + strings.Repeat("0=;", maxKittyColorRequests),
	} {
		term.Write([]byte("\x1b]21;" + body + "\x1b\\"))
		if term.Palette() != base {
			t.Fatalf("invalid or oversized batch changed colors: %q", body[:min(64, len(body))])
		}
	}
}

func TestNamedAndIntensityColors(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want RGB
		ok   bool
	}{
		{"\tFoReSt GrEeN ", RGB{34, 139, 34}, true}, {"LawnGreen", RGB{124, 252, 0}, true},
		{"rgbi:1/.5/-0", RGB{255, 127, 0}, true}, {"rgbi:2/-1/1e-1", RGB{255, 0, 25}, true},
		{"rgbi:1e999/-1e999/1e-999", RGB{255, 0, 0}, true},
		{"rgbi:NaN/0/0", RGB{}, false}, {"rgbi:Inf/0/0", RGB{}, false}, {"rgbi:0x1p-1/0/0", RGB{}, false},
		{"rgbi:0/0/0/0", RGB{}, false}, {"rgbi:/0/0", RGB{}, false}, {"rgbi:./0/0", RGB{}, false},
	} {
		if got, ok := parseColor(tc.in); got != tc.want || ok != tc.ok {
			t.Fatalf("parseColor(%q) = %v,%v, want %v,%v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
	for i, e := range x11ColorNames {
		if i > 0 && x11ColorNames[i-1].name >= e.name {
			t.Fatal("generated names are not sorted and unique")
		}
		if got, ok := parseColor(e.name); !ok || got != e.color {
			t.Fatalf("generated color %q failed lookup", e.name)
		}
	}
	if n := testing.AllocsPerRun(100, func() { _, _ = parseColor("FoReStGrEeN") }); n != 0 {
		t.Fatalf("named color lookup allocated %g times", n)
	}
}
