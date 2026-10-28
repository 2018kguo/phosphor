package phosphor

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unsafe"
)

func feed(cols, rows int, in string) *Terminal {
	t := New(cols, rows)
	t.Write([]byte(in))
	return t
}

func TestBasics(t *testing.T) {
	cases := []struct {
		name, in string
		want     []string
		x, y     int
	}{
		{"text and newline", "ab\r\ncd", []string{"ab", "cd", ""}, 2, 1},
		{"pending wrap", "abcde", []string{"abcde", "", ""}, 4, 0},
		{"wrap on next char", "abcdef", []string{"abcde", "f", ""}, 1, 1},
		{"CR clears pending wrap", "abcde\rX", []string{"Xbcde", "", ""}, 1, 0},
		{"scroll", "1\r\n2\r\n3\r\n4", []string{"2", "3", "4"}, 1, 2},
		{"cup and erase", "hello\x1b[1;3H\x1b[K", []string{"he", "", ""}, 2, 0},
		{"wide char", "a漢b", []string{"a漢b", "", ""}, 4, 0},
		{"wide char wraps early", "abcd漢", []string{"abcd", "漢", ""}, 2, 1},
		{"overwrite wide half", "漢\x1b[1;2Hx", []string{" x", "", ""}, 2, 0},
		{"insert line", "1\r\n2\r\n3\x1b[2;1H\x1b[L", []string{"1", "", "2"}, 0, 1},
		{"delete chars", "abcde\x1b[1;2H\x1b[2P", []string{"ade", "", ""}, 1, 0},
		{"insert chars", "abc\x1b[1;2H\x1b[2@", []string{"a  bc", "", ""}, 1, 0},
		{"erase chars", "abcde\x1b[1;2H\x1b[3X", []string{"a   e", "", ""}, 1, 0},
		{"dec graphics", "\x1b(0lqk\x1b(Bx", []string{"┌─┐x", "", ""}, 4, 0},
		{"split utf8", "\xe6\xbc\xa2", []string{"漢", "", ""}, 2, 0},
		{"invalid utf8", "a\xffb", []string{"a�b", "", ""}, 3, 0},
		{"rep", "x\x1b[3b", []string{"xxxx", "", ""}, 4, 0},
		{"reverse index scrolls", "1\r\n2\x1b[H\x1bM", []string{"", "1", "2"}, 0, 0},
		{"scroll region", "\x1b[1;2r\x1b[2;1Ha\nb\nc", []string{" b", "  c", ""}, 3, 1},
		{"alt screen keeps cursor", "main\x1b[?1049hALT", []string{"    A", "LT", ""}, 2, 1},
		{"alt screen", "main\x1b[?1049h\x1b[HALT", []string{"ALT", "", ""}, 3, 0},
		{"alt screen exit", "main\x1b[?1049hALT\x1b[?1049l", []string{"main", "", ""}, 4, 0},
		{"osc ignored", "a\x1b]0;title\x07b\x1b]8;;http://x\x1b\\c", []string{"abc", "", ""}, 3, 0},
		{"dcs ignored", "a\x1bP1$r0m\x1b\\b", []string{"ab", "", ""}, 2, 0},
		{"cut sequence then text", "a\x1b[3", []string{"a", "", ""}, 1, 0},
	}
	for _, c := range cases {
		term := feed(5, 3, c.in)
		if got := term.Lines(); !slices.Equal(got, c.want) {
			t.Errorf("%s: lines %q, want %q", c.name, got, c.want)
		}
		if x, y := term.Cursor().X, term.Cursor().Y; x != c.x || y != c.y {
			t.Errorf("%s: cursor %d,%d want %d,%d", c.name, x, y, c.x, c.y)
		}
	}
}

func TestScrollbackAndModes(t *testing.T) {
	term := feed(10, 2, "\x1b]2;my title\x07\x1b[?2004h\x1b[?1h\x1b[>1u\x1b[?25l1\r\n2\r\n3\r\n4")
	if got := term.ScrollbackLines(); !slices.Equal(got, []string{"1", "2"}) {
		t.Fatalf("scrollback %q", got)
	}
	if term.Title() != "my title" || !term.bracketed || !term.appCursor || len(term.kitty) != 1 || !term.cursorHidden {
		t.Fatalf("modes not tracked: %+v", term)
	}
	// Scrollback respects its line budget.
	small := New(10, 2, WithScrollback(3))
	for i := range 100 {
		fmt.Fprintf(small, "%d\r\n", i)
	}
	if got := small.ScrollbackLines(); !slices.Equal(got, []string{"96", "97", "98"}) {
		t.Fatalf("bounded scrollback %q", got)
	}
	// And its byte budget.
	byBytes := New(200, 2, WithScrollbackBytes(4096))
	for range 1000 {
		fmt.Fprintf(byBytes, "%s\r\n", strings.Repeat("x", 150))
	}
	if byBytes.sbBytes > 4096 || byBytes.ScrollbackLen() == 0 {
		t.Fatalf("scrollback bytes %d lines %d", byBytes.sbBytes, byBytes.ScrollbackLen())
	}
	// The alternate screen does not feed scrollback.
	alt := feed(10, 2, "\x1b[?1049h1\r\n2\r\n3\r\n4")
	if alt.ScrollbackLen() != 0 {
		t.Fatalf("alt screen scrolled into scrollback")
	}
}

func TestResize(t *testing.T) {
	term := feed(5, 4, "1\r\n2\r\n3")
	term.Resize(3, 2) // blank row 4 goes first, then row 1 to scrollback
	if got := term.Lines(); !slices.Equal(got, []string{"2", "3"}) {
		t.Fatalf("lines after shrink %q", got)
	}
	if got := term.ScrollbackLines(); !slices.Equal(got, []string{"1"}) {
		t.Fatalf("scrollback after shrink %q", got)
	}
	if x, y := term.Cursor().X, term.Cursor().Y; x != 1 || y != 1 {
		t.Fatalf("cursor %d,%d", x, y)
	}
	// Growing brings the line back from the scrollback.
	term.Resize(6, 3)
	term.Write([]byte("\x1b[3;1Habcdef"))
	if got := term.Lines(); !slices.Equal(got, []string{"1", "2", "abcdef"}) || term.ScrollbackLen() != 0 {
		t.Fatalf("lines after grow %q, scrollback %q", got, term.ScrollbackLines())
	}
}

func TestRestart(t *testing.T) {
	// The old content stays in view; the new program starts below it.
	term := feed(10, 4, "old1\r\nold2")
	term.Restart()
	term.Write([]byte("$ "))
	if got := term.Lines(); !slices.Equal(got, []string{"old1", "old2", "$", ""}) || term.ScrollbackLen() != 0 {
		t.Fatalf("lines %q, scrollback %q", got, term.ScrollbackLines())
	}

	// What does not fit above the first row scrolls into scrollback.
	term = feed(10, 3, "old1\r\nold2\x1b[?1049hALT")
	term.Restart()
	if got := term.ScrollbackLines(); !slices.Equal(got, []string{"old1", "old2"}) {
		t.Fatalf("scrollback %q", got)
	}
	if got := term.Lines(); term.Modes().AltScreen || !slices.Equal(got, []string{"", "    ALT", ""}) {
		t.Fatalf("screen %q, alt %v", got, term.Modes().AltScreen)
	}
}

// Idle and lightly used terminals cost little: rows exist only once drawn
// on; a previously used alternate screen keeps its contents.
func TestLazyAllocation(t *testing.T) {
	term := New(200, 50)
	term.Write([]byte("$ "))
	allocated := 0
	for _, row := range term.main {
		if row != nil {
			allocated++
		}
	}
	if allocated != 1 || term.alt != nil {
		t.Fatalf("%d rows allocated, alt %v", allocated, term.alt != nil)
	}
	term.Write([]byte("\x1b[?1049hx\x1b[?1049l"))
	if term.alt == nil {
		t.Fatal("1049 exit discarded alternate contents")
	}
}

// --- invariants over random input ---

// tokens is the alphabet of the random streams: text, controls, and the
// sequences full-screen programs use.
var tokens = []string{
	"a", "hello", "x y", "漢字", "🙂", "é", "\r", "\n", "\r\n", "\b", "\t",
	"\x1b[H", "\x1b[2;3H", "\x1b[10;80H", "\x1b[A", "\x1b[3B", "\x1b[2C", "\x1b[D", "\x1b[5G", "\x1b[2d", "\x1b[E", "\x1b[F",
	"\x1b[K", "\x1b[1K", "\x1b[2K", "\x1b[J", "\x1b[1J", "\x1b[2J", "\x1b[3J",
	"\x1b[L", "\x1b[2M", "\x1b[@", "\x1b[3P", "\x1b[2X", "\x1b[S", "\x1b[2T", "\x1b[2b",
	"\x1b[m", "\x1b[1;31m", "\x1b[4:3m", "\x1b[38;5;200m", "\x1b[48;2;1;2;3m", "\x1b[38:2::10:20:30m", "\x1b[7;9;53m", "\x1b[22;24;27m", "\x1b[94;103m", "\x1b[39;49m", "\x1b[58;5;3m", "\x1b[58:2::10:20:30m", "\x1b[59m",
	"\x1b[2;4r", "\x1b[r", "\x1b[?6h", "\x1b[?6l", "\x1b[?7l", "\x1b[?7h", "\x1b[4h", "\x1b[4l",
	"\x1b[?1049h", "\x1b[?1049l", "\x1b[?47h", "\x1b[?47l", "\x1b[?1047h", "\x1b[?1047l",
	"\x1b7", "\x1b8", "\x1b[s", "\x1b[u", "\x1bM", "\x1bD", "\x1bE",
	"\x1b(0", "\x1b(B", "\x1b)0", "\x0e", "\x0f", "lqk",
	"\x1b]2;t\x07", "\x1b[?1h", "\x1b[?2004h", "\x1b[?1000h", "\x1b[?1006h", "\x1b[?1004h", "\x1b[?25l", "\x1b[?25h",
	"\x1b[>3u", "\x1b[<u", "\x1b[?45h", "\x1b]8;;http://a\x1b\\", "\x1b]8;id=1;http://b\a", "\x1b]8;;\x1b\\", "\x1b[1\"q", "\x1b[0\"q", "\x1bV", "\x1bW", "\x1b[?J", "\x1b[?1K", "\x1b[?2K", "\x1b[?69h", "\x1b[?69l", "\x1b[2;4s", "\x1b[3;5s", "\x1b[?5h", "\x1b[?5l", "\x1b[?1045h", "\x1b[?45l", "\x1b[?1045l", "\x1b[3D", "\x1bH", "\x1b[g", "\x1b[3g", "\x1b[2Z", "\x1b[2I", "\u0301", "e\u0301\u0302", "\u200d", "\x1b*0", "\x1bn", "\x1bN", "\x1b[?1s", "\x1b[?7s", "\x1b[?7r", "\x1b[?1r", "\x1b[=5;1u", "\x1b[2 q", "\x1b=", "\x1b>", "\x1b[!p",
}

func randomStream(rng *rand.Rand, n int) string {
	var b strings.Builder
	for range n {
		b.WriteString(tokens[rng.IntN(len(tokens))])
	}
	return b.String()
}

// state is a comparable summary of everything Repaint promises to carry.
func state(t *Terminal) string { return stateOf(t, true) }

func stateOf(t *Terminal, withScrollback bool) string {
	var b strings.Builder
	dump := func(name string, scr [][]cell, wrap []rowFlags) {
		for y := range t.rows {
			var row []cell
			if y < len(scr) {
				row = scr[y]
			}
			fmt.Fprintf(&b, "%s%d", name, y)
			for x := range t.cols {
				c := rowCell(row, x)
				s := "' '"
				switch {
				case c.R == wideTail:
					s = "tail"
					c = cell{} // a right half shows its left half's style
				case c.R == spacerHead:
				case c.R != 0 && c.R != ' ':
					s = strconv.Quote(t.cellString(c))
				}
				fmt.Fprintf(&b, " %s/%x/%x/%x", s, c.Fg, c.Bg, c.A)
				if c.E != 0 {
					fmt.Fprintf(&b, "/%+v", t.extraAt(c.E))
				}
			}
			if rowFlagsAt(wrap, y)&rowWrap != 0 {
				b.WriteString(" wrapped")
			}
			b.WriteByte('\n')
		}
	}
	dump("main", t.main, t.mainWrap)
	// An unallocated alternate screen has the same logical zero cells as
	// an allocated empty screen. Resize and Compact may release its storage.
	dump("alt", t.alt, t.altWrap)
	fmt.Fprintf(&b, "alternate active %v modes %d\n", t.altActive, t.Modes().AltScreenModes)
	cur := t.cur
	if cur.origin {
		// Repaint positions the cursor with origin mode on, which cannot
		// reach outside the region (DECRC and scroll-and-clear can).
		cur.wrapPending = false
		cur.x = min(max(cur.x, t.left), t.right)
		cur.y = min(max(cur.y, t.top), t.bot)
	}
	cur.pen.E = 0
	fmt.Fprintf(&b, "pen extra %+v\n", t.extraAt(t.cur.pen.E))
	fmt.Fprintf(&b, "cur %+v region %d-%d margins %v %d-%d ss %d iso %v\n", cur, t.top, t.bot, t.lrmm, t.left, t.right, t.cur.ss, t.protISO)
	slot := 0
	if t.altActive {
		slot = 1
	}
	var s savedCursor
	if t.saved != nil {
		s = t.saved.screen[slot]
	}
	extra := t.extraAt(s.pen.E)
	s.pen.E = 0
	fmt.Fprintf(&b, "saved %+v extra %+v\n", s, extra)
	fmt.Fprintf(&b, "modes %v %v %v %v %v %v %v %v %d %d %v %v %q\n", t.autowrap, t.insert, t.newline, t.cursorHidden,
		t.appCursor, t.appKeypad, t.bracketed, t.focus, t.mouse, t.mouseEnc, t.cursorStyle, t.kitty, t.title)
	var tabs []int
	for x := range t.cols {
		if t.isTab(x) {
			tabs = append(tabs, x)
		}
	}
	var saved []savedMode
	if t.x != nil {
		saved = t.x.savedModes
	}
	fmt.Fprintf(&b, "tabs %v xtsave %v revwrap %v %v reverse %v\n", tabs, saved, t.revWrap, t.revWrapExt, t.reverseVideo)
	if withScrollback {
		// A line cut off from its continuation (by scrolling down at the
		// top) keeps the form of a wrapped row: trailing blanks and no
		// erase; replayed, it comes back closed. Compare them alike.
		var lines [][]byte
		for i, l := range t.sb[t.sbStart:] {
			if !t.sbOpen || i < t.ScrollbackLen()-1 {
				l = normLine(l)
			}
			lines = append(lines, l)
		}
		sb := bytes.Join(lines, []byte("\n"))
		if t.sbOpen && t.sbSpacer && !startsWide(t.main[0]) {
			sb = append(sb, ' ') // what Repaint puts back
		}
		fmt.Fprintf(&b, "scrollback %q open %v\n", sb, t.sbOpen)
	}
	return b.String()
}

// Repaint must reproduce the state on a fresh terminal of the same size.
func TestRepaintRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 8))
	for i := range 3000 {
		cols, rows := 1+rng.IntN(12), 1+rng.IntN(8)
		in := randomStream(rng, rng.IntN(60))
		src := feed(cols, rows, in)
		dst := New(cols, rows)
		dst.Write(src.Repaint(true))
		if got, want := state(dst), state(src); got != want {
			t.Fatalf("case %d (%dx%d) input %q\nrepaint %q\n%s", i, cols, rows, in, src.Repaint(true), lineDiff(got, want))
		}
	}
}

// How output is split into writes must not matter.
func TestChunkingInvariant(t *testing.T) {
	rng := rand.New(rand.NewPCG(9, 10))
	for range 500 {
		in := []byte(randomStream(rng, 80))
		whole := New(9, 5)
		whole.Write(in)
		split := New(9, 5)
		for rest := in; len(rest) > 0; {
			n := min(len(rest), 1+rng.IntN(7))
			split.Write(rest[:n])
			rest = rest[n:]
		}
		if state(whole) != state(split) {
			t.Fatalf("chunking changed the result for %q", in)
		}
	}
}

// A stream cut anywhere, even inside an escape sequence, leaves a sane
// terminal that keeps working.
func TestCutAnywhere(t *testing.T) {
	in := []byte("\x1b[1;31mred\x1b]2;title\x07\x1b[?1049h\x1b[2;3H漢\x1b[38;2;1;2;3mx")
	for cut := range len(in) {
		term := New(10, 3)
		term.Write(in[:cut])
		term.Write([]byte("\x18\x1b[0m\x1b[?1049l\x1b[Hok"))
		if !strings.HasPrefix(term.Lines()[0], "ok") {
			t.Fatalf("cut at %d: %q", cut, term.Lines())
		}
	}
}

func FuzzWrite(f *testing.F) {
	f.Add([]byte("\x1b[2;4r\x1b[?1049h漢\x1b[3M\x1b[?6h\x1b[5;5H"), uint8(10), uint8(4))
	f.Fuzz(func(t *testing.T, in []byte, c, r uint8) {
		cols, rows := 1+int(c%40), 1+int(r%20)
		src := New(cols, rows)
		src.Write(in)
		_ = src.Lines()
		dst := New(cols, rows)
		dst.Write(src.Repaint(true))
		if !slices.Equal(dst.Lines(), src.Lines()) {
			t.Fatalf("repaint lines differ: %q vs %q", dst.Lines(), src.Lines())
		}
	})
}

// normLine renders a closed scrollback line on one wide row and lists its
// cells without the trailing blanks, whatever their background (a line
// cut from its continuation keeps blanks a replay need not recreate).
func normLine(line []byte) []byte {
	tmp := New(2*len(line)+2, 1, WithScrollback(0))
	tmp.Write(line)
	row := tmp.main[0]
	end := len(row)
	for end > 0 {
		c := row[end-1]
		if (c.R == 0 || c.R == ' ' || c.R == spacerHead) && c.A&^AttrProtected == 0 && c.Fg == 0 && c.E == 0 {
			end--
			continue
		}
		break
	}
	var b []byte
	for _, c := range row[:end] {
		b = fmt.Appendf(b, "%q/%x/%x/%x/%+v ", tmp.cellString(c), c.Fg, c.Bg, c.A, tmp.extraAt(c.E))
	}
	return b
}

func lineDiff(got, want string) string {
	g, w := strings.Split(got, "\n"), strings.Split(want, "\n")
	var b strings.Builder
	for i := range max(len(g), len(w)) {
		var gl, wl string
		if i < len(g) {
			gl = g[i]
		}
		if i < len(w) {
			wl = w[i]
		}
		if gl != wl {
			fmt.Fprintf(&b, "got:  %s\nwant: %s\n", gl, wl)
		}
	}
	return b.String()
}

// Redrawing lines in place (erase, cursor moves, insert and delete lines,
// as TUIs do every frame) reuses the screen's rows: no allocation once
// every row has been drawn on.
func TestRedrawDoesNotAllocate(t *testing.T) {
	term := New(80, 24)
	frame := []byte(strings.Repeat("\x1b[2K\x1b[1A", 10) + "\x1b[2K\x1b[G" +
		strings.Repeat("\x1b[1;38;2;200;100;50mredrawn line\x1b[0m\r\n", 10) +
		"\x1b[5;1H\x1b[2L\x1b[3M\x1b[J\x1b[2;24r\x1b[24;1H\x1b[S\x1b[T\n\x1b[r")
	term.Write([]byte("\x1b[24;1H" + strings.Repeat("\r\n", 30))) // scroll every row into use
	for y := range 24 {
		term.Write(fmt.Appendf(nil, "\x1b[%d;1Hx", y+1))
	}
	term.Write(frame)
	if n := testing.AllocsPerRun(100, func() { term.Write(frame) }); n != 0 {
		t.Fatalf("a redraw frame allocates %v times", n)
	}
}

// Shrinking the screen drops blank rows below the cursor first, also
// rows that were drawn on and then erased.
func TestResizeDropsErasedRowsFirst(t *testing.T) {
	term := feed(10, 6, "top\r\nmid\r\nlow\r\nx\x1b[3;1H\x1b[J\x1b[2;4H")
	term.Resize(10, 3)
	if got := term.Lines(); !slices.Equal(got, []string{"top", "mid", ""}) || term.ScrollbackLen() != 0 {
		t.Fatalf("lines %q, %d scrollback lines", got, term.ScrollbackLen())
	}
}

// Hyperlinks (OSC 8) stay on the cells they were printed on, survive
// Repaint and scrolling, and do not leak into the scrollback text.
func TestHyperlinks(t *testing.T) {
	term := feed(20, 2, "see \x1b]8;id=x;https://example.com/a\x1b\\docs\x1b]8;;\x1b\\ here\r\n")
	if c := term.cellAt(4, 0); term.link(c.E) != "id=x;https://example.com/a" || term.cellAt(8, 0).E != 0 {
		t.Fatalf("link %q on docs, %d after", term.link(c.E), term.cellAt(8, 0).E)
	}
	dst := New(20, 2)
	dst.Write(term.Repaint(true))
	if state(dst) != state(term) {
		t.Fatalf("repaint lost the link:\n%s", lineDiff(state(dst), state(term)))
	}
	term.Write([]byte("\r\n\r\n"))
	if got := term.ScrollbackLines(); !slices.Equal(got, []string{"see docs here", ""}) {
		t.Fatalf("scrollback %q", got)
	}
	if !bytes.Contains(term.Repaint(true), []byte("\x1b]8;id=x;https://example.com/a\x1b\\docs")) {
		t.Fatal("scrolled-off link not repainted")
	}
}

// isZeroAt and appendRow read a cell as two words: R and Fg, then Bg, A
// and E.
func TestCellLayout(t *testing.T) {
	var c cell
	if unsafe.Sizeof(c) != 16 || unsafe.Offsetof(c.Fg) != 4 || unsafe.Offsetof(c.Bg) != 8 || unsafe.Offsetof(c.E) != 14 {
		t.Fatalf("cell layout changed: size %d", unsafe.Sizeof(c))
	}
	// A Terminal, with the 8-byte header the allocator adds to objects of
	// 512 bytes or more that hold pointers, fits the 896-byte size class:
	// one more word costs every terminal 128 bytes.
	if n := unsafe.Sizeof(Terminal{}); n+8 > 896 {
		t.Fatalf("Terminal is %d bytes, over its 896-byte size class with the malloc header", n)
	}
}

// printable8 agrees with a byte-at-a-time check: every byte value in
// every position, and random words.
func TestPrintable8(t *testing.T) {
	check := func(b [8]byte) {
		want := true
		for _, x := range b {
			want = want && x >= 0x20 && x < 0x7f
		}
		if got := printable8(binary.LittleEndian.Uint64(b[:])); got != want {
			t.Fatalf("% x: got %v want %v", b, got, want)
		}
	}
	for pos := range 8 {
		for c := range 256 {
			b := [8]byte{'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a'}
			b[pos] = byte(c)
			check(b)
		}
	}
	for range 1 << 20 {
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], rand.Uint64())
		for i := range b {
			if rand.IntN(2) == 0 {
				b[i] = byte(0x20 + rand.IntN(0x5f))
			}
		}
		check(b)
	}
}

func TestBritishCharacterSet(t *testing.T) {
	for _, c := range []struct{ input, want string }{
		{"\x1b(A#q\x1b(B#", "£q#"},
		{"\x1b*A#\x1bN\u0301#", "#́#"},
		{"\x1b(0\x1b*A\x1bN#q\x1b[2b\x1b(B\x1b[2b", "£───qq"},
		{"\x1b+A\x1bO##\x1bo#\x1b(B\x0f#", "£#£#"},
		{"\x1b)A\x0e#\x1b7\x1b)B#\x1b8#", "££"},
	} {
		term := New(20, 4)
		for i := range len(c.input) {
			term.Write([]byte(c.input[i : i+1]))
		}
		if got := term.Line(0); got != c.want {
			t.Errorf("input%q got%q want%q", c.input, got, c.want)
		}
	}
}

func TestCharacterSetContinuation(t *testing.T) {
	for _, prefix := range []string{
		"\x1b*A\x1bN", "\x1b+A\x1bO", "\x1b)A\x0e\x1b7\x1b)B",
		"\x1b(A#\x1b7\x1b[?1049h\x1b*A\x1bN#\x1b7",
	} {
		term := New(20, 4, WithScrollback(20))
		term.Write([]byte(prefix))
		for _, binary := range []bool{false, true} {
			dst := New(20, 4, WithScrollback(20))
			if binary {
				if err := dst.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
					t.Fatal(err)
				}
			} else {
				dst.Write(term.Repaint(true))
			}
			source := New(1, 1)
			if err := source.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
				t.Fatal(err)
			}
			for _, next := range []string{"#q", "\x1b8#", "\x1b[?1049l\x1b8#", "\x1b[!p#"} {
				source.Write([]byte(next))
				dst.Write([]byte(next))
				dst.Compact()
				if source.Cursor() != dst.Cursor() || source.Modes() != dst.Modes() {
					t.Fatalf("charset continuation cursor/modes changed binary%v prefix%q next%q got%+v want%+v modes%+v/%+v saved%+v/%+v", binary, prefix, next, dst.Cursor(), source.Cursor(), dst.Modes(), source.Modes(), dst.saved, source.saved)
				}
				for y := range 4 {
					for x := range 20 {
						if source.Cell(x, y) != dst.Cell(x, y) {
							t.Fatalf("charset cell%d,%d binary%v prefix%q next%q got%+v want%+v", x, y, binary, prefix, next, dst.Cell(x, y), source.Cell(x, y))
						}
					}
				}
			}
		}
	}
}

func TestCSIPrefixChunkingPreservesSnapshots(t *testing.T) {
	for _, input := range []string{
		"\x1b[",
		"\x1b[31mred\x1b[0mplain",
		"\x1b[38:2::1:2:3mcolor\x1b[4:3munderline",
		"\x1b[?25l\x1b[>4;2m\x1b[1 q",
		"\x1b[5n\x1b[6n\x1b[c",
		"\x1b[99999999999999999999;2H",
		"\x1b[" + strings.Repeat("1;", maxParams+2) + "m",
		"\x1b[31;\x07\x00mtext\x1b[2\x18m",
		"\x1b]2;title\x1b[31mtext\x1bP$qm\x1b\\",
		"\x1b[1;2H\x1b[?1049h\x1b[2Jalt",
	} {
		for cut := 0; cut <= len(input); cut++ {
			var batchedReply, byteReply bytes.Buffer
			batched := New(10, 4, WithReply(&batchedReply))
			bytewise := New(10, 4, WithReply(&byteReply))
			batched.Write([]byte(input[:cut]))
			for i := 0; i < cut; i++ {
				bytewise.Write([]byte(input[i : i+1]))
			}
			if !bytes.Equal(batched.Snapshot(), bytewise.Snapshot()) || !bytes.Equal(batchedReply.Bytes(), byteReply.Bytes()) {
				t.Fatalf("input %q at byte %d: batching changed parser state or replies", input, cut)
			}
			var restored Terminal
			if err := restored.RestoreSnapshot(batched.Snapshot(), SnapshotLimits{}); err != nil {
				t.Fatalf("input %q at byte %d: restore: %v", input, cut, err)
			}
			future := []byte(input[cut:] + "\x1b[0mfuture\x1b[1;1Hdone")
			batched.Write(future)
			restored.Write(future)
			if !bytes.Equal(batched.Snapshot(), restored.Snapshot()) {
				t.Fatalf("input %q at byte %d: checkpoint continuation changed", input, cut)
			}
		}
	}
}

func TestCSIParameterSlotsAndSaturation(t *testing.T) {
	for count := 1; count <= maxParams+2; count++ {
		parts := make([]string, count)
		want := make([]uint32, count)
		for i := range parts {
			switch i % 5 {
			case 0:
				parts[i], want[i] = "0", 0
			case 1:
				parts[i], want[i] = "65535", 65535
			case 2:
				parts[i], want[i] = "65536", 65536
			case 3:
				parts[i], want[i] = "65537", 65536
			case 4:
				parts[i], want[i] = "999999999999999999999999", 65536
			}
		}
		for _, separator := range []string{";", ":"} {
			in := []byte("\x1b[" + strings.Join(parts, separator))
			for chunk := 1; chunk <= len(in); chunk++ {
				term := New(12, 3)
				for off := 0; off < len(in); off += chunk {
					term.Write(in[off:min(off+chunk, len(in))])
				}
				for i := 0; i < min(count, maxParams); i++ {
					if term.params[i] != want[i] || term.subParam(i) != (separator == ":" && i > 0) {
						t.Fatalf("count=%d separator=%q chunk=%d slot=%d: value=%d sub=%v, want %d", count, separator, chunk, i, term.params[i], term.subParam(i), want[i])
					}
				}
			}
		}
	}
}

func TestCSIColonAtLastSlotDoesNotBecomeMode(t *testing.T) {
	prefix := "\x1b[?" + strings.Repeat("999;", maxParams-2) + "999"
	for _, c := range []struct {
		separator string
		visible   bool
	}{{":", true}, {";", false}} {
		in := []byte(prefix + c.separator + "25l")
		for cut := 0; cut <= len(in); cut++ {
			term := New(12, 3)
			term.Write(in[:cut])
			restored := New(12, 3)
			restored.Write(term.Repaint(true))
			term.Write(in[cut:])
			restored.Write(in[cut:])
			if term.Cursor().Visible != c.visible || restored.Cursor().Visible != c.visible {
				t.Fatalf("separator=%q cut=%d: cursor visibility %v/%v, want %v", c.separator, cut, term.Cursor().Visible, restored.Cursor().Visible, c.visible)
			}
		}
	}
	term := New(12, 3)
	term.Write([]byte(prefix + ":25l\x1b[?25l"))
	if term.Cursor().Visible {
		t.Fatal("a new CSI retained an old colon bit")
	}
}

func TestCSIParameterBoundaryBehavior(t *testing.T) {
	for _, c := range []struct {
		name, in  string
		wantFg    Color
		wantX     int
		wantReply string
	}{
		{"saturated-column", "\x1b[65536G", 0, 11, ""},
		{"overflowed-column", "\x1b[999999999999999999999999G", 0, 11, ""},
		{"last-sgr-slot", "\x1b[" + strings.Repeat("0;", maxParams-1) + "31m", IndexedColor(1), 0, ""},
		{"extra-sgr-slot-ignored", "\x1b[" + strings.Repeat("0;", maxParams-1) + "31;32m", IndexedColor(1), 0, ""},
		{"colon-rgb-saturation", "\x1b[38:2::65536:65535:65537m", RGBColor(255, 255, 255), 0, ""},
		{"semicolon-rgb-saturation", "\x1b[38;2;65536;65535;65537m", RGBColor(255, 255, 255), 0, ""},
		{"saturated-mode-reply", "\x1b[?999999999999999999999999$p", 0, 0, "\x1b[?65536;0$y"},
	} {
		t.Run(c.name, func(t *testing.T) {
			in := []byte(c.in)
			for cut := 0; cut <= len(in); cut++ {
				var replies, restoredReplies bytes.Buffer
				term := New(12, 3, WithReply(&replies))
				term.Write(in[:cut])
				restored := New(12, 3, WithReply(&restoredReplies))
				restored.Write(term.Repaint(true))
				if restoredReplies.Len() != 0 {
					t.Fatalf("cut=%d: repaint emitted reply %q", cut, restoredReplies.Bytes())
				}
				restored.Write(in[cut:])
				term.Write(in[cut:])
				if term.cur.pen.Fg != c.wantFg || term.Cursor().X != c.wantX || replies.String() != c.wantReply {
					t.Fatalf("cut=%d: foreground=%v x=%d reply=%q", cut, term.cur.pen.Fg, term.Cursor().X, replies.Bytes())
				}
				assertContinuationState(t, term, restored, fmt.Sprintf("cut=%d", cut))
			}
		})
	}
}

func TestCSIParameterStorageDoesNotAllocate(t *testing.T) {
	term := New(12, 3)
	in := []byte("\x1b[38:2::10:20:30m\x1b[0;1;4:3;31m\x1b[0m")
	term.Write(in)
	if allocs := testing.AllocsPerRun(100, func() { term.Write(in) }); allocs != 0 {
		t.Fatalf("CSI parsing allocated %.1f times", allocs)
	}
}

func TestShortCSIBytewiseContinuation(t *testing.T) {
	prefix := []byte("\x1b[?2027h\x1b[?1049hALT\x1b[?1049lmain\r\n\x1b]8;;https://example.test/\x1b\\\x1b[4:3;58:2::1:2:3m👩‍💻\x1b7\x1b[1\"q")
	for code := byte(0x40); code <= 0x7e; code++ {
		params := []string{"", "00", "01", "09"}
		for value := 0; value < 100; value++ {
			params = append(params, fmt.Sprint(value))
		}
		for _, param := range params {
			input := append(bytes.Clone(prefix), []byte("\x1b["+param+string(code))...)
			var gotReply, wantReply bytes.Buffer
			a := New(12, 5, WithReply(&gotReply))
			b := New(12, 5, WithReply(&wantReply))
			a.Write(input)
			for i := range input {
				b.Write(input[i : i+1])
			}
			if !bytes.Equal(a.Snapshot(), b.Snapshot()) || !bytes.Equal(gotReply.Bytes(), wantReply.Bytes()) {
				t.Fatalf("CSI %s%c changed complete input", param, code)
			}
			for _, suffix := range []string{"\x1b[3", "1mX\x1b[6n", "\x1b8Y\x1b[?1049h\x1b[?1049l"} {
				a.Write([]byte(suffix))
				for i := range suffix {
					b.Write([]byte(suffix[i : i+1]))
				}
				a.Resize(9, 4)
				b.Resize(9, 4)
				if !bytes.Equal(a.Snapshot(), b.Snapshot()) || !bytes.Equal(gotReply.Bytes(), wantReply.Bytes()) {
					t.Fatalf("CSI %s%c changed future input %q", param, code, suffix)
				}
			}
		}
	}
}

func TestOSCEndsAtEscape(t *testing.T) {
	for _, suffix := range []string{"\\", "[31mX", "X"} {
		term := New(8, 3)
		term.Write([]byte("\x1b]2;old\a\x1b]2;new\x1b"))
		if term.Title() != "new" {
			t.Fatalf("title before ST %q", term.Title())
		}
		if got := term.appendContinuation(nil); !bytes.Equal(got, []byte{0x1b}) {
			t.Fatalf("continuation %q, want ESC", got)
		}
		term.Write([]byte(suffix))
		if term.Title() != "new" {
			t.Fatalf("title after %q: %q", suffix, term.Title())
		}
		if suffix == "[31mX" && (term.Cell(0, 0).Rune != 'X' || term.Cell(0, 0).Style.Fg != IndexedColor(1)) {
			t.Fatal("escape after OSC did not execute")
		}
	}
}

func TestOSCCancel(t *testing.T) {
	for _, cancel := range []byte{0x18, 0x1a} {
		term := New(8, 3)
		term.Write(append([]byte("\x1b]2;old\a\x1b]2;cancel"), cancel, 'X'))
		if term.Title() != "old" || term.Cell(0, 0).Rune != 'X' {
			t.Fatalf("cancel %#x: title %q, cell %+v", cancel, term.Title(), term.Cell(0, 0))
		}
	}
}

func TestStringsEndBeforeNextByte(t *testing.T) {
	for _, prefix := range []string{"\x1bXignored", "\x1b^ignored", "\x1b_ignored", "\x1bPqignored", "\x1bP$qm"} {
		for _, end := range []byte{0x1b, 0x18, 0x1a} {
			var replies bytes.Buffer
			term := New(8, 3, WithReply(&replies))
			term.Write(append([]byte(prefix), end))
			if prefix == "\x1bP$qm" && replies.String() != "\x1bP1$r0m\x1b\\" {
				t.Fatalf("DCS end %#x: replies %q", end, replies.String())
			}
			if end == 0x1b {
				term.Write([]byte("[31m"))
			}
			term.Write([]byte("X"))
			if term.Cell(0, 0).Rune != 'X' {
				t.Fatalf("%q end %#x did not return to ground", prefix, end)
			}
		}
	}
}

func TestAlternateScreensCopyPendingWrap(t *testing.T) {
	for _, mode := range []int{47, 1047, 1049} {
		term := New(4, 3)
		fmt.Fprintf(term, "main\x1b[?%dhabcd", mode)
		if term.Line(0) != "" || term.Line(1) != "abcd" || !term.Cursor().PendingWrap {
			t.Fatalf("mode%d did not carry main's pending wrap into alternate: %q %+v", mode, term.Lines(), term.Cursor())
		}
		fmt.Fprintf(term, "\x1b[?%dl", mode)
		if term.Cursor().PendingWrap != (mode != 1047) {
			t.Fatalf("mode%d left the wrong pending wrap after exit: %+v", mode, term.Cursor())
		}
	}
}

func TestMarginModeResetKeepsPendingWrap(t *testing.T) {
	for _, origin := range []bool{false, true} {
		term := New(8, 3)
		term.Write([]byte("\x1b[?69h\x1b[3;4s"))
		if origin {
			term.Write([]byte("\x1b[?6h"))
		} else {
			term.Write([]byte("\x1b[3G"))
		}
		term.Write([]byte("ab\x1b[?69l"))
		if cur := term.Cursor(); cur.X != 3 || !cur.PendingWrap || term.lrmm {
			t.Fatal("margin reset consumed pending wrap", cur)
		}
		term.Write([]byte("\tX"))
		if term.Cell(0, 1).Text() != "X" || term.Cursor().Y != 1 {
			t.Fatal("margin reset changed future wrapping", term.Cursor())
		}
	}
}

func TestPendingWrapSurvivesModeAndTabs(t *testing.T) {
	for _, mode := range []string{"", "\x1b[?7l"} {
		for _, tab := range []string{"\t", "\x1b[I", "\x1b[Z"} {
			term := New(6, 3)
			term.Write([]byte("abcdef" + mode + tab))
			wantX := 5
			if tab == "\x1b[Z" {
				wantX = 0
			}
			if cur := term.Cursor(); cur.X != wantX || cur.Y != 0 || !cur.PendingWrap {
				t.Fatalf("mode=%q tab=%q: %+v", mode, tab, cur)
			}
			term.Compact()
			restored := New(6, 3)
			restored.Write(term.Repaint(false))
			if restored.Cursor() != term.Cursor() || restored.Modes() != term.Modes() {
				t.Fatalf("repaint changed pending wrap: mode=%q tab=%q", mode, tab)
			}
			for _, e := range []*Terminal{term, restored} {
				e.Write([]byte("\x1b[?7hX"))
				if cur := e.Cursor(); cur.X != 1 || cur.Y != 1 || cur.PendingWrap || e.Cell(0, 1).Text() != "X" {
					t.Fatalf("mode=%q tab=%q did not wrap next print: %+v", mode, tab, cur)
				}
				if e.Wrapped(1) != (tab != "\x1b[Z") {
					t.Fatalf("mode=%q tab=%q marked a soft wrap away from the physical edge", mode, tab)
				}
			}
		}
	}
}

func TestWideGlyphWithoutRoomMatchesNative(t *testing.T) {
	for _, text := range []string{"AAAA", "AAAAA"} {
		term := New(5, 3)
		term.Write([]byte("\x1b[?7l" + text + "🚨"))
		if term.Cursor().X != 4 || term.Cursor().PendingWrap || term.Lines()[0] != text {
			t.Fatalf("did not drop unprintable wide glyph: %+v %q", term.Cursor(), term.Lines())
		}
		term.Write([]byte("\x1b[?7hX"))
		if term.Cursor().Y != 0 || term.Cell(4, 0).Text() != "X" || !term.Cursor().PendingWrap {
			t.Fatal("dropped glyph changed future wrapping", term.Cursor())
		}
	}
	term := New(1, 3)
	term.Write([]byte("\x1b[1;31m😀"))
	if c := term.Cell(0, 0); c.Text() != "" || c.Style.Fg != IndexedColor(1) || c.Style.Attrs&AttrBold == 0 || !term.Cursor().PendingWrap {
		t.Fatal("one-column dropped glyph lost native blank style/pending wrap", c, term.Cursor())
	}
	copy := New(1, 3)
	copy.Write(term.Repaint(false))
	if copy.Cell(0, 0) != term.Cell(0, 0) || copy.Cursor() != term.Cursor() {
		t.Fatal("one-column repaint lost dropped-wide state")
	}
	term.Write([]byte("X"))
	if term.Cursor().Y != 1 || term.Cell(0, 1).Text() != "X" {
		t.Fatal("one-column dropped glyph failed to wrap the next print")
	}
}

func TestSmoothScrollingModeSavesAndRepaints(t *testing.T) {
	var replies bytes.Buffer
	term := New(12, 3, WithReply(&replies))
	term.Write([]byte("\x1b[?4h\x1b[?4s\x1b[?4l\x1b[?4r\x1b[?4$p"))
	if !term.Modes().SmoothScrolling || replies.String() != "\x1b[?4;1$y" {
		t.Fatal(term.Modes(), replies.String())
	}
	term.Compact()
	copy := New(12, 3)
	copy.Write(term.Repaint(false))
	if !copy.Modes().SmoothScrolling {
		t.Fatal("repaint discarded smooth scrolling")
	}
	copy.Write([]byte("\x1b[?4l\x1b[?4r"))
	if !copy.Modes().SmoothScrolling {
		t.Fatal("repaint discarded the saved smooth scrolling mode")
	}
}

func TestModifierRequiresImmediateBase(t *testing.T) {
	for _, tc := range []struct {
		name, prefix string
		cursor       int
	}{
		{"direct", "👋\x1b[?2027h", 3},
		{"control", "👋\u200b\x1b[?2027h", 5},
		{"extend", "👋\u0301\x1b[?2027h", 5},
		{"selector-ignored", "👋\ufe0f\x1b[?2027h", 3},
		{"joiner", "👋\u200d\x1b[?2027h", 5},
		{"prepend", "👋\u0600\x1b[?2027h", 4},
		{"executed-NUL", "👋\x00\x1b[?2027h", 3},
		{"text-selector-mode-off", "✍\ufe0f\x1b[?2027h", 2},
		{"text-selector-mode-on", "\x1b[?2027h✍\ufe0f", 5},
		{"extend-mode-on", "\x1b[?2027h👋\u0301", 5},
		{"repeated-modifier", "\x1b[?2027h👋🏿", 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := []byte(tc.prefix + "🏿X")
			for cut := 0; cut <= len(input); cut++ {
				term := New(12, 3)
				term.Write(input[:cut])
				term.Write(input[cut:])
				if term.Cursor().X != tc.cursor {
					t.Fatalf("cut%d cursor%+v wantX%d", cut, term.Cursor(), tc.cursor)
				}
				if tc.cursor == 5 && term.Cell(2, 0).Rune != '🏿' {
					t.Fatalf("modifier crossed intervening scalar: %+v", term.Cell(0, 0))
				}
				assertGraphemeInvariants(t, term)
				checkRepaint(t, term)
			}
		})
	}
}
