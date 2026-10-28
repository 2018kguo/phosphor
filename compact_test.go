package phosphor

import (
	"bytes"
	"fmt"
	"runtime"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCompactKeepsNonzeroCells(t *testing.T) {
	term := New(5, 6)
	link := term.internExtra(cellExtra{link: ";https://example.com"})
	values := []cell{{}, {R: ' '}, {R: spacerHead}, {Bg: IndexedColor(1)}, {A: AttrProtected}, {E: link}}
	term.alt = make([][]cell, term.rows)
	for _, screen := range [][][]cell{term.main, term.alt} {
		for y, c := range values {
			screen[y] = make([]cell, term.cols)
			screen[y][0] = c
		}
	}
	term.mainWrap = make([]rowFlags, term.rows)
	term.altWrap = make([]rowFlags, term.rows)
	term.altWrap[1] = rowWrap
	term.ClearDirty()
	term.scratch = make([]byte, 0, 4096)
	term.osc = []byte("2;completed title")
	term.Compact()
	if term.main[0] != nil || term.alt[0] != nil || term.mainWrap != nil || term.scratch != nil || term.osc != nil {
		t.Fatal("disposable storage retained")
	}
	for _, screen := range [][][]cell{term.main, term.alt} {
		for y, c := range values[1:] {
			if screen[y+1] == nil || screen[y+1][0] != c {
				t.Fatalf("nonzero cell %d changed: %+v", y+1, screen[y+1])
			}
		}
	}
	if term.altWrap[1]&rowWrap == 0 || term.Dirty() {
		t.Fatal("wrap or damage state changed")
	}
}

func assertCompactEquivalent(t *testing.T, original, compacted *Terminal) {
	t.Helper()
	assertContinuationState(t, original, compacted, "Compact")
	if a, b := original.Repaint(true), compacted.Repaint(true); !bytes.Equal(a, b) {
		t.Fatalf("repaint differs:\n%q\n%q", a, b)
	}
	if a, b := original.Cell(original.cur.x, original.cur.y), compacted.Cell(compacted.cur.x, compacted.cur.y); a != b {
		t.Fatalf("cursor cell differs: %+v / %+v", a, b)
	}
	for y := range original.rows {
		if original.RowDirty(y) != compacted.RowDirty(y) {
			t.Fatalf("row %d damage changed", y)
		}
	}
}

func TestCompactEmptyWrappedRowRepaint(t *testing.T) {
	a, b := New(6, 10), New(6, 10)
	for _, v := range []byte("0\n0") {
		a.Write([]byte{v})
		b.Write([]byte{v})
		b.Compact()
	}
	a.Resize(1, 3)
	b.Resize(1, 3)
	b.Compact()
	assertContinuationState(t, a, b, "before repaint")
	aRepaint, bRepaint := a.Repaint(true), b.Repaint(true)
	ar, br := New(1, 3), New(1, 3)
	ar.Write(aRepaint)
	br.Write(bRepaint)
	assertContinuationState(t, ar, br, "after repaint")
	assertContinuationState(t, a, ar, "original repaint")
	assertContinuationState(t, b, br, "compacted repaint")
	// ANSI repaint does not export the source's REP register. The two
	// exports must still agree with each other before another print.
	repeatA, repeatB := New(1, 3), New(1, 3)
	repeatA.Write(aRepaint)
	repeatB.Write(bRepaint)
	repeatA.Write([]byte("\x1b[3b"))
	repeatB.Write([]byte("\x1b[3b"))
	assertContinuationState(t, repeatA, repeatB, "immediate REP")
	for _, next := range []string{"X\x1b[3b", "\x1b8X", "\r\nnext", "\x1b[?47hALT\x1b[?47l"} {
		a.Write([]byte(next))
		b.Write([]byte(next))
		ar.Write([]byte(next))
		br.Write([]byte(next))
		assertContinuationState(t, ar, br, fmt.Sprintf("after %q", next))
		assertContinuationState(t, a, ar, fmt.Sprintf("original after %q", next))
		assertContinuationState(t, b, br, fmt.Sprintf("compacted after %q", next))
	}
	a.Resize(4, 2)
	b.Resize(4, 2)
	ar.Resize(4, 2)
	br.Resize(4, 2)
	assertContinuationState(t, ar, br, "after future resize")
	assertContinuationState(t, a, ar, "original after future resize")
	assertContinuationState(t, b, br, "compacted after future resize")
	if !bytes.Equal(aRepaint, bRepaint) {
		t.Fatalf("equivalent replay futures have different repaint bytes:\n%q\n%q", aRepaint, bRepaint)
	}
}

func TestCompactTrace(t *testing.T) {
	cases := []struct{ name, input string }{
		{"erase-and-rep", "á\x1b[2J\x1b[H$ \x1b[3b"},
		{"saved-links", "\x1b]8;;https://example.com\x1b\\á\x1b7\x1b[2J\x1b]8;;\x1b\\\x1b8X\x1b[3b"},
		{"alternate-screen", "main\x1b7\x1b[?1049h\x1b[2J\x1b[HALT\x1b[?1049l\x1b8X"},
		{"kept-alternate", "\x1b[?47h\x1b[2Jalt\x1b[?47lmain\x1b[?47hX"},
		{"csi", "\x1b[?2004h\x1b[38:2::10:20:30mX\x1b[?1$p\x1b[6n"},
		{"osc", "\x1b]2;title\x1b\\X\x1b]10;?\a"},
		{"osc-overflow", "\x1b]2;" + strings.Repeat("x", maxOSC+1) + "\aX"},
		{"dcs", "\x1b[31m\x1bP$qm\x1b\\X"},
		{"ignored-dcs", "\x1bPqignored\x1b\\X"},
		{"ignored-string", "\x1b_ignored\x1b\\X"},
		{"utf8", "😀é\xe6\xbc\xa2X"},
		{"scrollback", strings.Repeat("\x1b[32mlog\x1b[0m\r\n", 50) + "\x1b[2J\x1b[H$ "},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var aReply, bReply bytes.Buffer
			aBells, bBells := 0, 0
			a := New(12, 5, WithReply(&aReply), WithBell(func() { aBells++ }))
			b := New(12, 5, WithReply(&bReply), WithBell(func() { bBells++ }))
			a.ClearDirty()
			b.ClearDirty()
			for _, v := range []byte(c.input) {
				a.Write([]byte{v})
				b.Write([]byte{v})
				b.Compact()
				b.Compact()
				assertCompactEquivalent(t, a, b)
				if !bytes.Equal(aReply.Bytes(), bReply.Bytes()) || aBells != bBells {
					t.Fatalf("callbacks differ after byte %02x", v)
				}
			}
			for _, size := range [][2]int{{7, 3}, {20, 8}, {12, 5}} {
				a.Resize(size[0], size[1])
				b.Resize(size[0], size[1])
				b.Compact()
				assertCompactEquivalent(t, a, b)
			}
			// Continue through a checkpoint, not only an equivalent current view.
			originalCheckpoint, compactedCheckpoint := New(12, 5), New(12, 5)
			originalCheckpoint.Write(a.Repaint(true))
			compactedCheckpoint.Write(b.Repaint(true))
			compactedCheckpoint.Compact()
			for _, next := range []string{"\x1b8", "\x1b[3b", "\r\n漢", "\x1b[?1049hALT\x1b[?1049l"} {
				a.Write([]byte(next))
				b.Write([]byte(next))
				originalCheckpoint.Write([]byte(next))
				compactedCheckpoint.Write([]byte(next))
				b.Compact()
				compactedCheckpoint.Compact()
				assertCompactEquivalent(t, a, b)
				assertCompactEquivalent(t, originalCheckpoint, compactedCheckpoint)
			}
		})
	}
}

func FuzzCompactTrace(f *testing.F) {
	f.Add([]byte("abc\x1b[2J\x1b[H$ \x1b[3b"), uint8(5), uint8(3))
	f.Add([]byte("\x1b]2;long title\a\x1b[38:2::10:20:30m😀\x1bP$qm\x1b\\"), uint8(8), uint8(4))
	f.Fuzz(func(t *testing.T, input []byte, cols, rows uint8) {
		if len(input) > 4096 {
			input = input[:4096]
		}
		var aReply, bReply bytes.Buffer
		a := New(int(cols%30)+1, int(rows%12)+1, WithReply(&aReply), WithScrollback(20))
		b := New(int(cols%30)+1, int(rows%12)+1, WithReply(&bReply), WithScrollback(20))
		a.ClearDirty()
		b.ClearDirty()
		fuzzOps(input, 30, 12, func(p []byte) {
			for _, v := range p {
				a.Write([]byte{v})
				b.Write([]byte{v})
				b.Compact()
			}
		}, func(c, r int) { a.Resize(c, r); b.Resize(c, r); b.Compact() })
		assertCompactEquivalent(t, a, b)
		if !bytes.Equal(aReply.Bytes(), bReply.Bytes()) {
			t.Fatal("query replies changed")
		}
		for _, p := range []string{"\x1b\\mX", "\x1b[3b", "\x1b8", "\r\nnext\x1b[6n"} {
			a.Write([]byte(p))
			b.Write([]byte(p))
			b.Compact()
			assertCompactEquivalent(t, a, b)
		}
		if !bytes.Equal(aReply.Bytes(), bReply.Bytes()) {
			t.Fatal("future query replies changed")
		}
	})
}

// Compact should bring an erased idle screen back near its fresh-prompt
// footprint. Report actual retained heap, without asserting a noisy RSS limit.
func BenchmarkCompactIdleMemory(b *testing.B) {
	for _, mode := range []string{"fresh", "erased", "compacted"} {
		b.Run(mode, func(b *testing.B) {
			const count = 512
			for b.Loop() {
				terms := make([]*Terminal, count)
				runtime.GC()
				var before, after runtime.MemStats
				runtime.ReadMemStats(&before)
				for i := range terms {
					t := New(80, 24)
					if mode != "fresh" {
						for y := range 24 {
							t.Write(fmt.Appendf(nil, "\x1b[%d;1H%s", y+1, strings.Repeat("x", 80)))
						}
						t.Write([]byte("\x1b[2J\x1b[H"))
					}
					t.Write([]byte("$ "))
					if mode == "compacted" {
						t.Compact()
					}
					terms[i] = t
				}
				runtime.GC()
				runtime.GC()
				runtime.ReadMemStats(&after)
				b.ReportMetric(float64(int64(after.HeapAlloc)-int64(before.HeapAlloc))/count, "heap-B/term")
				if !slices.Equal(terms[count-1].Lines()[:2], []string{"$", ""}) {
					b.Fatal("prompt changed")
				}
				runtime.KeepAlive(terms)
			}
		})
	}
}

func poolChurnInput() string {
	var b strings.Builder
	for i := range 64 {
		fmt.Fprintf(&b, "\x1b[%d;%dH\x1b]8;;https://unused/%d/%s\x1b\\a%c%c", 1+i/16, 1+4*(i%16), i, strings.Repeat("x", 512), rune(0x300+i%32), rune(0x300+i/32))
	}
	b.WriteString("\x1b]8;;\x1b\\\x1b[2J\x1b[H$ ")
	return b.String()
}

func assertPoolEquivalent(t *testing.T, original, compacted *Terminal) {
	t.Helper()
	assertCompactEquivalent(t, original, compacted)
	for _, screen := range []struct {
		name string
		a, b [][]cell
	}{{"main", original.main, compacted.main}, {"alternate", original.alt, compacted.alt}} {
		if len(screen.a) != len(screen.b) {
			t.Fatalf("%s screen height changed", screen.name)
		}
		for y := range screen.a {
			for x := range original.cols {
				a, b := Cell{Width: 1}, Cell{Width: 1}
				if screen.a[y] != nil {
					a = original.publicCell(screen.a[y], x)
				}
				if screen.b[y] != nil {
					b = compacted.publicCell(screen.b[y], x)
				}
				if a != b {
					t.Fatalf("%s cell %d,%d changed: %+v / %+v", screen.name, x, y, a, b)
				}
			}
		}
	}
}

func TestCompactReleasesDeadPools(t *testing.T) {
	input := []byte(poolChurnInput())
	original, compacted := New(80, 24), New(80, 24)
	original.Write(input)
	compacted.Write(input)
	if compacted.x == nil || len(compacted.x.rare) == 0 || len(compacted.x.clusters) == 0 {
		t.Fatal("trace did not populate both pools")
	}
	compacted.ClearDirty()
	original.ClearDirty()
	compacted.Compact()
	if compacted.x != nil {
		t.Fatal("erased cells kept dead Unicode or hyperlink storage")
	}
	assertPoolEquivalent(t, original, compacted)
	for _, next := range []string{"\x1b[2b", "\x1b]8;;https://new\x1b\\é\x1b]8;;\x1b\\", "\r\nnext"} {
		original.Write([]byte(next))
		compacted.Write([]byte(next))
		assertPoolEquivalent(t, original, compacted)
	}
}

func TestCompactKeepsLivePoolReferences(t *testing.T) {
	prelude := "\x1b]8;;https://main-cell\x1b\\á\x1b]8;;https://main-save\x1b\\\x1b7" +
		"\x1b]8;;https://1049-save\x1b\\\x1b[?1049h\x1b[H\x1b]8;;https://alt-cell\x1b\\é" +
		"\x1b]8;;https://alt-save\x1b\\\x1b7"
	var churn strings.Builder
	for i := range 60 {
		fmt.Fprintf(&churn, "\x1b[4;1H\x1b]8;;https://unused/%d\x1b\\a%c", i, rune(0x300+i))
	}
	// The last cluster is erased, while REP still remembers its base scalar.
	churn.WriteString("\x1b[4;1H\x1b[2K\x1b]8;;https://current-pen\x1b\\")
	original, compacted := New(12, 5), New(12, 5)
	for _, term := range []*Terminal{original, compacted} {
		term.Write([]byte(prelude + churn.String()))
		term.ClearDirty()
	}
	compacted.Compact()
	assertPoolEquivalent(t, original, compacted)
	if compacted.lastChar != 'a' || compacted.lastChar != original.lastChar {
		t.Fatal("REP lost its independently printed base scalar")
	}
	for _, next := range []string{"\x1b[2b", "\x1b8X", "\x1b[?1049lY", "\x1b8Z", "\x1b[?47hkept\x1b[?47l"} {
		original.Write([]byte(next))
		compacted.Write([]byte(next))
		compacted.Compact()
		assertPoolEquivalent(t, original, compacted)
	}
	for _, term := range []*Terminal{original, compacted} {
		term.Resize(8, 8)
		term.Resize(14, 5)
	}
	compacted.Compact()
	assertPoolEquivalent(t, original, compacted)
}

func TestCompactKeepsSerializedHistory(t *testing.T) {
	var input strings.Builder
	for i := range 30 {
		fmt.Fprintf(&input, "\x1b]8;;https://history/%d\x1b\\line %d á\x1b]8;;\x1b\\\r\n", i, i)
	}
	input.WriteString("\x1b[2J\x1b[H$ ")
	original, compacted := New(40, 4), New(40, 4)
	original.Write([]byte(input.String()))
	compacted.Write([]byte(input.String()))
	if compacted.ScrollbackLen() == 0 {
		t.Fatal("trace did not populate history")
	}
	compacted.Compact()
	if compacted.x != nil {
		t.Fatal("self-contained history retained unused lookup tables")
	}
	assertPoolEquivalent(t, original, compacted)
	for _, term := range []*Terminal{original, compacted} {
		term.Resize(40, 12)
		term.Write([]byte("\x1b[?1049hnew á\x1b[?1049l"))
	}
	assertPoolEquivalent(t, original, compacted)
}

func TestCompactDoesNotRebuildLivePools(t *testing.T) {
	term := New(12, 4)
	term.Write([]byte("\x1b]8;;https://live\x1b\\á\x1b7"))
	term.Compact()
	cluster, link := &term.x.clusters[0], &term.x.rare[0]
	if n := testing.AllocsPerRun(100, term.Compact); n != 0 {
		t.Fatalf("repeated compaction of unchanged live pools allocated %g times", n)
	}
	if cluster != &term.x.clusters[0] || link != &term.x.rare[0] {
		t.Fatal("unchanged live pool storage was replaced")
	}
}

func TestCompactPoolsKeepsPendingInput(t *testing.T) {
	for _, c := range []struct{ prefix, suffix string }{
		{"\x1b]8;;https://partial", "\x1b\\X"},
		{"\x1b[38:2::10:20:", "30mX"},
		{"\x1bP$q", "m\x1b\\X"},
		{"\xf0\x9f", "\x98\x80X"},
	} {
		original, compacted := New(80, 24), New(80, 24)
		for _, term := range []*Terminal{original, compacted} {
			term.Write([]byte(poolChurnInput() + c.prefix))
		}
		compacted.Compact()
		if compacted.x != nil {
			t.Fatal("pending input kept dead lookup tables")
		}
		original.Write([]byte(c.suffix))
		compacted.Write([]byte(c.suffix))
		assertPoolEquivalent(t, original, compacted)
	}
}

func TestCompactPoolsKeepsTabsAndSavedModes(t *testing.T) {
	original, compacted := New(80, 24), New(80, 24)
	for _, term := range []*Terminal{original, compacted} {
		term.Write([]byte(poolChurnInput() + "\x1b[3g\x1b[1;5H\x1bH\x1b[?2004s\x1b[?2004h"))
	}
	compacted.Compact()
	if compacted.x == nil || compacted.x.tabs == nil || len(compacted.x.savedModes) == 0 {
		t.Fatal("other live extra state was collected with the lookup tables")
	}
	for _, term := range []*Terminal{original, compacted} {
		term.Write([]byte("\x1b[1;1H\tX\x1b[?2004r"))
	}
	assertPoolEquivalent(t, original, compacted)
	if compacted.Cell(4, 0).Rune != 'X' || compacted.Modes().BracketedPaste {
		t.Fatal("custom tab or saved mode no longer takes effect")
	}
}

func TestCompactDropsLinksFromSavedStyle(t *testing.T) {
	term := New(12, 4)
	term.Write([]byte("\x1b]8;;https://old\x1b\\\x1b[?1049h\x1b[?1049l\x1b]8;;\x1b\\$ "))
	term.Compact()
	if term.x != nil || term.saved == nil || !term.saved.screen[0].valid {
		t.Fatal("1049 must retain its primary save without retaining the hyperlink")
	}
	// A subsequent table must not revisit the invalid slot's old index.
	for i := range 40 {
		term.Write(fmt.Appendf(nil, "\x1b]8;;https://new/%d\x1b\\", i))
	}
	term.Write([]byte("\x1b[?1049h\x1b[?1049lX"))
	if got := term.Cell(2, 0).Link.URI; got != "" {
		t.Fatalf("new hyperlink table restored %q", got)
	}
}

func TestCompactRetainsShared1049Save(t *testing.T) {
	original, compacted := New(12, 4), New(12, 4)
	for _, term := range []*Terminal{original, compacted} {
		term.Write([]byte("\x1b[2;3H\x1b[31m\x1b[?1049hALT\x1b[?1049l"))
	}
	compacted.Compact()
	if compacted.saved == nil || !compacted.saved.screen[0].valid {
		t.Fatal("compaction lost the primary DECSC slot written by 1049")
	}
	for _, next := range []string{"\x1b8X", "\x1b[2b", "\x1b[?47h\x1b[Hkept á\x1b[?47l", "\x1b[?47hX\x1b[?47l"} {
		original.Write([]byte(next))
		compacted.Write([]byte(next))
		compacted.Compact()
		assertPoolEquivalent(t, original, compacted)
	}
}

func TestCompactLargeLivePools(t *testing.T) {
	term := New(40, 12)
	for i := range 300 {
		term.Write(fmt.Appendf(nil, "\x1b[%d;%dH\x1b]8;;https://live/%d\x1b\\%c%c", 1+i/40, 1+i%40, i, 'a'+rune(i%26), rune(0x300+i/26)))
	}
	term.ClearDirty()
	before := term.Repaint(true)
	cluster, link := &term.x.clusters[0], &term.x.rare[0]
	term.Compact()
	if !bytes.Equal(before, term.Repaint(true)) || term.Dirty() {
		t.Fatal("large liveness map changed the rendered state")
	}
	if cluster != &term.x.clusters[0] || link != &term.x.rare[0] {
		t.Fatal("large unchanged pools were rebuilt")
	}
}

func historyPoolInput(kind string) string {
	var b strings.Builder
	printCell := func(group string, i int) {
		fmt.Fprintf(&b, "\x1b[38;5;%dm", i%16)
		if kind != "clusters" {
			fmt.Fprintf(&b, "\x1b]8;;https://history/%s/%d\x1b\\", group, i)
		}
		if kind != "links" {
			fmt.Fprintf(&b, "a%c%c", rune(0x300+i%64), rune(0x300+i/64))
		} else {
			b.WriteByte('a')
		}
	}
	// Keep enough entries live to give the uncollected control a large
	// collection threshold. The first three rows later become history.
	for y := range 6 {
		fmt.Fprintf(&b, "\x1b[%d;1H", y+1)
		for x := range 100 {
			printCell("warm", 100*y+x)
		}
	}
	b.WriteString("\x1b]8;;\x1b\\\x1b[4;1H\x1b[J\x1b[6;1H\r\n\r\n\r\n")
	// Existing main rows will be rewrapped before history is imported.
	b.WriteString("\x1b[2;1H")
	for x := range 80 {
		printCell("main", 700+x)
	}
	b.WriteString("\x1b]8;;https://saved-main\x1b\\\x1b7\x1b]8;;\x1b\\")
	// A kept alternate screen and its saved pen are also collection roots.
	b.WriteString("\x1b[?47h\x1b[1;1H")
	printCell("alt", 800)
	b.WriteString("\x1b]8;;https://saved-alt\x1b\\\x1b7\x1b]8;;\x1b\\\x1b[?47l\x1b[6;1H$ ")
	return b.String()
}

func TestCompactResizeHistoryPools(t *testing.T) {
	for _, kind := range []string{"clusters", "links", "mixed"} {
		for _, size := range [][2]int{{100, 10}, {40, 18}} {
			for _, alt := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%dx%d/alt=%v", kind, size[0], size[1], alt), func(t *testing.T) {
					original, compacted := New(100, 6), New(100, 6)
					for _, term := range []*Terminal{original, compacted} {
						term.Write([]byte(historyPoolInput(kind)))
						if term.ScrollbackLen() != 3 {
							t.Fatalf("trace produced %d history lines, want three", term.ScrollbackLen())
						}
						if alt {
							term.Write([]byte("\x1b[?1049h\x1b[H\x1b]8;;https://active-alt\x1b\\ALT á\x1b7"))
						}
					}
					compacted.Compact()
					original.Resize(size[0], size[1])
					compacted.Resize(size[0], size[1])
					assertPoolEquivalent(t, original, compacted)
					// Continue across normal collection thresholds after the
					// imported and rewrapped rows are installed.
					for i := range 100 {
						next := fmt.Appendf(nil, "\x1b[%d;%dH\x1b]8;;https://future/%d\x1b\\z%c%c", size[1]-1, size[0], i, rune(0x300+i%64), rune(0x300+i/64))
						original.Write(next)
						compacted.Write(next)
					}
					assertPoolEquivalent(t, original, compacted)
					for _, next := range []string{"\x1b8X", "\x1b[?1049l", "\x1b8Y", "\x1b[?47h\x1b8Z\x1b[?47l", "\x1b[2b"} {
						original.Write([]byte(next))
						compacted.Write([]byte(next))
						assertPoolEquivalent(t, original, compacted)
					}
					for _, later := range [][2]int{{25, 22}, {100, 12}} {
						original.Resize(later[0], later[1])
						compacted.Resize(later[0], later[1])
						assertPoolEquivalent(t, original, compacted)
						compacted.Compact()
						assertPoolEquivalent(t, original, compacted)
					}
				})
			}
		}
	}
}

func TestResizeHistoryPoolsStayBoundedWithoutOutput(t *testing.T) {
	for _, kind := range []string{"clusters", "links", "mixed"} {
		t.Run(kind, func(t *testing.T) {
			var input strings.Builder
			for y := range 4 {
				fmt.Fprintf(&input, "\x1b[%d;1H", y+1)
				for x := range 100 {
					if kind != "clusters" {
						fmt.Fprintf(&input, "\x1b]8;;https://history/%d/%d\x1b\\", y, x)
					}
					input.WriteByte('a')
					if kind != "links" {
						fmt.Fprintf(&input, "%c%c", rune(0x300+x%64), rune(0x300+(100*y+x)/64))
					}
				}
			}
			input.WriteString("\x1b]8;;\x1b\\\x1b[2;1H\x1b[J\x1b[4;1H\r\n$ ")
			term := New(100, 4)
			term.Write([]byte(input.String()))
			term.Compact()
			before := term.Repaint(true)
			for cycle := range 100 {
				term.Resize(100, 5)
				first := term.Cell(0, 0)
				if first.Rune != 'a' || kind != "links" && first.Combining != "\u0300\u0300" || kind != "clusters" && first.Link.URI != "https://history/0/0" {
					t.Fatalf("cycle %d lost imported metadata: %+v", cycle, first)
				}
				term.Resize(100, 4)
				if got := term.Repaint(true); string(got) != string(before) {
					t.Fatalf("cycle %d changed the checkpoint", cycle)
				}
				// One retained history row has no live pool IDs. Allow a
				// bounded cache, but never one entry set per resize cycle.
				if term.x != nil && len(term.x.clusters)+len(term.x.rare) > 4*100 {
					t.Fatalf("cycle %d retained %d clusters and %d attribute pairs", cycle, len(term.x.clusters), len(term.x.rare))
				}
			}
		})
	}
}

func TestResizePlainHistoryDoesNotAllocateExtra(t *testing.T) {
	term := New(12, 3)
	term.Write([]byte(strings.Repeat("plain\r\n", 8)))
	if term.x != nil {
		t.Fatal("plain history trace allocated extra state")
	}
	term.Compact()
	term.Resize(8, 10)
	if term.x != nil {
		t.Fatal("plain history import allocated Unicode or hyperlink storage")
	}
}

func TestFullClusterStillChangesGeometry(t *testing.T) {
	wide := "☔" + strings.Repeat("\u0301", 62) + "\u200d☔"
	narrow := "❤" + strings.Repeat("\u0301", 63) + "\u200d"
	for _, c := range []struct {
		mode, base, overflow string
		width                int
	}{
		{"\x1b[?2027h", wide, "\ufe0e", 1},
		{"\x1b[?2027h", narrow, "☔", 2},
		{"", narrow, "\x1b[?2027h☔", 2},
		{"\x1b[?2027h", wide, "\ufe0e\ufe0f\ufe0e", 1},
	} {
		term := New(8, 3)
		term.Write([]byte(c.mode + c.base))
		term.Write([]byte(c.overflow))
		if got := term.Cell(0, 0); got.Text() != c.base || got.Width != c.width || term.Cursor().X != c.width {
			t.Fatalf("fullcluster overflow%q: cell%+v cursor%+v", c.overflow, got, term.Cursor())
		}
		var binary Terminal
		if err := binary.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
			t.Fatal(err)
		}
		binary.Compact()
		assertSnapshotState(t, term, &binary)
		painted := New(8, 3)
		painted.Write(term.Repaint(false))
		if painted.Cell(0, 0) != term.Cell(0, 0) || painted.Cursor() != term.Cursor() {
			t.Fatalf("fullcluster geometry repaint: cell%+v/%+v cursor%+v/%+v", painted.Cell(0, 0), term.Cell(0, 0), painted.Cursor(), term.Cursor())
		}
		for _, suffix := range []string{"\u0302", "\x1b[?2027lX"} {
			term.Write([]byte(suffix))
			binary.Write([]byte(suffix))
			painted.Write([]byte(suffix))
			assertSnapshotState(t, term, &binary)
			for y := range 3 {
				for x := range 8 {
					if term.Cell(x, y) != painted.Cell(x, y) {
						t.Fatalf("fullcluster continued cell%d,%d changed", x, y)
					}
				}
			}
		}
	}
}

func TestNativeClusterCapacity(t *testing.T) {
	for _, mode := range []string{"", "\x1b[?2027h"} {
		for _, mark := range []string{"\u0301", "\U000e0100"} {
			term := New(8, 3, WithScrollback(20))
			term.Write([]byte(mode + "A" + strings.Repeat(mark, 64)))
			want := "A" + strings.Repeat(mark, 64)
			if got := term.Cell(0, 0).Text(); got != want {
				t.Fatalf("capacity mode%q mark%q: %d codepoints, want65", mode, mark, utf8.RuneCountInString(got))
			}
			term.ClearDirty()
			term.Write([]byte(mark))
			if term.Cell(0, 0).Text() != want || term.RowDirty(0) {
				t.Fatal("overflow changed a full cluster")
			}
			var restored Terminal
			if err := restored.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
				t.Fatal(err)
			}
			assertSnapshotState(t, term, &restored)
			for _, next := range []string{"\x1b[?2027l\u0302", "\r\nnext\r\nlast\r\nmore"} {
				term.Write([]byte(next))
				restored.Write([]byte(next))
				assertSnapshotState(t, term, &restored)
			}
			term.Resize(4, 8)
			restored.Resize(4, 8)
			restored.Compact()
			assertSnapshotState(t, term, &restored)
		}
	}
}

func TestLongClusterMixedJoinModes(t *testing.T) {
	term := New(12, 3, WithScrollback(20))
	term.Write([]byte("界"))
	for i := range 64 {
		if i%2 == 0 {
			term.Write([]byte("\x1b[?2027h"))
		} else {
			term.Write([]byte("\x1b[?2027l"))
		}
		term.Write([]byte("\U000e0100"))
	}
	c := term.x.clusters[term.cellAt(0, 0).R-clusterBase]
	for i := 1; i < 65; i++ {
		if c.joinModes.has(i) != (i%2 == 1) {
			t.Fatalf("join mode%d lost", i)
		}
	}
	for _, restore := range []func(*Terminal){
		func(dst *Terminal) {
			if err := dst.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
				t.Fatal(err)
			}
		},
		func(dst *Terminal) { dst.Write(term.Repaint(false)) },
	} {
		dst := New(12, 3, WithScrollback(20))
		restore(dst)
		if dst.Cell(0, 0) != term.Cell(0, 0) {
			t.Fatal("long mixed-mode cluster changed on replay")
		}
		dst.Write([]byte("\x1b[?2027h\u0303"))
		if dst.Cell(0, 0) != term.Cell(0, 0) {
			t.Fatal("overflow append changed replayed cluster")
		}
	}
}

func TestInternAppendMatchesConcatenatedCluster(t *testing.T) {
	term, control := New(12, 3), New(12, 3)
	check := func(prefix string, next rune, width uint8, modes clusterModes) {
		t.Helper()
		want := cluster{text: prefix + string(next), width: width, joinModes: modes}
		got := term.internAppend(prefix, next, width, modes)
		expected := control.intern(want)
		if got != expected || !slices.Equal(term.x.clusters, control.x.clusters) {
			t.Fatalf("%q + %U width %d modes %v: appended lookup changed pool identity", prefix, next, width, modes)
		}
	}
	for round := range 4 {
		for i := range 48 {
			prefix := fmt.Sprintf("%02d日", i)
			next := []rune{'\u0301', '👩', '\u200d', '\ufe0f'}[i%4]
			var modes clusterModes
			modes.set(1 + i%3)
			check(prefix, next, uint8(1+i%2), modes)
			check(prefix, next, uint8(1+i%2), modes)
			// A matching text with different width or mode metadata is
			// a distinct cluster, even while it remains in the recent cache.
			if round%2 == 0 {
				check(prefix, next, uint8(2-i%2), modes)
				check(prefix, next, uint8(1+i%2), clusterModes{})
			}
		}
	}
	for _, prefix := range []string{"e", "👩\u200d", strings.Repeat("👩", maxClusterRunes-1)} {
		for _, next := range []rune{0x301, 0x200d, 0x1f469, 0xfe0e, 0xfe0f} {
			check(prefix, next, 2, clusterModes{})
		}
	}
}

func TestRepeatedCombiningTextWarmAllocations(t *testing.T) {
	for _, mode := range []string{"", "\x1b[?2027h"} {
		term := New(12, 3, WithScrollback(0))
		data := []byte(mode + strings.Repeat("e\u0301 ", 64) + "\r\n")
		term.Write(data)
		if n := testing.AllocsPerRun(20, func() { term.Write(data) }); n != 0 {
			t.Fatalf("mode %q: repeated combining text allocated %.1f times", mode, n)
		}
	}
}
