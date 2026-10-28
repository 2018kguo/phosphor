package phosphor

import (
	"bytes"
	"fmt"
	"testing"
	"unsafe"
)

func TestColumnModesGateResizeAndPreservePen(t *testing.T) {
	var replies bytes.Buffer
	term := New(5, 4, WithReply(&replies))
	term.Write([]byte("ABCDE\x1b[?3h\x1b[?3$p\x1b[?40$p"))
	if cols, _ := term.Size(); cols != 5 || term.Modes().Columns132 || term.Modes().AllowColumnSwitching || replies.String() != "\x1b[?3;2$y\x1b[?40;2$y" {
		t.Fatalf("disabled column switching: size=%d modes=%+v replies=%q", cols, term.Modes(), replies.String())
	}
	term.Write([]byte("\x1b[48;2;1;2;3m\x1b[?40h\x1b[?3h"))
	if cols, rows := term.Size(); cols != 132 || rows != 4 || !term.Modes().Columns132 || !term.Modes().AllowColumnSwitching || term.Cursor().X != 0 || term.Cursor().Y != 0 || term.Cursor().PendingWrap {
		t.Fatalf("132 columns: size=%d,%d modes=%+v cursor=%+v", cols, rows, term.Modes(), term.Cursor())
	}
	for y := 0; y < 4; y++ {
		for x := 0; x < 132; x++ {
			if c := term.Cell(x, y); c.Text() != "" || c.Style.Bg != RGBColor(1, 2, 3) {
				t.Fatalf("column erase did not use pen background at%d,%d: %+v", x, y, c)
			}
		}
	}
	term.Write([]byte("X\x1b[?3lY"))
	if cols, _ := term.Size(); cols != 80 || term.Modes().Columns132 || term.Cell(0, 0).Text() != "Y" || term.Cell(0, 0).Style.Bg != RGBColor(1, 2, 3) {
		t.Fatal("80-column reset lost the pen or failed to erase")
	}
	term.Resize(91, 4)
	term.Write([]byte("\x1b[?3h"))
	term.Resize(93, 4)
	if cols, _ := term.Size(); cols != 93 || !term.Modes().Columns132 {
		t.Fatal("host resize was incorrectly forced by DECCOLM")
	}
	term.Write([]byte("\x1b[?40l\x1b[?3l"))
	if cols, _ := term.Size(); cols != 93 || term.Modes().Columns132 || term.Modes().AllowColumnSwitching {
		t.Fatal("disabled mode3 was not ignored")
	}
	if unsafe.Sizeof(*term) > 560 {
		t.Fatal("column modes grew the idle Terminal")
	}
}

func TestColumnModesSameWidthKeepsMarginsAndTabs(t *testing.T) {
	term := New(80, 5)
	term.Write([]byte("\x1b[?40h\x1b[?69h\x1b[2;4r\x1b[3;6s\x1b[?6h\x1b[3g\x1b[?3l"))
	if term.top != 1 || term.bot != 3 || term.left != 2 || term.right != 5 || term.Cursor().X != 2 || term.Cursor().Y != 1 || term.tabs() == nil {
		t.Fatalf("same-width switch reset margins/tabs: cursor=%+v region=%d,%d,%d,%d", term.Cursor(), term.top, term.bot, term.left, term.right)
	}
	term.Write([]byte("\x1b[?3h"))
	if term.top != 0 || term.bot != 4 || term.left != 0 || term.right != 131 || term.Cursor().X != 0 || term.Cursor().Y != 0 || term.tabs() != nil {
		t.Fatal("changed-width switch did not reset margins/tabs")
	}
}

func TestColumnModesCompactResetAndRepaint(t *testing.T) {
	for _, input := range []string{
		"\x1b[?40h\x1b[?3h",
		"\x1b[?40h\x1b[?3h\x1b[?40l",
		"\x1b[?40h\x1b[?3h\x1b[?3s\x1b[?3l",
		"\x1b[?40h\x1b[?3s\x1b[?40s\x1b[?3h",
	} {
		term := New(80, 4)
		term.Write([]byte(input + "MARK"))
		before := term.Modes()
		term.Compact()
		if term.Modes() != before || term.x == nil {
			t.Fatal("Compact discarded column modes")
		}
		cols, rows := term.Size()
		restored := New(cols, rows)
		restored.Write(term.Repaint(false))
		if restored.Modes() != term.Modes() || restored.Cell(0, 0).Text() != "M" || restored.Cursor() != term.Cursor() {
			t.Fatalf("repaint %q lost column state or painted cells: %+v / %+v", input, term.Modes(), restored.Modes())
		}
		term.Write([]byte("\x1b[?3r\x1b[?40r"))
		restored.Write([]byte("\x1b[?3r\x1b[?40r"))
		if restored.Modes() != term.Modes() {
			t.Fatal("column saved-mode continuation differs")
		}
	}
	term := New(80, 4)
	term.Write([]byte("\x1b[?40h\x1b[?3h\x1b[!p"))
	if !term.Modes().Columns132 || !term.Modes().AllowColumnSwitching {
		t.Fatal("DECSTR forgot column modes")
	}
	term.Write([]byte("\x1bc"))
	if term.Modes().Columns132 || term.Modes().AllowColumnSwitching {
		t.Fatal("RIS retained column modes")
	}
}

func TestMouseModesKeepIndependentBits(t *testing.T) {
	var replies bytes.Buffer
	term := New(12, 3, WithReply(&replies))
	term.Write([]byte("\x1b[?9;1000;1002;1003;1005;1006;1015;1016h"))
	m := term.Modes()
	if m.Mouse != MouseAnyEvent || m.MouseTrackingModes != 15 || m.MouseEncoding != MouseEncodingSGRPixels || m.MouseEncodingModes != 15 {
		t.Fatal(m)
	}
	term.Write([]byte("\x1b[?9l\x1b[?1005l"))
	m = term.Modes()
	if m.Mouse != MouseNone || m.MouseTrackingModes != 14 || m.MouseEncoding != MouseEncodingDefault || m.MouseEncodingModes != 14 {
		t.Fatal(m)
	}
	for _, n := range []int{9, 1000, 1002, 1003, 1005, 1006, 1015, 1016} {
		fmt.Fprintf(term, "\x1b[?%d$p", n)
		status := 1
		if n == 9 || n == 1005 {
			status = 2
		}
		want := fmt.Sprintf("\x1b[?%d;%d$y", n, status)
		if got := replies.String(); got != want {
			t.Fatalf("mode %d: reply %q, want %q", n, got, want)
		}
		replies.Reset()
	}
	if unsafe.Sizeof(*term) > 560 {
		t.Fatalf("mouse bits grew the idle Terminal to %d bytes", unsafe.Sizeof(*term))
	}
}

func TestMouseSavedModesAndReset(t *testing.T) {
	term := New(12, 3)
	term.Write([]byte("\x1b[?9;1000;1005;1016h\x1b[?9;1000;1005;1016s\x1b[?9;1000;1005;1016l\x1b[?9;1000;1005;1016r"))
	want := term.Modes()
	if want.MouseTrackingModes != MouseTrackingX10|MouseTrackingNormal || want.Mouse != MouseNormal || want.MouseEncodingModes != MouseEncodingModeUTF8|MouseEncodingModeSGRPixels || want.MouseEncoding != MouseEncodingSGRPixels {
		t.Fatal(want)
	}
	term.Write([]byte("\x1b[!p"))
	if got := term.Modes(); got != want {
		t.Fatalf("DECSTR changed mouse modes: %+v", got)
	}
	term.Write([]byte("\x1bc"))
	if got := term.Modes(); got != (Modes{Autowrap: true}) {
		t.Fatalf("RIS kept mouse modes: %+v", got)
	}
}

func TestMouseRepaintEveryReachableSelection(t *testing.T) {
	for _, numbers := range [][4]int{mouseTrackingNumbers, mouseEncodingNumbers} {
		for bits := uint16(0); bits <= mouseModeBits; bits++ {
			for selected := uint16(0); selected <= 4; selected++ {
				state := bits | selected<<4
				if !validMouseModes(state) {
					continue
				}
				term := New(12, 3)
				for i, mode := range numbers {
					if bits&(1<<i) != 0 {
						fmt.Fprintf(term, "\x1b[?%dh", mode)
					}
				}
				if selected != 0 {
					fmt.Fprintf(term, "\x1b[?%dh", numbers[selected-1])
				} else if bits != 0 {
					for i, mode := range numbers {
						if bits&(1<<i) == 0 {
							fmt.Fprintf(term, "\x1b[?%dl", mode)
							break
						}
					}
				}
				if term.mouse != state && term.mouseEnc != state {
					t.Fatalf("setup failed for bits=%x selected=%d", bits, selected)
				}
				// Saved false/true bits may transiently select other modes.
				fmt.Fprintf(term, "\x1b[?%ds\x1b[?%ds", numbers[0], numbers[3])
				restored := New(12, 3)
				restored.Write(term.Repaint(false))
				if got, want := restored.Modes(), term.Modes(); got != want {
					t.Fatalf("bits=%x selected=%d: got %+v want %+v", bits, selected, got, want)
				}
				fmt.Fprintf(term, "\x1b[?%dr\x1b[?%dr", numbers[3], numbers[0])
				fmt.Fprintf(restored, "\x1b[?%dr\x1b[?%dr", numbers[3], numbers[0])
				if restored.Modes() != term.Modes() {
					t.Fatalf("saved-mode continuation differs for bits=%x selected=%d", bits, selected)
				}
			}
		}
	}
}

func TestResizeTabsResetOnlyOnWidthChange(t *testing.T) {
	term := New(20, 3)
	term.Write([]byte("\x1b[3g\x1b[5G\x1bH\r\t"))
	if term.Cursor().X != 4 {
		t.Fatal(term.Cursor())
	}
	term.Resize(20, 4)
	term.Write([]byte("\r\t"))
	if term.Cursor().X != 4 {
		t.Fatal("height-only resize lost custom stop", term.Cursor())
	}
	for _, cols := range []int{30, 10} {
		term.Resize(cols, 4)
		term.Write([]byte("\r\t"))
		if term.Cursor().X != 8 || term.tabs() != nil {
			t.Fatalf("width %d did not restore implicit defaults: %+v", cols, term.Cursor())
		}
	}
}

func TestResizeEndsSynchronizedUpdate(t *testing.T) {
	for _, size := range [][2]int{{8, 4}, {9, 4}, {8, 5}} {
		term := New(8, 4)
		term.Write([]byte("\x1b[?2026h"))
		term.Resize(size[0], size[1])
		if term.Modes().SyncUpdate {
			t.Fatalf("resize%v retained synchronized update", size)
		}
	}
}

func TestNativeFixedModeReports(t *testing.T) {
	for _, mode := range []int{1, 5, 7, 10, 11, 13, 14, 15, 16, 17, 18, 19} {
		var replies bytes.Buffer
		term := New(8, 4, WithReply(&replies))
		term.Write(fmt.Appendf(nil, "\x1b[%dh\x1b[%d$p\x1b[%dl\x1b[%d$p", mode, mode, mode, mode))
		want := fmt.Sprintf("\x1b[%d;4$y\x1b[%d;4$y", mode, mode)
		if replies.String() != want {
			t.Fatalf("ANSI%d: got%q want%q", mode, replies.String(), want)
		}
	}
	var replies bytes.Buffer
	term := New(8, 4, WithReply(&replies))
	term.Write([]byte("\x1b[?117h\x1b[?117$p\x1b[?117l\x1b[?117$p\x1bP+q536D756C78\x1b\\"))
	if want := "\x1b[?117;4$y\x1b[?117;4$y\x1bP1+r536D756C78=5C455B343A25703125646D\x1b\\"; replies.String() != want {
		t.Fatalf("fixed mode/capability replies got%q want%q", replies.String(), want)
	}
}

func TestSnapshotRarePrivateModes(t *testing.T) {
	term := New(80, 4)
	term.Write([]byte("\x1b[?4;40h\x1b[?3h\x1b[?3;4;40s\x1b[?40l"))
	term.Compact()
	var dst Terminal
	if err := dst.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
		t.Fatal(err)
	}
	assertSnapshotState(t, term, &dst)
	for _, next := range []string{"\x1b[!p", "\x1b[?40r\x1b[?3l", "\x1b[?3;4;40r", "\x1bc"} {
		term.Write([]byte(next))
		dst.Write([]byte(next))
		dst.Compact()
		assertSnapshotState(t, term, &dst)
	}
}

func TestCursorAppearance(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		shape       CursorShape
		blink       bool
		parameter   string
	}{
		{"default", "", CursorDefault, false, "2"},
		{"blink-default", "\x1b[?12h", CursorDefault, true, "1"},
		{"steady-bar", "\x1b[5 q\x1b[?12l", CursorBar, false, "6"},
		{"blink-underline", "\x1b[4 q\x1b[?12h", CursorUnderline, true, "3"},
		{"invalid-style", "\x1b[5 q\x1b[7 q\x1b[65536 q", CursorBar, true, "5"},
		{"saved-blink", "\x1b[?12h\x1b[?12s\x1b[?12l\x1b[?12r", CursorDefault, true, "1"},
		{"soft-reset", "\x1b[5 q\x1b[!p", CursorDefault, false, "2"},
		{"hard-reset", "\x1b[5 q\x1bc", CursorDefault, false, "2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var replies bytes.Buffer
			term := New(10, 3, WithReply(&replies))
			term.Write([]byte(tc.input))
			if cur := term.Cursor(); cur.Shape != tc.shape || cur.Blink != tc.blink {
				t.Fatalf("cursor %+v, want shape %v blink %v", cur, tc.shape, tc.blink)
			}
			term.Write([]byte("\x1bP$q q\x1b\\\x1b[?12$p"))
			mode := "2"
			if tc.blink {
				mode = "1"
			}
			if want := "\x1bP1$r" + tc.parameter + " q\x1b\\\x1b[?12;" + mode + "$y"; replies.String() != want {
				t.Fatalf("replies %q, want %q", replies.String(), want)
			}
			restored := New(10, 3)
			restored.Write(term.Repaint(false))
			if got := restored.Cursor(); got.Shape != tc.shape || got.Blink != tc.blink {
				t.Fatalf("repaint cursor %+v, want %+v", got, term.Cursor())
			}
		})
	}
}

func TestCursorLifetimeCheckpoint(t *testing.T) {
	for _, input := range []string{
		"\x1b]8;;https://old\aA\x1b7\x1b]8;;\a\x1b8B",
		"\x1b[2;2H\x1b7\x1b[3;3H\x1b[?1049h\x1b[?1049l\x1b8X",
		"\x1b]8;;https://main\aA\x1b[?47h\x1b]8;;https://alt\aB\x1b[?47lC",
		"\x1b*A\x1bN\x1b7#\x1b8#",
		"\x1b*A\x1bN\x1b7",
		"\x1b*A\x1bN\x1b[?1049h\x1b7",
		"\x1b[?47h\x1b[2;2H\x1b[?1049h\x1b[3;3H\x1b[?1049h",
		"\x1b[?1049hALT\x1b[?1049l",
	} {
		t.Run(fmt.Sprintf("%q", input), func(t *testing.T) {
			source := New(8, 4)
			source.Write([]byte(input))
			source.Compact()
			var snapshot Terminal
			if err := snapshot.RestoreSnapshot(source.Snapshot(), SnapshotLimits{}); err != nil {
				t.Fatal(err)
			}
			repaint := New(8, 4)
			repaint.Write(source.Repaint(true))
			for _, next := range []string{"", "\x1b8#", "\x1b[?47l\x1b8M", "\x1b[?47h\x1b8A", "\x1b[?1049l\x1b8P"} {
				for _, term := range []*Terminal{source, &snapshot, repaint} {
					term.Write([]byte(next))
				}
				assertSnapshotState(t, source, &snapshot)
				for y := range 4 {
					for x := range 8 {
						if source.Cell(x, y) != repaint.Cell(x, y) {
							t.Fatalf("after %q cell %d,%d: %+v / %+v", next, x, y, source.Cell(x, y), repaint.Cell(x, y))
						}
					}
				}
				if source.Cursor() != repaint.Cursor() {
					t.Fatalf("after %q cursor", next)
				}
			}
		})
	}
}

func TestIndependentScreenModesAndAllSavedModes(t *testing.T) {
	var replies bytes.Buffer
	term := New(8, 4, WithReply(&replies))
	term.Write([]byte("\x1b[?47h\x1b[?1049h\x1b[?1049l\x1b[?47$p\x1b[?1047$p\x1b[?1049$p"))
	if want := "\x1b[?47;1$y\x1b[?1047;2$y\x1b[?1049;2$y"; replies.String() != want {
		t.Fatalf("%q", replies.String())
	}
	for _, mode := range savedModeNumbers {
		if _, ok := term.modeValue(mode); !ok {
			t.Fatalf("unknown canonical saved mode %d", mode)
		}
		term.Write(fmt.Appendf(nil, "\x1b[?%ds", mode))
	}
	if len(term.x.savedModes) != len(savedModeNumbers) {
		t.Fatal("saved mode inventory truncated")
	}
	var restored Terminal
	if err := restored.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
		t.Fatal(err)
	}
	assertSnapshotState(t, term, &restored)
	if unsafe.Sizeof(cursor{}) != 40 || unsafe.Sizeof(Terminal{}) > 560 || unsafe.Sizeof(cell{}) != 16 {
		t.Fatal("cursor changes enlarged idle storage")
	}
}

func TestSnapshotRetainsResizedInactiveAlternate(t *testing.T) {
	source := New(8, 3)
	source.Write([]byte("\x1b[?47h\x1b[?1049h\x1b[41m\x1b[?1047l"))
	source.Resize(8, 5)
	var restored Terminal
	if err := restored.RestoreSnapshot(source.Snapshot(), SnapshotLimits{}); err != nil {
		t.Fatal(err)
	}
	for _, term := range []*Terminal{source, &restored} {
		term.Write([]byte("\x1b[?47h"))
	}
	assertSnapshotState(t, source, &restored)
	if source.Cell(0, 0).Style.Bg != IndexedColor(1) || source.Cell(0, 4).Style.Bg != 0 {
		t.Fatal("resize did not retain mixed old/new backgrounds")
	}
}

func TestSavedCursorStorageIsLazy(t *testing.T) {
	term := New(80, 24)
	term.Write([]byte("\x1b[32muser@host\x1b[0m$ "))
	term.Write([]byte("\x1b8\x1b[!p")) // restoring an unset slot needs no storage
	term.Resize(80, 30)
	term.Repaint(true)
	if term.saved != nil {
		t.Fatal("prompt, read, resize or unset restore allocated saved cursors")
	}

	term.Write([]byte("\x1b7"))
	storage := term.saved
	if storage == nil {
		t.Fatal("first save did not allocate its cursor storage")
	}
	if n := testing.AllocsPerRun(100, func() {
		term.saveCursor()
		term.restoreCursor()
	}); n != 0 {
		t.Fatalf("repeated save/restore allocated %g times", n)
	}
	term.Write([]byte("\x1b[?1049h\x1b7\x1b[?1049l\x1b[!p"))
	if term.saved != storage {
		t.Fatal("alternate-screen switch or soft reset replaced reusable cursor storage")
	}
	term.Write([]byte("\x1bc"))
	if term.saved != nil {
		t.Fatal("full reset kept saved cursor storage")
	}
}

func TestSavedCursorStylesAndLiveLinksSurviveCompaction(t *testing.T) {
	term := New(12, 5)
	term.Write([]byte("\x1b[2;3H\x1b[31m\x1b]8;;https://main\a\x1b7"))
	term.Write([]byte("\x1b[34m\x1b]8;;https://1049\a\x1b[?1049h"))
	term.Write([]byte("\x1b[3;4H\x1b[36m\x1b]8;;https://alternate\a\x1b7"))
	for i := range 100 {
		term.Write([]byte(fmt.Sprintf("\x1b]8;;https://unused/%d\a", i)))
	}
	term.Resize(14, 6)
	term.Write([]byte("\x1b8A"))
	if c := term.Cell(3, 2); c.Rune != 'A' || c.Link.URI != "https://unused/99" || c.Style.Fg != IndexedColor(6) {
		t.Fatalf("alternate DECSC after compaction: %+v", c)
	}
	term.Write([]byte("\x1b[?1049lB"))
	if c := term.Cell(2, 1); c.Rune != 'B' || c.Link.URI != "" || c.Style.Fg != IndexedColor(4) {
		t.Fatalf("1049 restore after compaction: %+v", c)
	}
	term.Write([]byte("\x1b8M"))
	if c := term.Cell(2, 1); c.Rune != 'M' || c.Link.URI != "" || c.Style.Fg != IndexedColor(4) {
		t.Fatalf("main DECSC after compaction: %+v", c)
	}
	checkRepaint(t, term)
}

func TestSoftResetKeepsAlternateRestore(t *testing.T) {
	term := New(10, 4)
	term.Write([]byte("\x1b[2;3H\x1b[31m\x1b[?1049h\x1b7\x1b[!p\x1b[?1049lX"))
	if c := term.Cell(2, 1); c.Rune != 'X' || c.Style.Fg != IndexedColor(1) {
		t.Fatalf("soft reset lost the main cursor saved by 1049: %+v", c)
	}
}
