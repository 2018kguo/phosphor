package phosphor

import (
	"bytes"
	"fmt"
	"runtime"
	"slices"
	"strings"
	"testing"
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
	term.mainWrap = make([]bool, term.rows)
	term.altWrap = make([]bool, term.rows)
	term.altWrap[1] = true
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
	if !term.altWrap[1] || term.Dirty() {
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
