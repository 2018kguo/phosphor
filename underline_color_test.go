package phosphor

import (
	"fmt"
	"testing"
	"unsafe"
)

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
