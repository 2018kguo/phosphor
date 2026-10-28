package phosphor

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"
	"unsafe"
)

func TestSemanticCommandsAndCursorSave(t *testing.T) {
	term := feed(8, 4, "\x1b]133;P\a$ \x1b]133;B\a")
	if term.Prompt(0) != PromptPrimary || term.semanticContent() != semanticInput {
		t.Fatal("prompt/input marker not retained")
	}
	term.Write([]byte("\x1b7\x1b]133;C\a\x1b8\r\nX"))
	if term.semanticContent() != semanticOutput || term.Prompt(1) != PromptNone {
		t.Fatal("DECRC restored semantic cursor state")
	}
	term.Write([]byte("\x1b]133;P;k=s\a"))
	if term.Prompt(1) != PromptContinuation {
		t.Fatal("secondary prompt marker lost")
	}
	term.Write([]byte("\r\x1b]133;C\a"))
	if term.Prompt(1) != PromptNone || term.Prompt(0) != PromptPrimary {
		t.Fatal("column-zero output marker removed the wrong row annotation")
	}
	for _, command := range []string{"B", "I"} {
		e := feed(4, 3, "\x1b]133;"+command+"\aX\r\n")
		want := PromptContinuation
		if command == "I" {
			want = PromptNone
		}
		if e.Prompt(1) != want {
			t.Fatalf("%s newline prompt kind %d want %d", command, e.Prompt(1), want)
		}
	}
}

func TestSemanticStorageUsesExistingPadding(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) == 8 && unsafe.Sizeof(Terminal{}) != 560 {
		t.Fatalf("Terminal size=%d, want560", unsafe.Sizeof(Terminal{}))
	}
	if unsafe.Sizeof(cell{}) != 16 {
		t.Fatalf("cell size=%d, want16", unsafe.Sizeof(cell{}))
	}
	term := feed(80, 3, "warm")
	if n := testing.AllocsPerRun(100, func() { term.Write([]byte("\rplain ascii\r")) }); n != 0 {
		t.Fatalf("ASCII allocated %v times", n)
	}
	if n := testing.AllocsPerRun(100, func() { term.Write([]byte("\x1b]133;B\a\x1b]133;D\a")) }); n != 0 {
		t.Fatalf("input-only semantic modes allocated %v times", n)
	}
	if term.x != nil || term.Prompt(-1) != PromptNone || term.Prompt(3) != PromptNone {
		t.Fatal("input-only semantic state allocated extra storage or leaked an invalid row")
	}
}

func TestSemanticSnapshotPreservesExactEnumsAndRowFlags(t *testing.T) {
	for _, input := range []string{
		"\x1b]133;P\a\x1b[2J",
		"\x1b]133;A;redraw=1\ahello\x1b]133;B\a",
		"\x1b]133;A;redraw=last\ahello\x1b]133;I\a\x1b[?1049h\x1b]133;P;k=s\aALT",
		"\x1b]133;P\aMAIN\x1b[?47h\x1b]133;I\aALT\x1b[?47l",
	} {
		term := feed(8, 4, input)
		copy := New(8, 4)
		if err := copy.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
			t.Fatal(err)
		}
		if copy.semantic != term.semantic || copy.promptRedraw != term.promptRedraw {
			t.Fatalf("snapshot normalized semantic enum: input %q got %v/%d want %v/%d", input, copy.semantic, copy.promptRedraw, term.semantic, term.promptRedraw)
		}
		for y := 0; y < 4; y++ {
			if rowFlagsAt(copy.mainWrap, y) != rowFlagsAt(term.mainWrap, y) || rowFlagsAt(copy.altWrap, y) != rowFlagsAt(term.altWrap, y) {
				t.Fatal("snapshot changed row annotations")
			}
		}
		before := copy.semantic
		copy.Compact()
		if copy.semantic != before {
			t.Fatal("Compact changed semantic cursor state")
		}
	}
	for _, invalid := range []func(*Terminal){
		func(e *Terminal) { e.promptRedraw = 3 },
		func(e *Terminal) { e.setRowFlags(0, 6) },
	} {
		bad := New(8, 4)
		invalid(bad)
		target := feed(8, 4, "KEEP")
		before := target.Snapshot()
		if err := target.RestoreSnapshot(bad.Snapshot(), SnapshotLimits{}); err == nil || !bytes.Equal(before, target.Snapshot()) {
			t.Fatal("invalid semantic enum accepted or restore changed its receiver")
		}
	}
}

func TestSemanticHistoryMarkersSurvivePopAndSnapshot(t *testing.T) {
	term := feed(4, 2, "\x1b]133;P\aabcdefgh\r\nIJKL\r\n\x1b]133;D\a")
	if term.ScrollbackLen() == 0 {
		t.Fatal("missing prompt history")
	}
	for _, line := range term.sb[term.sbStart:] {
		if !validateHistoryLine(line) {
			t.Fatalf("invalid annotated history %q", line)
		}
	}
	copy := New(4, 2)
	if err := copy.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
		t.Fatal(err)
	}
	for _, e := range []*Terminal{term, copy} {
		e.Compact()
		e.Resize(4, 5)
		if e.Line(0) != "abcd" || e.Line(1) != "efgh" || e.Prompt(0) != PromptPrimary || e.Prompt(1) != PromptContinuation {
			t.Fatalf("history pop lost annotations: %q %v", e.Lines(), e.mainWrap)
		}
	}
	plain := feed(4, 2, "one\r\ntwo\r\nthree")
	for _, line := range plain.sb[plain.sbStart:] {
		if bytes.Contains(line, []byte{0, 0, 'P'}) {
			t.Fatal("ordinary output paid for semantic history markers")
		}
	}
}

func TestSemanticHistoryGrammarRejectsMalformedMarkers(t *testing.T) {
	for _, bad := range [][]byte{{0}, {0, 0}, {0, 0, 'P'}, {0, 0, 'P', 3}, {0, 0, 'X', 0}, {0, 0x80, 0, 'P', 1}} {
		if validateHistoryLine(bad) {
			t.Fatalf("accepted malformed row marker %x", bad)
		}
	}
	for _, kind := range []byte{0, 1, 2} {
		line := append([]byte{0, 0, 'P', kind}, 'A')
		if !validateHistoryLine(line) || stripSGR(line) != "A" || walkHistoryText(line, nil) {
			t.Fatalf("typed marker %d changed text or leaked into the v1 grammar", kind)
		}
	}
	for _, kind := range []byte{1, 2} {
		row := replayLine([]byte{0, 0, 'P', kind}, 8)
		if row.Prompt(0) != PromptKind(kind) || row.Cell(0, 0).Text() != "" || row.cur.y != 0 {
			t.Fatalf("empty row marker%d lost annotation or invented text", kind)
		}
	}
	lookalike := []byte("\x1b]8;;https://example/" + string([]byte{0, 0, 'P', 1}) + "\x1b\\A")
	called := false
	if !walkHistory(lookalike, nil, ignoreHistoryHoles, func(start, end int, flags rowFlags) { called = true }) || called {
		t.Fatal("NUL inside OSC payload was parsed as row metadata")
	}
}

func TestSemanticHeadlessDefaultsAndRedraw(t *testing.T) {
	for _, test := range []struct {
		prefix, option string
		cleared        bool
	}{
		{"", "", false}, {"", ";redraw=0", false}, {"", ";redraw=1", true},
		{"\x1bc", "", true}, {"\x1bc", ";redraw=0", false},
	} {
		term := feed(8, 4, test.prefix+"\x1b]133;A"+test.option+"\ahello")
		term.Resize(9, 4)
		if (term.Line(0) == "") != test.cleared {
			t.Fatalf("prefix=%q option=%q got %q", test.prefix, test.option, term.Line(0))
		}
	}
	// Pinned native returns the starting continuation when an all-
	// continuation chain reaches absolute history top, preserving row0.
	term := feed(3, 4, "\x1b]133;A;redraw=1\ahello")
	term.Resize(5, 4)
	if term.Line(0) != "hello" || term.Cursor().X != 0 || term.Cursor().Y != 1 {
		t.Fatal("native prompt-iterator boundary behavior changed", term.Lines(), term.Cursor())
	}
}

func TestSemanticEraseAndReplayWithoutRowMark(t *testing.T) {
	term := feed(8, 4, "\x1b]133;P\a\x1b[2J")
	if term.Prompt(0) != PromptNone || term.semanticContent() != semanticPrompt || term.ScrollbackLen() != 0 {
		t.Fatal("empty prompt ED2 must keep its cursor state without adding history")
	}
	copy := New(8, 4)
	copy.Write(term.Repaint(true))
	if copy.semanticContent() != semanticPrompt || copy.Prompt(0) != PromptNone || copy.ScrollbackLen() != 0 {
		t.Fatal("empty unmarked prompt replay did not preserve the exact semantic enum")
	}
	for _, e := range []*Terminal{term, copy} {
		e.Write([]byte("\r\nX"))
		if e.Prompt(1) != PromptContinuation || !strings.Contains(e.Line(1), "X") {
			t.Fatal("prompt continuation changed after ED2/replay")
		}
	}
	line := feed(8, 4, "\x1b]133;P\aTEXT\x1b[2K")
	if line.Prompt(0) != PromptPrimary {
		t.Fatal("whole-line erase removed the prompt annotation")
	}
}

func TestSoftResetRetainsPendingWrap(t *testing.T) {
	term := feed(4, 3, "abcd\x1b[!p")
	if !term.Cursor().PendingWrap {
		t.Fatal("DECSTR consumed pending wrap")
	}
	copy := New(4, 3)
	copy.Write(term.Repaint(false))
	for _, e := range []*Terminal{term, copy} {
		e.Write([]byte("X"))
		if e.Cursor().X != 1 || e.Cursor().Y != 1 || e.Line(0) != "abcd" || e.Line(1) != "X" {
			t.Fatal("DECSTR changed future wrapping", e.Lines(), e.Cursor())
		}
	}
	margin := feed(8, 4, "\x1b[?69h\x1b[2;7s\x1b[!p")
	if margin.lrmm || margin.left != 0 || margin.right != 7 {
		t.Fatal("DECSTR kept mode69 or margins")
	}
}

func TestHeightGrowDoesNotResurrectClearedHistory(t *testing.T) {
	term := feed(4, 2, "\x1b]133;P\aabcdefgh\r\nIJKL\r\n\x1b]133;D\a")
	term.Resize(4, 5)
	term.Write([]byte("\x1b[H\x1b[2J"))
	history := term.ScrollbackLines()
	term.Resize(4, 8)
	if strings.Join(term.Lines(), "") != "" || term.Cursor().Y != 0 || !slices.Equal(history, term.ScrollbackLines()) {
		t.Fatal("height grow pulled history despite non-bottom cursor", term.Lines(), term.Cursor())
	}
}

func TestInactivePrimaryCursorIsIndependentOfSavedSlots(t *testing.T) {
	for _, mode := range []int{47, 1047, 1049} {
		for _, row := range []int{1, 3} {
			t.Run(fmt.Sprintf("mode%d-row%d", mode, row), func(t *testing.T) {
				term := feed(4, 3, fmt.Sprintf("1111\r\n2222\r\n3333\r\n4444\x1b[2;1H\x1b7\x1b[%d;1H\x1b[?%dh\x1b[H", row, mode))
				if term.x.inactivePrimary.y != row-1 {
					t.Fatal("inactive primary cursor conflated with DECSC or active alternate")
				}
				term.Compact()
				copy := New(1, 1)
				if err := copy.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
					t.Fatal(err)
				}
				if copy.x.inactivePrimary != term.x.inactivePrimary {
					t.Fatal("snapshot changed actual inactive primary cursor")
				}
				for _, e := range []*Terminal{term, copy} {
					e.Resize(4, 5)
					e.Write([]byte(fmt.Sprintf("\x1b[?%dl", mode)))
					want := "2222"
					if row == 3 {
						want = "1111"
					}
					if e.Line(0) != want {
						t.Fatalf("height growth first row %q, want %q", e.Line(0), want)
					}
				}
			})
		}
	}
}

func TestUnflowedResizePreservesSemanticRowsAndPendingCursor(t *testing.T) {
	term := feed(4, 3, "\x1b]133;P\aabcd\x1b[?7l")
	term.Resize(5, 3)
	if term.Prompt(0) != PromptPrimary || term.Cursor().X != 4 || term.Cursor().PendingWrap {
		t.Fatal("cut-only resize lost prompt metadata or the next printed position", term.Cursor())
	}
	term.Write([]byte("\x1b[?7hX"))
	if term.Line(0) != "abcdX" {
		t.Fatal("resize overwrote the character before the old pending cursor")
	}
	term.Write([]byte("\x1b[?7l"))
	term.Resize(3, 3)
	if !term.Cursor().PendingWrap || term.Cursor().X != 2 || term.Prompt(0) != PromptPrimary {
		t.Fatal("cut-only shrink lost pending wrap at the new edge")
	}
}

func TestSemanticPromptFreshLine(t *testing.T) {
	for _, command := range []string{"L", "A", "N", "A;redraw=1", "N;aid=abc"} {
		for _, c := range []struct {
			prefix  string
			x, y    int
			history int
		}{
			{"", 0, 0, 0}, {"Hello", 0, 1, 0}, {"12345678", 0, 1, 0},
			{"\x1b[3;1Htext", 0, 2, 1},
			{"\x1b[?69h\x1b[3;7s\x1b[2;3H", 2, 1, 0},
			{"\x1b[?69h\x1b[3;7s\x1b[2;4H", 2, 2, 0},
			{"\x1b[?69h\x1b[3;7s\x1b[2;1H", 0, 1, 0},
			{"\x1b[?69h\x1b[3;7s\x1b[2;2H", 0, 2, 0},
		} {
			term := New(8, 3, WithScrollback(20))
			term.Write([]byte(c.prefix))
			input := []byte("\x1b]133;" + command + "\x1b\\")
			for _, b := range input {
				term.Write([]byte{b})
			}
			cur := term.Cursor()
			if cur.X != c.x || cur.Y != c.y || term.ScrollbackLen() != c.history {
				t.Fatalf("%q after%q: cursor%+v history%d", command, c.prefix, cur, term.ScrollbackLen())
			}
			var dst Terminal
			if err := dst.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
				t.Fatal(err)
			}
			dst.Compact()
			assertSnapshotState(t, term, &dst)
		}
	}
}

func TestSemanticPromptNonmovingMarkers(t *testing.T) {
	for _, cmd := range []string{"P", "B", "I", "C", "D", "Lextra", "Aextra", "Nextra", ""} {
		term := New(8, 3)
		term.Write([]byte("abc\x1b]133;" + cmd + "\a"))
		if cur := term.Cursor(); cur.X != 3 || cur.Y != 0 {
			t.Fatalf("%q moved cursor%+v", cmd, cur)
		}
	}
}
