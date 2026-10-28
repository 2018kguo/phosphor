package phosphor

import (
	"bytes"
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
