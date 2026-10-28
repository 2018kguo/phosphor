package phosphor

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A repaint taken at any byte boundary must leave the next bytes in the
// same parser, including queries and controls which already took effect.
func TestRepaintContinuation(t *testing.T) {
	cases := []struct{ name, stream string }{
		{"csi-style", "\x1b[38:2::10:20:30mX\x1b[0mY"},
		{"csi-style-missing-final", "\x1b[58;5;9m\x1b[58:5:mX\x1b[58;mY\x1b[58:2:1:2mZ"},
		{"csi-mode", "\x1b[?2004hX"},
		{"csi-intermediate", "\x1b[4 qX"},
		{"csi-control", "abc\x1b[3\b\a1mX"},
		{"csi-ignore", "\x1b[?1?2004hX"},
		{"csi-overflow", "\x1b[" + strings.Repeat("1;", maxParams+8) + "31mX"},
		{"csi-saturated", "\x1b[999999999999999999999mX"},
		{"csi-cancel", "\x1b[3\x18X"},
		{"escape-intermediate", "\x1b(0lqk\x1b(BX"},
		{"escape-last-intermediate", "\x1b()0\x0elqk\x0fX"},
		{"osc-title", "\x1b]2;hello world\aX"},
		{"osc-hyperlink", "\x1b]8;id=a;https://example.com\x1b\\L\x1b]8;;\x1b\\X"},
		{"osc-overflow", "\x1b]2;" + strings.Repeat("x", maxOSC) + "\aX"},
		{"osc-escape-abort", "\x1b]2;ignored\x1b[31mX"},
		{"dcs-query", "\x1b[1;31m\x1bP$qm\x1b\\X"},
		{"dcs-cap", "\x1bP$q" + strings.Repeat("x", 300) + "\x1b\\X"},
		{"dcs-ignored", "\x1bPqignored\a\x1b\\X"},
		{"sos", "\x1bXignored\a\x1b\\X"},
		{"pm", "\x1b^ignored\x1b\\X"},
		{"apc", "\x1b_ignored\x1b\\X"},
		{"string-escape-abort", "\x1b_ignored\x1b[31mX"},
		{"utf8", "😀X"},
		{"utf8-truncated", "\xf0\x9f\x1b[31mX"},
		{"utf8-c1", "\xc2\x9b31mX"},
		{"synchronized-update", "\x1b[?2026hX\x1b[?2026lY"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := []byte(c.stream)
			for cut := 0; cut <= len(in); cut++ {
				var originalReply, restoredReply bytes.Buffer
				originalBells, restoredBells := 0, 0
				original := New(12, 5, WithReply(&originalReply), WithBell(func() { originalBells++ }))
				restored := New(12, 5, WithReply(&restoredReply), WithBell(func() { restoredBells++ }))
				original.Write(in[:cut])
				restored.Write(original.Repaint(true))
				if restoredReply.Len() != 0 || restoredBells != 0 {
					t.Fatalf("cut %d: repaint repeated callbacks: reply %q, bells %d", cut, restoredReply.Bytes(), restoredBells)
				}
				assertContinuationState(t, original, restored, fmt.Sprintf("cut %d", cut))
				// Repeated checkpoints must retain the same continuation.
				another := New(12, 5)
				another.Write(restored.Repaint(true))
				assertContinuationState(t, original, another, fmt.Sprintf("cut %d then second repaint", cut))
				originalReply.Reset()
				originalBells = 0
				original.Write(in[cut:])
				restored.Write(in[cut:])
				assertContinuationState(t, original, restored, fmt.Sprintf("cut %d then suffix", cut))
				if !bytes.Equal(originalReply.Bytes(), restoredReply.Bytes()) || originalBells != restoredBells {
					t.Fatalf("cut %d: continuation callbacks differ: replies %q / %q; bells %d / %d", cut, originalReply.Bytes(), restoredReply.Bytes(), originalBells, restoredBells)
				}
			}
		})
	}
}

func assertContinuationState(t *testing.T, original, restored *Terminal, context string) {
	t.Helper()
	if got, want := state(restored), state(original); got != want {
		t.Fatalf("%s: %s", context, lineDiff(got, want))
	}
	if got, want := restored.Modes(), original.Modes(); got != want {
		t.Fatalf("%s: modes %+v, want %+v", context, got, want)
	}
	if got, want := restored.appendContinuation(nil), original.appendContinuation(nil); !bytes.Equal(got, want) {
		t.Fatalf("%s: parser continuation %q, want %q", context, got, want)
	}
}

func TestRepaintRecordingContinuation(t *testing.T) {
	paths, err := filepath.Glob("testdata/*.raw")
	if err != nil || len(paths) == 0 {
		t.Fatalf("recordings: %v (%d files)", err, len(paths))
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			in, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			// Use the original recording geometry and every byte boundary,
			// including those inside CSI, string and UTF-8 sequences.
			cuts := 0
			for cut := 0; cut <= len(in); cut++ {
				original := New(100, 30)
				original.Write(in[:cut])
				restored := New(100, 30)
				restored.Write(original.Repaint(true))
				// Continue just far enough to exercise this sequence before
				// a later full-screen redraw could hide a bad continuation.
				end := min(cut+32, len(in))
				original.Write(in[cut:end])
				restored.Write(in[cut:end])
				if got, want := restored.Repaint(true), original.Repaint(true); !bytes.Equal(got, want) {
					assertContinuationState(t, original, restored, fmt.Sprintf("cut %d then %d bytes", cut, end-cut))
				}
				cuts++
			}
			t.Logf("checked all %d boundaries across %d bytes", cuts, len(in))
		})
	}
}

// The parser model is bounded even when hostile input never terminates a
// string or supplies arbitrarily many CSI parameters.
func TestContinuationBound(t *testing.T) {
	for _, in := range []string{
		"\x1b]2;" + strings.Repeat("x", 1<<20),
		"\x1bP$q" + strings.Repeat("x", 1<<20),
		"\x1b[" + strings.Repeat("99999999999;", 1<<16),
	} {
		term := New(12, 5)
		term.Write([]byte(in))
		if n := len(term.appendContinuation(nil)); n > maxOSC+4 {
			t.Fatalf("continuation has %d bytes for %d bytes of input", n, len(in))
		}
		if cap(term.osc) > maxOSC*2 {
			t.Fatalf("string buffer grew to %d", cap(term.osc))
		}
	}
}

func FuzzRepaintContinuation(f *testing.F) {
	for _, in := range []string{
		"\x1b[38:2::1:2:3mX\x1b[?2004hY",
		"\x1b[58;5;9m\x1b[58:5:mX\x1b[58;mY\x1b[58:2:1:2mZ",
		"\x1b[?" + strings.Repeat("999;", maxParams-2) + "999:25lX",
		"\x1b[" + strings.Repeat("0;", maxParams-1) + "31;32mX",
		"\x1b[38:2::65536:65535:999999999999999999999mX",
		"\x1b]8;id=a;https://example.com\x1b\\X",
		"\x1bP$qm\x1b\\X",
		"\x1b_ignored\x1b\\X",
		"abc\x1b[3\b1mX",
		"😀\x1b(0lqk\x1b(B",
	} {
		f.Add([]byte(in), uint16(len(in)/2))
	}
	f.Fuzz(func(t *testing.T, in []byte, at uint16) {
		if len(in) > 4096 {
			in = in[:4096]
		}
		cut := int(at) % (len(in) + 1)
		original := New(12, 5)
		original.Write(in[:cut])
		restored := New(12, 5)
		restored.Write(original.Repaint(true))
		assertContinuationState(t, original, restored, "before suffix")
		original.Write(in[cut:])
		restored.Write(in[cut:])
		assertContinuationState(t, original, restored, "after suffix")
	})
}

// A final MAIN with only 1047 reset must leave ALT through an erasing
// transition. It cannot retain uniformly erased old rows plus default rows
// introduced by height or width growth. The allowed shape is one top-left
// erased rectangle with zero padding. Text, protection, other styles, and row flags
// are excluded; those still require repaint equality. Width-only growth at 80/132 remains strict: DECCOLM may stage
// those widths. Native Terminal.zig deccolm only reaches 80/132 and cannot
// restore other host widths; switchScreenMode 1047 erases the entire ALT.
// The raw native fixture cursor-1047-resize-background-limit records
// the ANSI failure. FuzzSnapshotContinuation is the universal exact checkpoint
// oracle, including states outside portable ANSI's reconstruction contract.
func repaintRectangleGrowthLimit(src *Terminal) bool {
	if src.altActive || src.privateModes()&(privateMode47|privateMode1047|privateMode1049) != privateMode47|privateMode1049 || len(src.alt) != src.rows || src.rows < 1 {
		return false
	}
	bg := rowCell(src.alt[0], 0).Bg
	if bg == 0 {
		return false
	}
	width := 0
	for width < src.cols && rowCell(src.alt[0], width) == (cell{Bg: bg}) {
		width++
	}
	if width == 0 {
		return false
	}
	height := 0
	for height < src.rows && rowCell(src.alt[height], 0) == (cell{Bg: bg}) {
		height++
	}
	if width == src.cols && height == src.rows {
		return false
	}
	if height == src.rows && (src.cols == 80 || src.cols == 132) {
		return false
	}
	for y, row := range src.alt {
		if rowFlagsAt(src.altWrap, y) != 0 {
			return false
		}
		for x := range src.cols {
			want := cell{}
			if x < width && y < height {
				want.Bg = bg
			}
			if rowCell(row, x) != want {
				return false
			}
		}
	}
	return true
}

// Portable repaint's declared loss is only the default background in zero
// ALT padding. Keep every other assertion active against this explicit export.
func repaintRectangleGrowthExpectation(t *testing.T, src *Terminal) *Terminal {
	t.Helper()
	expected := New(1, 1)
	if err := expected.RestoreSnapshot(src.Snapshot(), SnapshotLimits{}); err != nil {
		t.Fatal(err)
	}
	bg := rowCell(src.alt[0], 0).Bg
	for y, row := range expected.alt {
		full := make([]cell, src.cols)
		copy(full, row)
		for x := range full {
			if full[x] == (cell{}) {
				full[x].Bg = bg
			}
		}
		expected.alt[y] = full
	}
	return expected
}

func checkRectangleGrowthSnapshot(t *testing.T, src *Terminal) {
	t.Helper()
	before := src.Snapshot()
	defer func() {
		if err := src.RestoreSnapshot(before, SnapshotLimits{}); err != nil {
			t.Fatal(err)
		}
	}()
	a, b := src, New(1, 1)
	if err := b.RestoreSnapshot(before, SnapshotLimits{}); err != nil {
		t.Fatal(err)
	}
	assertSnapshotState(t, a, b)
	for _, suffix := range []string{"\x1b[?47h", "\x1b8X\r\n\x1b[31mY", "\x1b[?1049l\x1b8Z"} {
		a.Write([]byte(suffix))
		b.Write([]byte(suffix))
		assertSnapshotState(t, a, b)
	}
	cols, rows := src.cols+1, src.rows+1
	a.Resize(cols, rows)
	b.Resize(cols, rows)
	a.Write([]byte("\x1b[?47h\x1b8Q"))
	b.Write([]byte("\x1b[?47h\x1b8Q"))
	assertSnapshotState(t, a, b)
}

func TestRepaintHeightGrowthLimit(t *testing.T) {
	source := New(3, 11, WithScrollback(20))
	source.Write([]byte("\x1b[?1049h\x1b[?47h\x1b[48;2;1;2;3m\x1b[?1047l"))
	source.Resize(3, 12)
	if !repaintRectangleGrowthLimit(source) {
		t.Fatal("height growth not recognized")
	}
	checkRectangleGrowthSnapshot(t, source)
	replay := New(3, 12)
	replay.Write(source.Repaint(false))
	if stateOf(source, false) == stateOf(replay, false) {
		t.Fatal("ANSI limit changed; reassess fuzz boundary")
	}
	checkRepaint(t, source)       // all facts except the declared ALT background loss
	source.Write([]byte("\x1bV")) // live protection does not protect existing ALT cells
	if !repaintRectangleGrowthLimit(source) {
		t.Fatal("live ISO protection changed rectangle classification")
	}
	checkRepaint(t, source)
	source.Write([]byte("\x1bW"))
	for name, change := range map[string]func(*Terminal){
		"active alternate":       func(s *Terminal) { s.Write([]byte("\x1b[?47h")) },
		"non-erasing exit":       func(s *Terminal) { s.Write([]byte("\x1b[?47l")) },
		"text":                   func(s *Terminal) { s.alt[0][0].R = 'X' },
		"protected cell":         func(s *Terminal) { s.alt[0][0].A = AttrProtected },
		"foreground":             func(s *Terminal) { s.alt[0][0].Fg = IndexedColor(1) },
		"row flags":              func(s *Terminal) { s.altWrap = make([]rowFlags, s.rows); s.altWrap[0] = rowWrap },
		"uneven row widths":      func(s *Terminal) { s.alt[0] = s.alt[0][:2] },
		"no new rows":            func(s *Terminal) { s.Resize(3, 11) },
		"colored row after zero": func(s *Terminal) { s.alt[5] = nil },
	} {
		t.Run(name, func(t *testing.T) {
			var s Terminal
			if err := s.RestoreSnapshot(source.Snapshot(), SnapshotLimits{}); err != nil {
				t.Fatal(err)
			}
			change(&s)
			if repaintRectangleGrowthLimit(&s) {
				t.Fatal("near miss excluded from strict repaint checks")
			}
		})
	}
}

func TestRepaintWidthGrowthLimit(t *testing.T) {
	for _, dims := range [][3]int{{13, 47, 8}, {80, 132, 8}, {13, 80, 8}, {1, 24, 1}} {
		source := New(dims[0], 12, WithScrollback(20))
		source.Write([]byte("\x1b[?47h\x1b[?1049h\x1b[48;2;1;2;3m\x1b[?1047l"))
		source.Resize(dims[1], dims[2])
		want := dims[1] != 80 && dims[1] != 132
		if repaintRectangleGrowthLimit(source) != want {
			t.Fatalf("width %d to%d boundary", dims[0], dims[1])
		}
		if want {
			checkRepaint(t, source)
		}
	}
}

func TestRepaintNarrowedClusterAtRightEdge(t *testing.T) {
	for _, tc := range []struct {
		prefix string
		cols   int
	}{
		{"A", 2}, {"AB", 3}, {"A界", 4},
		{"\x1b[32mA\x1b[0m", 2},
	} {
		for _, history := range []bool{false, true} {
			term := New(tc.cols+1, 4)
			term.Write([]byte("\x1b[?2027h" + tc.prefix + "\x1b[31;4m\x1b]8;;https://narrow\x1b\\☔︎\x1b]8;;\x1b\\\x1b[0m\x1b[K"))
			if history {
				term.Write([]byte("\r\n1\r\n2\r\n3\r\n4"))
			}
			term.Resize(tc.cols, 4)
			term.Write([]byte("\x1b[3;1H"))
			restored := New(tc.cols, 4)
			restored.Write(term.Repaint(true))
			assertSparseObservations(t, term, restored)
			for _, suffix := range []string{"Z", "\x1b8Q", "\r\nnext\r\nlast"} {
				term.Write([]byte(suffix))
				restored.Write([]byte(suffix))
				assertSparseObservations(t, term, restored)
			}
		}
	}
}

func TestRepaintNarrowedEdgeKeepsSoftWrap(t *testing.T) {
	term := New(3, 4)
	term.Write([]byte("\x1b[?2027hA☔︎Q"))
	term.Resize(2, 4)
	restored := New(2, 4)
	restored.Write(term.Repaint(true))
	assertSparseObservations(t, term, restored)
	for y := range term.rows {
		if rowFlagsAt(term.mainWrap, y) != rowFlagsAt(restored.mainWrap, y) {
			t.Fatalf("row %d soft wrap changed", y)
		}
	}
}

func TestRepaintNarrowedEdgeScratchLimits(t *testing.T) {
	for _, cols := range []int{1, 2} {
		term := New(4, 4)
		term.Write([]byte("\x1b[?2027h☔︎☔︎"))
		term.Resize(cols, 4)
		if term.canStageNarrowEdge(term.main[0]) {
			t.Fatal("narrowed clusters filling the row have no two-cell scratch area")
		}
	}
}

func TestRepaintNarrowedStyledTailBoundary(t *testing.T) {
	term := New(3, 4)
	term.Write([]byte("\x1b[?2027hA\x1b[31;4m\x1b]8;;https://tail\x1b\\☔︎\x1b]8;;\x1b\\\x1b[0m"))
	term.Resize(2, 4)
	term.Write([]byte("\x1b[3;1H"))
	want := term.Cell(0, 1)
	if want.Rune != 0 || want.Style.Fg != IndexedColor(1) || want.Link.URI != "https://tail" {
		t.Fatalf("fixture lost its styled empty tail: %+v", want)
	}
	replay := New(2, 4)
	replay.Write(term.Repaint(true))
	// ECH creates background-only empty cells. This relocated foreground,
	// underline and link still use the formatter's styled-space fallback.
	if got := replay.Cell(0, 1); got.Rune != ' ' || got.Style != want.Style || got.Link != want.Link {
		t.Fatalf("styled-empty VT boundary changed: %+v / %+v", got, want)
	}
	exact := New(2, 4)
	if err := exact.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
		t.Fatal(err)
	}
	assertSnapshotState(t, term, exact)
}

func TestRepaintOutsideOriginWithBlankAlternate(t *testing.T) {
	for _, prefix := range []string{
		"",
		"\x1b[?47h\x1b[3;4H\x1b[32m\x1b)A\x0e\x1b7\x1b[?47l",
		"\x1b[?6h\x1b[5;8H\x1b[34m\x1b(0\x1b7\x1b[?6l",
	} {
		for _, suffix := range []string{"", "q\x1b[B\x1b[C\x1b[41;1m\x1b[1\"q"} {
			term := New(10, 6)
			term.Write([]byte(prefix + "\x1b[0m\x1b(B\x0f\x1b[H\x1b[2JA\x1b[3;5r\x1b[?69h\x1b[4;8s\x1b[?6hB\x1b[22J" + suffix))
			if !term.canStageOriginCursor() {
				t.Fatalf("fixture did not reach an outside-origin cursor: %+v", term.cur)
			}
			restored := New(10, 6)
			restored.Write(term.Repaint(true))
			assertSparseObservations(t, term, restored)
			for _, continuation := range []string{"X", "\x1b8Z", "\x1b[?47h\x1b8A\x1b[?47l", "\x1b[?1047h\x1b8B\x1b[?1047l", "\x1b[?1049hQ\x1b[?1049lR"} {
				term.Write([]byte(continuation))
				restored.Write([]byte(continuation))
				assertSparseObservations(t, term, restored)
			}
		}
	}
}

func TestRepaintOutsideOriginKeepsScratchLimitsExplicit(t *testing.T) {
	term := New(10, 6)
	term.Write([]byte("\x1b[?47hkept\x1b[?47lA\x1b[3;5r\x1b[?69h\x1b[4;8s\x1b[?6hB\x1b[22J"))
	if term.canStageOriginCursor() {
		t.Fatal("nonempty alternate screen was treated as scratch")
	}
	term = New(10, 6)
	term.Write([]byte("\x1b[?6h\x1b[6;10H\x1b7\x1b[3;5r\x1b[?69h\x1b[4;8s\x1b8"))
	if term.canStageOriginCursor() {
		t.Fatal("scratch origin claimed an unreachable lower/right cursor")
	}
}

func TestRepaintSavedPendingWrap(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		cols        int
	}{
		{"overwritten-first-cell", "abcd\x1b7\rX", 4},
		{"current-also-pending", "abcd\x1b7", 4},
		{"erased-under-saved-cursor", "abcd\x1b7\x1b[2J", 4},
		{"wide-under-saved-cursor", "ab漢\x1b7\rX", 4},
		{"saved-pen-differs", "abcd\x1b[31m\x1b]8;;https://saved\x1b\\\x1b7\x1b[0m\x1b]8;;\x1b\\\rX", 4},
		{"saved-origin", "\x1b[?6habcd\x1b7\x1b[?6l\rX", 4},
		{"old-right-margin", "\x1b[?69h\x1b[1;2sab\x1b7\x1b[?69l\rX", 8},
		{"custom-tabs", "\x1b[3g\x1b[3G\x1bH\x1b[6G\x1bH\rabcdefgh\x1b[Z\x1b7\rX", 8},
		{"tab-to-zero", "abcdefgh\x1b[Z\x1b7\rX", 8},
		{"one-column", "a\x1b7\rX", 1},
		{"wrapped-row", "abcd\x1b7e\rX", 4},
		{"alternate-active", "main\x1b[?1049habcd\x1b7\rX", 4},
		{"alternate-kept", "\x1b[?47habcd\x1b7\rX\x1b[?47l", 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			term := New(tc.cols, 3)
			term.Write([]byte(tc.input))
			restored := New(tc.cols, 3)
			restored.Write(term.Repaint(true))
			assertSparseObservations(t, term, restored)
			assertRepaintSavedSlots(t, term, restored)
			for _, suffix := range []string{"\x1b8Z", "\x1b[?47h\x1b8Q\x1b[?47l", "\x1b[?1049lR", "\x1b8S"} {
				term.Write([]byte(suffix))
				restored.Write([]byte(suffix))
				assertSparseObservations(t, term, restored)
			}
		})
	}
}

func assertRepaintSavedSlots(t *testing.T, a, b *Terminal) {
	t.Helper()
	for i := range 2 {
		var left, right savedCursor
		if a.saved != nil {
			left = a.saved.screen[i]
		}
		if b.saved != nil {
			right = b.saved.screen[i]
		}
		if left.valid != right.valid || left.valid && !snapshotCursorsEqual(a, b, left.cursor, right.cursor) {
			t.Fatalf("saved cursor %d changed: %+v / %+v", i, left, right)
		}
	}
}

func TestSavedPendingRepaintKeepsExistingWrappedRow(t *testing.T) {
	for _, input := range []string{"abcd\x1b[?47h\x1b7X", "abcd\x1b[?47h\x1b7XY\x1b[41m\x1b[2;3H\x1b[X"} {
		source := New(4, 3, WithScrollback(20))
		source.Write([]byte(input))
		restored := New(4, 3, WithScrollback(20))
		restored.Write(source.Repaint(true))
		for _, next := range []string{"", "\x1b8Z", "\r\nnext"} {
			source.Write([]byte(next))
			restored.Write([]byte(next))
			if source.Cursor() != restored.Cursor() {
				t.Fatalf("input%q next%q cursor%+v/%+v", input, next, source.Cursor(), restored.Cursor())
			}
			for y := range 3 {
				if source.Wrapped(y) != restored.Wrapped(y) {
					t.Fatalf("input%q next%q row%d lostwrap", input, next, y)
				}
				for x := range 4 {
					if source.Cell(x, y) != restored.Cell(x, y) {
						t.Fatalf("input%q next%q cell%d,%d changed", input, next, x, y)
					}
				}
			}
		}
	}
}

// The CI token crasher enters 47, leaves with 1049, then resizes. Resize
// releases the empty inactive ALT; repaint must visit ALT to set mode 47.
func TestRepaintImplicitAlternateContinuation(t *testing.T) {
	for _, suffix := range []string{"", "\x1b8X", "\x1b[?47h", "\x1b[?47h\x1b8X", "\x1b[?1047h\x1b8X", "\x1b[?1049h\x1b8X\x1b[?1049l\x1b8Y"} {
		t.Run(fmt.Sprintf("%q", suffix), func(t *testing.T) {
			source := New(3, 1, WithScrollback(20))
			source.Write([]byte("\x1b[?47h\x1b[?1049l"))
			source.Resize(15, 7)
			replay := New(15, 7, WithScrollback(20))
			replay.Write(source.Repaint(false))
			if source.alt != nil || replay.alt == nil {
				t.Fatal("expected implicit source ALT and allocated replay ALT")
			}
			check := func() {
				t.Helper()
				if source.Modes() != replay.Modes() || source.Cursor() != replay.Cursor() || !snapshotCursorsEqual(source, replay, source.cur, replay.cur) {
					t.Fatal("modes or current cursor differ")
				}
				for i := range 2 {
					var a, b savedCursor
					if source.saved != nil {
						a = source.saved.screen[i]
					}
					if replay.saved != nil {
						b = replay.saved.screen[i]
					}
					if a.valid != b.valid || !snapshotCursorsEqual(source, replay, a.cursor, b.cursor) {
						t.Fatalf("saved slot %d differs", i)
					}
				}
				for y := range source.rows {
					if source.Wrapped(y) != replay.Wrapped(y) {
						t.Fatalf("row %d wrap differs", y)
					}
					for x := range source.cols {
						if source.Cell(x, y) != replay.Cell(x, y) {
							t.Fatalf("cell %d,%d differs", x, y)
						}
					}
				}
			}
			check()
			source.Write([]byte(suffix))
			replay.Write([]byte(suffix))
			check()
			source.Resize(9, 4)
			replay.Resize(9, 4)
			source.Write([]byte("\x1b[?47h\x1b8A\x1b[?47l\x1b8B"))
			replay.Write([]byte("\x1b[?47h\x1b8A\x1b[?47l\x1b8B"))
			check()
		})
	}
}
