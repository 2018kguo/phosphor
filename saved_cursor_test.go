package phosphor

import (
	"fmt"
	"testing"
)

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
