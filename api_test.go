package phosphor

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

func TestCell(t *testing.T) {
	term := New(10, 3)
	term.Write([]byte("a\x1b[1;3;4:3;38;5;196;48;2;1;2;3m漢\x1b[0;9;53;21mé\x1b[m\x1b]8;id=x:k=v;https://e.com\x1b\\L\x1b]8;;\x1b\\"))
	cases := []struct {
		x    int
		want Cell
	}{
		{0, Cell{Rune: 'a', Width: 1}},
		{1, Cell{Rune: '漢', Width: 2, Style: Style{Fg: IndexedColor(196), Bg: RGBColor(1, 2, 3), Attrs: AttrBold | AttrItalic, Underline: UnderlineCurly}}},
		{2, Cell{Width: 0, Style: Style{Fg: IndexedColor(196), Bg: RGBColor(1, 2, 3), Attrs: AttrBold | AttrItalic, Underline: UnderlineCurly}}},
		{3, Cell{Rune: 'e', Combining: "́", Width: 1, Style: Style{Attrs: AttrStrikethrough | AttrOverline, Underline: UnderlineDouble}}},
		{4, Cell{Rune: 'L', Width: 1, Link: Hyperlink{ID: "x", URI: "https://e.com"}}},
		{5, Cell{Width: 1}},
	}
	for _, c := range cases {
		if got := term.Cell(c.x, 0); got != c.want {
			t.Errorf("cell %d: got %+v\nwant %+v", c.x, got, c.want)
		}
	}
	if got := term.Cell(0, 2); got != (Cell{Width: 1}) {
		t.Errorf("blank row: %+v", got)
	}
	for _, p := range [][2]int{{-1, 0}, {10, 0}, {0, 3}, {0, -1}} {
		if got := term.Cell(p[0], p[1]); got != (Cell{}) {
			t.Errorf("out of range %v: %+v", p, got)
		}
	}
	if got := term.Cell(3, 0).Text(); got != "é" {
		t.Errorf("text %q", got)
	}
	if got := string(term.Cell(1, 0).AppendText([]byte(">"))); got != ">漢" {
		t.Errorf("append text %q", got)
	}
	if term.Cell(2, 0).Text() != "" || term.Cell(5, 0).Text() != "" {
		t.Errorf("blank text")
	}
	// A spacer head (left when a wide character wrapped) reads as blank.
	sp := New(3, 2)
	sp.Write([]byte("ab漢"))
	if got := sp.Cell(2, 0); got != (Cell{Width: 1}) || sp.Cell(0, 1).Rune != '漢' {
		t.Errorf("spacer head %+v", got)
	}
}

func TestCellDoesNotAllocate(t *testing.T) {
	term := New(20, 4)
	term.Write([]byte("ascii \x1b[31m漢字 é̂ \x1b]8;id=a;http://x\x07link\x1b]8;;\x07"))
	var b []byte
	allocs := testing.AllocsPerRun(100, func() {
		for y := range 4 {
			for x := range 20 {
				c := term.Cell(x, y)
				b = c.AppendText(b[:0])
				_ = c.Text() // ASCII and blanks only allocate nothing; see below
			}
		}
	})
	// Text allocates for the non-ASCII cells (漢, 字, é): 3 per run.
	if allocs > 3 {
		t.Fatalf("%v allocations reading the screen", allocs)
	}
	allocs = testing.AllocsPerRun(100, func() {
		for y := range 4 {
			for x := range 20 {
				b = term.Cell(x, y).AppendText(b[:0])
			}
		}
	})
	if allocs != 0 {
		t.Fatalf("%v allocations reading cells with AppendText", allocs)
	}
}

func TestColor(t *testing.T) {
	for _, c := range []struct {
		c     Color
		kind  ColorKind
		index uint8
		rgb   RGB
	}{
		{DefaultColor, ColorDefault, 0, RGB{}},
		{IndexedColor(0), ColorIndexed, 0, RGB{}},
		{IndexedColor(255), ColorIndexed, 255, RGB{}},
		{RGBColor(1, 2, 3), ColorRGB, 0, RGB{1, 2, 3}},
		{RGBColor(0, 0, 0), ColorRGB, 0, RGB{}},
	} {
		if c.c.Kind() != c.kind || c.c.Index() != c.index || c.c.RGB() != c.rgb {
			t.Errorf("%#x: kind %v index %v rgb %v", uint32(c.c), c.c.Kind(), c.c.Index(), c.c.RGB())
		}
	}
	p := DefaultPalette()
	if p.Colors[1] != (RGB{205, 0, 0}) || p.Colors[16] != (RGB{}) || p.Colors[231] != (RGB{255, 255, 255}) || p.Colors[232] != (RGB{8, 8, 8}) || p.Colors[255] != (RGB{238, 238, 238}) {
		t.Errorf("default palette %v %v %v %v", p.Colors[1], p.Colors[16], p.Colors[231], p.Colors[255])
	}
	if p.Resolve(DefaultColor, true) != p.Background || p.Resolve(DefaultColor, false) != p.Foreground ||
		p.Resolve(IndexedColor(9), false) != p.Colors[9] || p.Resolve(RGBColor(4, 5, 6), true) != (RGB{4, 5, 6}) {
		t.Errorf("resolve")
	}
}

func TestCursorAndModes(t *testing.T) {
	term := New(5, 3)
	if got := term.Cursor(); got != (Cursor{Visible: true}) {
		t.Fatalf("initial cursor %+v", got)
	}
	term.Write([]byte("abcde\x1b[?25l\x1b[3 q"))
	if got := term.Cursor(); got != (Cursor{X: 4, Shape: CursorUnderline, Blink: true, PendingWrap: true}) {
		t.Fatalf("cursor %+v", got)
	}
	for _, c := range []struct {
		n     string
		shape CursorShape
		blink bool
	}{{"0", CursorDefault, false}, {"1", CursorBlock, true}, {"2", CursorBlock, false}, {"4", CursorUnderline, false}, {"5", CursorBar, true}, {"6", CursorBar, false}, {"7", CursorBar, false}} {
		term.Write([]byte("\x1b[" + c.n + " q"))
		if got := term.Cursor(); got.Shape != c.shape || got.Blink != c.blink {
			t.Errorf("DECSCUSR %s: %+v", c.n, got)
		}
	}
	if got := term.Modes(); got != (Modes{Autowrap: true}) {
		t.Fatalf("initial modes %+v", got)
	}
	term.Write([]byte("\x1b[?1049h\x1b[?1h\x1b=\x1b[?2004h\x1b[?1004h\x1b[?1002h\x1b[?1006h\x1b[>5u\x1b[?7l\x1b[4h\x1b[20h\x1b[?6h\x1b[?5h\x1b[?2026h"))
	want := Modes{AltScreen: true, ApplicationCursorKeys: true, ApplicationKeypad: true, BracketedPaste: true, FocusEvents: true,
		Mouse: MouseButtonEvent, MouseEncoding: MouseEncodingSGR, KittyKeyboard: 5, Insert: true, Newline: true, Origin: true,
		ReverseVideo: true, SyncUpdate: true}
	if got := term.Modes(); got != want {
		t.Fatalf("modes\n got %+v\nwant %+v", got, want)
	}
	term.Write([]byte("\x1bc"))
	if got := term.Modes(); got != (Modes{Autowrap: true}) {
		t.Fatalf("modes after RIS %+v", got)
	}
}

func TestTextHelpers(t *testing.T) {
	term := New(4, 2, WithScrollback(10))
	term.Write([]byte("abcdefg\r\nhi\r\n\x1b[31mred\x1b[m\r\nend"))
	if got := term.ScrollbackLines(); !slices.Equal(got, []string{"abcdefg", "hi"}) {
		t.Fatalf("scrollback %q", got)
	}
	if term.ScrollbackLine(0) != "abcdefg" || term.ScrollbackLine(1) != "hi" || term.ScrollbackLine(2) != "" || term.ScrollbackLine(-1) != "" {
		t.Fatalf("scrollback line")
	}
	if term.Line(0) != "red" || term.Line(1) != "end" || term.Line(2) != "" {
		t.Fatalf("lines %q", term.Lines())
	}
	rows := term.ScrollbackCells(0)
	if len(rows) != 2 || rows[0][3].Rune != 'd' || rows[1][2].Rune != 'g' || rows[1][3] != (Cell{Width: 1}) {
		t.Fatalf("scrollback cells %+v", rows)
	}
	if term.ScrollbackCells(5) != nil {
		t.Fatalf("scrollback cells out of range")
	}
	w := New(4, 3)
	w.Write([]byte("abcdefgh\r\nx"))
	if w.Wrapped(0) || !w.Wrapped(1) || w.Wrapped(2) || w.Wrapped(3) {
		t.Fatalf("wrapped %v %v %v", w.Wrapped(0), w.Wrapped(1), w.Wrapped(2))
	}
	red := New(4, 1)
	red.Write([]byte("\x1b[31mab"))
	if got := red.Cell(1, 0).Style.Fg; got != IndexedColor(1) {
		t.Fatalf("fg %#x", uint32(got))
	}
}

func TestScrollbackOptions(t *testing.T) {
	none := New(10, 2, WithScrollback(0))
	none.Write([]byte("1\r\n2\r\n3\r\n4"))
	if none.ScrollbackLen() != 0 {
		t.Fatalf("scrollback kept with WithScrollback(0)")
	}
	unlimited := New(100, 2, WithScrollbackBytes(0))
	for range 5000 {
		unlimited.Write([]byte(strings.Repeat("x", 90) + "\r\n"))
	}
	if unlimited.ScrollbackLen() != DefaultScrollbackLines {
		t.Fatalf("lines %d", unlimited.ScrollbackLen())
	}
	if New(10, 2).sbMaxBytes != DefaultScrollbackBytes || New(10, 2, WithBell(nil)).sbMaxLines != DefaultScrollbackLines {
		t.Fatalf("defaults")
	}
}

func TestPalette(t *testing.T) {
	var out bytes.Buffer
	base := DefaultPalette()
	base.Background = RGB{0xff, 0xff, 0xff}
	term := New(10, 2, WithPalette(base), WithReply(&out))
	if term.Palette() != base {
		t.Fatalf("base palette")
	}
	query := func(q, want string) {
		t.Helper()
		out.Reset()
		term.Write([]byte(q))
		if out.String() != want {
			t.Fatalf("%q: got %q want %q", q, out.String(), want)
		}
	}
	query("\x1b]11;?\x07", "\x1b]11;rgb:ffff/ffff/ffff\x07")
	query("\x1b]10;?\x1b\\", "\x1b]10;rgb:e5e5/e5e5/e5e5\x1b\\")
	query("\x1b]4;1;?\x07", "\x1b]4;1;rgb:cdcd/0000/0000\x07")
	query("\x1b]10;?;?\x07", "\x1b]10;rgb:e5e5/e5e5/e5e5\x07\x1b]11;rgb:ffff/ffff/ffff\x07")
	query("\x1b[?996n", "\x1b[?997;2n")
	query("\x1b]11;#102030\x07\x1b]11;?\x07", "\x1b]11;rgb:1010/2020/3030\x07")
	query("\x1b[?996n", "\x1b[?997;1n")
	query("\x1b]4;1;rgb:f/80/0\x07\x1b]4;1;?;2;?\x07", "\x1b]4;1;rgb:ffff/8080/0000\x07\x1b]4;2;rgb:0000/cdcd/0000\x07")
	query("\x1b]12;rgb:1234/5678/9abc\x07\x1b]12;?\x07", "\x1b]12;rgb:1212/5656/9a9a\x07")
	p := term.Palette()
	if p.Background != (RGB{0x10, 0x20, 0x30}) || p.Colors[1] != (RGB{0xff, 0x80, 0}) || p.Cursor != (RGB{0x12, 0x56, 0x9a}) {
		t.Fatalf("palette %v %v %v", p.Background, p.Colors[1], p.Cursor)
	}
	query("\x1b]104;1\x07\x1b]111\x07\x1b]4;1;?\x07\x1b]11;?\x07", "\x1b]4;1;rgb:cdcd/0000/0000\x07\x1b]11;rgb:ffff/ffff/ffff\x07")
	term.Write([]byte("\x1b]4;3;#abc\x07\x1bc"))
	if term.Palette() != base {
		t.Fatalf("RIS keeps palette changes")
	}
	// Without a reply writer colors are still tracked.
	plain := New(10, 2)
	plain.Write([]byte("\x1b]4;7;#ffffff\x07\x1b]4;8;nonsense\x07"))
	if got := plain.Palette(); got.Colors[7] != (RGB{255, 255, 255}) || got.Colors[8] != DefaultPalette().Colors[8] {
		t.Fatalf("plain palette %v %v", got.Colors[7], got.Colors[8])
	}
	for _, c := range []struct {
		in   string
		want RGB
		ok   bool
	}{
		{"rgb:ff/00/80", RGB{255, 0, 128}, true}, {"rgb:f/0/8", RGB{255, 0, 136}, true}, {"rgb:ffff/0000/8000", RGB{255, 0, 128}, true},
		{"#f08", RGB{0xf0, 0, 0x80}, true}, {"#ff0080", RGB{255, 0, 128}, true}, {"#fff000888", RGB{255, 0, 0x88}, true},
		{"rgb:1/2", RGB{}, false}, {"rgb:12345/0/0", RGB{}, false}, {"#12", RGB{}, false}, {"red", RGB{}, false}, {"#ggg", RGB{}, false},
	} {
		if got, ok := parseColor(c.in); got != c.want || ok != c.ok {
			t.Errorf("parseColor(%q) = %v, %v", c.in, got, ok)
		}
	}
}

func TestReplies(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"\x1b[c", "\x1b[?62;22c"},
		{"\x1b[0c", "\x1b[?62;22c"},
		{"\x1b[1c", ""},
		{"\x1b[>c", "\x1b[>1;10;0c"},
		{"\x1b[=c", "\x1bP!|00000000\x1b\\"},
		{"\x1b[5n", "\x1b[0n"},
		{"\x1b[6n", "\x1b[1;1R"},
		{"\x1b[3;7H\x1b[6n", "\x1b[3;7R"},
		{"\x1b[2;4r\x1b[?6h\x1b[2;2H\x1b[6n", "\x1b[2;2R"}, // relative to the region in origin mode
		{"\x1b[3;7H\x1b[?6n", "\x1b[?3;7R"},
		{"abcdefghij\x1b[6n", "\x1b[1;10R"}, // a pending wrap reports the last column
		{"\x1b[?15n", "\x1b[?13n"},
		{"\x1b[?26n", "\x1b[?27;1;0;0n"},
		{"\x1b[?996n", "\x1b[?997;1n"},
		{"\x1b[7n\x1b[?99n", ""},
		{"\x1b[>q", "\x1bP>|phosphor\x1b\\"},
		{"\x1b[>0q", "\x1bP>|phosphor\x1b\\"},
		{"\x1b[?u", "\x1b[?0u"},
		{"\x1b[>13u\x1b[?u", "\x1b[?13u"},
		{"\x1b[18t", "\x1b[8;5;10t"},
		{"\x1b[14t", ""},
		// DECRQM
		{"\x1b[?1$p", "\x1b[?1;2$y"},
		{"\x1b[?1h\x1b[?1$p", "\x1b[?1;1$y"},
		{"\x1b[?7$p", "\x1b[?7;1$y"},
		{"\x1b[?25$p", "\x1b[?25;1$y"},
		{"\x1b[?1049h\x1b[?1049$p\x1b[?47$p", "\x1b[?1049;1$y\x1b[?47;1$y"},
		{"\x1b[?2004h\x1b[?2004$p", "\x1b[?2004;1$y"},
		{"\x1b[?2026h\x1b[?2026$p", "\x1b[?2026;1$y"},
		{"\x1b[?1006$p", "\x1b[?1006;2$y"},
		{"\x1b=\x1b[?66$p", "\x1b[?66;1$y"},
		{"\x1b[?2027$p", "\x1b[?2027;2$y"},
		{"\x1b[?9999$p", "\x1b[?9999;0$y"},
		{"\x1b[4h\x1b[4$p\x1b[20$p\x1b[2$p", "\x1b[4;1$y\x1b[20;2$y\x1b[2;0$y"},
		// DECRQSS
		{"\x1b[1;4:3;38;5;100;48;2;1;2;3m\x1bP$qm\x1b\\", "\x1bP1$r0;1;4:3;38;5;100;48;2;1;2;3m\x1b\\"},
		{"\x1bP$qm\x1b\\", "\x1bP1$r0m\x1b\\"},
		{"\x1b[2;4r\x1bP$qr\x1b\\", "\x1bP1$r2;4r\x1b\\"},
		{"\x1bP$qr\x1b\\", "\x1bP1$r1;5r\x1b\\"},
		{"\x1b[?69h\x1b[2;8s\x1bP$qs\x1b\\", "\x1bP1$r2;8s\x1b\\"},
		{"\x1b[5 q\x1bP$q q\x1b\\", "\x1bP1$r5 q\x1b\\"},
		{"\x1bP$q q\x1b\\", "\x1bP1$r2 q\x1b\\"},
		{"A\x1b[2;3r\x1b[?69h\x1b[3;6s\x1b[?6hB\x1b[22JX\x1b[6n\x1b[?6n", "\x1b[1;1R\x1b[?1;1R"},
		{"\x1b[1\"q\x1bP$q\"q\x1b\\", "\x1bP1$r1\"q\x1b\\"},
		{"\x1bP$q\"p\x1b\\", "\x1bP1$r62;1\"p\x1b\\"},
		{"\x1bP$qx\x1b\\", "\x1bP0$r\x1b\\"},
		// XTGETTCAP
		{"\x1bP+q436f;636f6c6f7273;544e\x1b\\", "\x1bP1+r436f=323536\x1b\\\x1bP1+r636f6c6f7273=323536\x1b\\\x1bP0+r544e\x1b\\"},
		{"\x1bP+qzz\x1b\\", "\x1bP0+rzz\x1b\\"},
		// Not queries, or not answered.
		{"\x1bPq#0;2;0;0;0#0~~@@vv@@~~$\x1b\\", ""},
		{"\x1bP1$r0m\x1b\\", ""},
		{"\x1b[?6;7;8$p", "\x1b[?6;2$y"},
	} {
		var out bytes.Buffer
		term := New(10, 5, WithReply(&out))
		term.Write([]byte(c.in))
		if out.String() != c.want {
			t.Errorf("%q: got %q want %q", c.in, out.String(), c.want)
		}
		// Queries leave the screen alone.
		if strings.Contains(c.in, "\x1bP") && strings.Join(term.Lines(), "") != "" {
			t.Errorf("%q printed %q", c.in, term.Lines())
		}
		// And are answered whole when cut anywhere.
		for cut := 1; cut < len(c.in); cut++ {
			var split bytes.Buffer
			term := New(10, 5, WithReply(&split))
			term.Write([]byte(c.in[:cut]))
			term.Write([]byte(c.in[cut:]))
			if split.String() != c.want {
				t.Fatalf("%q cut at %d: got %q", c.in, cut, split.String())
			}
		}
	}
	// Without a reply writer nothing is answered, and nothing breaks.
	term := New(10, 5)
	term.Write([]byte("\x1b[c\x1b[6n\x1bP$qm\x1b\\\x1b]11;?\x07ok"))
	if term.Line(0) != "ok" {
		t.Fatalf("screen %q", term.Lines())
	}
}

func TestBell(t *testing.T) {
	n := 0
	term := New(10, 2, WithBell(func() { n++ }))
	term.Write([]byte("a\ab\x1b]2;title\x07\a"))
	if n != 2 || term.Title() != "title" {
		t.Fatalf("bells %d, title %q", n, term.Title())
	}
	term.Write([]byte("\x1bc\a"))
	if n != 3 {
		t.Fatalf("bell lost on RIS")
	}
}

// rowSig is everything a renderer draws for row y.
func rowSig(t *Terminal, y int) string {
	var b strings.Builder
	cols, _ := t.Size()
	for x := range cols {
		fmt.Fprintf(&b, "%+v|", t.Cell(x, y))
	}
	return b.String()
}

func TestDamage(t *testing.T) {
	term := New(5, 4)
	if !term.Dirty() || !term.RowDirty(0) || !term.RowDirty(3) {
		t.Fatalf("a terminal never cleared is all dirty")
	}
	dirty := func() []int {
		var out []int
		for y := range 4 {
			if term.RowDirty(y) {
				out = append(out, y)
			}
		}
		return out
	}
	term.ClearDirty()
	if term.Dirty() || dirty() != nil || term.RowDirty(-1) || term.RowDirty(4) {
		t.Fatalf("dirty after clear: %v", dirty())
	}
	for _, c := range []struct {
		in   string
		want []int
	}{
		{"\x1b[2;1Hab", []int{1}},
		{"\x1b[H\x1b[3B", nil}, // cursor moves only
		{"\x1b]2;title\x07\x1b[?2004h", nil},
		{"\x1b[4;1H\n", []int{0, 1, 2, 3}}, // scrolls the screen
		{"\x1b[2;3r\x1b[3;1H\n", []int{1, 2}},
		{"\x1b[r\x1b[1;1H\x1b[K", []int{0}},
		{"\x1b[3;1H\x1b[L", []int{2, 3}},
		{"\x1b[?1049h", []int{0, 1, 2, 3}},
		{"\x1b[?1049l", []int{0, 1, 2, 3}},
		{"\x1b[?5h", []int{0, 1, 2, 3}},
		{"\x1b]4;1;#ff0000\x07", []int{0, 1, 2, 3}},
		{"\x1b[4;1Hx́", []int{3}},
		{"\x1bc", []int{0, 1, 2, 3}},
	} {
		term.Write([]byte(c.in))
		if got := dirty(); !slices.Equal(got, c.want) {
			t.Errorf("%q: dirty %v want %v", c.in, got, c.want)
		}
		term.ClearDirty()
	}
	term.Resize(7, 70)
	if !term.RowDirty(69) || !term.RowDirty(0) {
		t.Fatalf("resize leaves rows clean")
	}
	term.ClearDirty()
	term.Write([]byte("\x1b[66;1Hz"))
	if !term.RowDirty(65) || term.RowDirty(64) || term.RowDirty(0) {
		t.Fatalf("row 65 after resize")
	}
	term.Restart()
	if !term.RowDirty(0) || !term.RowDirty(69) {
		t.Fatalf("restart leaves rows clean")
	}
}

// Every row whose drawn content changed is dirty, on random output.
func TestDamageCoversChanges(t *testing.T) {
	rng := rand.New(rand.NewPCG(11, 12))
	for i := range 300 {
		cols, rows := 1+rng.IntN(12), 1+rng.IntN(8)
		term := New(cols, rows, WithScrollback(5))
		term.ClearDirty()
		for range 60 {
			before := make([]string, rows)
			for y := range rows {
				before[y] = rowSig(term, y)
			}
			tok := tokens[rng.IntN(len(tokens))]
			term.Write([]byte(tok))
			for y := range rows {
				if rowSig(term, y) != before[y] && !term.RowDirty(y) {
					t.Fatalf("case %d (%dx%d): row %d changed by %q but is clean\nbefore %s\nafter  %s", i, cols, rows, y, tok, before[y], rowSig(term, y))
				}
			}
			term.ClearDirty()
		}
	}
}

// Damage tracking adds no allocations to output.
func TestDamageDoesNotAllocate(t *testing.T) {
	term := New(80, 24)
	term.ClearDirty()
	frame := []byte("\x1b[H\x1b[2J\x1b[1;31mhello\x1b[m world\r\n\x1b[5;10Hmore text\x1b[K\x1b[10;1H\x1b[L")
	term.Write(frame)
	if n := testing.AllocsPerRun(100, func() {
		term.Write(frame)
		for y := range 24 {
			_ = term.RowDirty(y)
		}
		term.ClearDirty()
	}); n != 0 {
		t.Fatalf("%v allocations per frame", n)
	}
}
